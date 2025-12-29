package routes

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Rafhael-Viana/m/db"
	"github.com/Rafhael-Viana/m/models"
	"github.com/google/uuid"
)

// --- CREATE ---
func CreateSetor(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var s models.Setor

		if err := json.NewDecoder(r.Body).Decode(&s); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if s.Nome == "" {
			http.Error(w, "nome is required", http.StatusBadRequest)
			return
		}

		s.Setor_ID = uuid.NewString()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		query := `
			INSERT INTO setores (setor_id, nome, quantidade, lider, created_by)
			VALUES ($1,$2,$3,$4,$5)
		`

		_, err := database.Pool().Exec(
			ctx,
			query,
			s.Setor_ID,
			s.Nome,
			s.Quantidade,
			s.Lider,
			s.CreatedBy,
		)

		if err != nil {
			http.Error(w, "could not create setor", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(s)
	}
}

func ListSetores(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := database.Pool().Query(ctx, `
			SELECT setor_id, nome, quantidade, lider, created_by, created_at
			FROM setores
			ORDER BY nome
		`)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var setores []models.Setor
		for rows.Next() {
			var s models.Setor
			if err := rows.Scan(
				&s.Setor_ID,
				&s.Nome,
				&s.Quantidade,
				&s.Lider,
				&s.CreatedBy,
				&s.CreatedAt,
			); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}
			setores = append(setores, s)
		}

		json.NewEncoder(w).Encode(setores)
	}
}

// --- GET SETOR + FUNCIONÁRIOS ---
func GetSetor(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setorID := r.PathValue("id")
		if setorID == "" {
			http.Error(w, "invalid setor id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		query := `
			SELECT
				s.setor_id,
				s.nome,
				u.user_id,
				u.name,
				sf.total_semana,
				sf.total_mes,
				sf.total_extra_mes,
				sf.faltas,
				sf.atestado
			FROM setores s
			LEFT JOIN setor_funcionarios sf ON sf.setor_id = s.setor_id
			LEFT JOIN users u ON u.user_id = sf.user_id
			WHERE s.setor_id = $1
			ORDER BY u.name
		`

		rows, err := database.Pool().Query(ctx, query, setorID)
		if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var response models.SetorResumo
		found := false

		for rows.Next() {
			found = true

			var (
				userID      *string
				userName    *string
				totalSemana *int
				totalMes    *int
				totalExtra  *int
				faltas      *int
				atestado    *int
			)

			if err := rows.Scan(
				&response.ID,
				&response.Setor,
				&userID,
				&userName,
				&totalSemana,
				&totalMes,
				&totalExtra,
				&faltas,
				&atestado,
			); err != nil {
				http.Error(w, "scan error", http.StatusInternalServerError)
				return
			}

			if userID != nil {
				response.Funcionarios = append(response.Funcionarios, models.FuncionarioResumo{
					ID:            *userID,
					Nome:          *userName,
					TotalSemana:   *totalSemana,
					TotalMes:      *totalMes,
					TotalExtraMes: *totalExtra,
					Faltas:        *faltas,
					Atestado:      *atestado,
				})
			}
		}

		if !found {
			http.Error(w, "setor not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(response)
	}
}

func UpdateSetor(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setorID := r.PathValue("id")

		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}

		fields := []string{}
		values := []any{}
		i := 1

		for key, value := range input {
			switch key {
			case "nome", "lider", "created_by":
				fields = append(fields, fmt.Sprintf("%s = $%d", key, i))
				values = append(values, value)
				i++
			case "quantidade":
				fields = append(fields, fmt.Sprintf("quantidade = $%d", i))
				values = append(values, int(value.(float64)))
				i++
			}
		}

		if len(fields) == 0 {
			http.Error(w, "no fields to update", http.StatusBadRequest)
			return
		}

		values = append(values, setorID)

		query := fmt.Sprintf(
			`UPDATE setores SET %s, updated_at = now() WHERE setor_id = $%d`,
			strings.Join(fields, ", "),
			len(values),
		)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(ctx, query, values...)
		if err != nil || cmd.RowsAffected() == 0 {
			http.Error(w, "setor not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

func DeleteSetor(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		setorID := r.PathValue("id")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(
			ctx,
			`DELETE FROM setores WHERE setor_id = $1`,
			setorID,
		)

		if err != nil {
			http.Error(w, "error deleting setor", http.StatusInternalServerError)
			return
		}

		if cmd.RowsAffected() == 0 {
			http.Error(w, "setor not found", http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}

