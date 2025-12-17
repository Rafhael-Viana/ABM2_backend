package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/Rafhael-Viana/m/db"
	"github.com/Rafhael-Viana/m/models" // ajuste conforme o seu path real
)

// --- CREATE ---
func CreatePoint(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		var input struct {
			UserID string `json:"user_id"`
		}

		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		if input.UserID == "" {
			http.Error(w, "user_id is required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// 1️⃣ Verifica se existe ponto aberto
		var pointID int
		err := database.Pool().QueryRow(ctx, `
			SELECT id FROM points
			WHERE user_id = $1 AND status = 'open'
		`, input.UserID).Scan(&pointID)

		now := time.Now()

		// 2️⃣ Se NÃO existir → cria clock_in
		if errors.Is(err, pgx.ErrNoRows) {

			query := `
				INSERT INTO points (user_id, clock_in, status)
				VALUES ($1, $2, 'open')
				RETURNING id
			`

			var id int
			err := database.Pool().QueryRow(ctx, query, input.UserID, now).Scan(&id)
			if err != nil {
				http.Error(w, "error creating point", http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"id":       id,
				"status":   "open",
				"clock_in": now,
			})
			return
		}

		// 3️⃣ Se EXISTIR → fecha com clock_out
		if err == nil {
			_, err := database.Pool().Exec(ctx, `
				UPDATE points
				SET clock_out = $1,
				    status = 'close',
				    updated_at = now()
				WHERE id = $2
			`, now, pointID)

			if err != nil {
				http.Error(w, "error closing point", http.StatusInternalServerError)
				return
			}

			json.NewEncoder(w).Encode(map[string]any{
				"id":        pointID,
				"status":    "close",
				"clock_out": now,
			})
			return
		}

		http.Error(w, "unexpected error", http.StatusInternalServerError)
	}
}

// --- LIST ALL ---
func ListPoints(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		rows, err := database.Pool().Query(context.Background(), `
			SELECT id, user_id, clock_in, clock_out, status, created_at, updated_at
			FROM points
			ORDER BY created_at DESC
		`)
		if err != nil {
			http.Error(w, "error fetching points", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var points []models.Point
		for rows.Next() {
			var p models.Point
			rows.Scan(
				&p.ID,
				&p.User_ID,
				&p.Clock_In,
				&p.Clock_Out,
				&p.Status,
				&p.CreatedAt,
				&p.UpdatedAt,
			)
			points = append(points, p)
		}

		json.NewEncoder(w).Encode(points)
	}
}

// --- GET BY ID ---
func GetPoint(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		var p models.Point

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		query := `
			SELECT id, user_id, clock_in, clock_out, status, created_at, updated_at
			FROM points
			WHERE id = $1
		`

		err = database.Pool().QueryRow(ctx, query, id).Scan(
			&p.ID,
			&p.User_ID,
			&p.Clock_In,
			&p.Clock_Out,
			&p.Status,
			&p.CreatedAt,
			&p.UpdatedAt,
		)

		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "point not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(p)
	}
}

// --- UPDATE (PATCH) ---
func UpdatePoint(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		fields := []string{}
		values := []any{}
		i := 1

		var clockIn, clockOut *time.Time
		var status models.StatusPoint

		for key, value := range input {
			switch key {

			case "clock_in":
				t, err := time.Parse(time.RFC3339, value.(string))
				if err != nil {
					http.Error(w, "invalid clock_in", http.StatusBadRequest)
					return
				}
				clockIn = &t
				fields = append(fields, fmt.Sprintf("clock_in = $%d", i))
				values = append(values, clockIn)
				i++

			case "clock_out":
				t, err := time.Parse(time.RFC3339, value.(string))
				if err != nil {
					http.Error(w, "invalid clock_out", http.StatusBadRequest)
					return
				}
				clockOut = &t
				fields = append(fields, fmt.Sprintf("clock_out = $%d", i))
				values = append(values, clockOut)
				i++

			case "status":
				status = models.StatusPoint(value.(string))
				if status != models.StatusOpen && status != models.StatusClosed {
					http.Error(w, "invalid status", http.StatusBadRequest)
					return
				}
				fields = append(fields, fmt.Sprintf("status = $%d", i))
				values = append(values, status)
				i++
			}
		}

		if len(fields) == 0 {
			http.Error(w, "no valid fields to update", http.StatusBadRequest)
			return
		}

		// Validação lógica
		if clockIn != nil && clockOut != nil && clockOut.Before(*clockIn) {
			http.Error(w, "clock_out cannot be before clock_in", http.StatusBadRequest)
			return
		}

		values = append(values, id)

		query := fmt.Sprintf(`
			UPDATE points
			SET %s, updated_at = now()
			WHERE id = $%d
		`, strings.Join(fields, ", "), len(values))

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(ctx, query, values...)
		if err != nil || cmd.RowsAffected() == 0 {
			http.Error(w, "point not found or not updated", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

// --- DELETE ---
func DeletePoint(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {

		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(ctx, `
			DELETE FROM points WHERE id = $1
		`, id)

		if err != nil {
			http.Error(w, "error deleting point", http.StatusInternalServerError)
			return
		}

		if cmd.RowsAffected() == 0 {
			http.Error(w, "point not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}
