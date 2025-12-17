package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/Rafhael-Viana/m/db"
	"github.com/Rafhael-Viana/m/models" // ajuste conforme o seu path real
)

// --- CREATE ---
func CreateUser(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var u models.User
		if err := json.NewDecoder(r.Body).Decode(&u); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		if u.Name == "" || u.Senha == "" || u.Username == "" || u.Email == "" {
			http.Error(w, "name, senha, email, username are required", http.StatusBadRequest)
			return
		}

		if u.Status == "" {
			u.Status = models.StatusActive
		}

		if !u.IsValidStatus(u.Status) {
			http.Error(w, "invalid status", http.StatusBadRequest)
			return
		}

		hashed, err := bcrypt.GenerateFromPassword([]byte(u.Senha), bcrypt.DefaultCost)
		if err != nil {
			http.Error(w, "error hashing password", http.StatusInternalServerError)
			return
		}

		u.User_ID = uuid.NewString()

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		query := `
			INSERT INTO users (
				name, senha, email, username, user_id,
				setor, cargo, nascimento, status, role
			)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
			RETURNING id
		`

		err = database.Pool().QueryRow(
			ctx,
			query,
			u.Name,
			string(hashed),
			u.Email,
			u.Username,
			u.User_ID,
			u.Setor,
			u.Cargo,
			u.Nascimento,
			u.Status,
			u.Role,
		).Scan(&u.ID)

		if err != nil {
			log.Println("DB error:", err)
			http.Error(w, "could not insert user", http.StatusInternalServerError)
			return
		}

		u.Senha = ""
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	}
}

// --- LIST ALL ---
func ListUsers(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := database.Pool().Query(ctx, `SELECT id, name FROM users ORDER BY id`)
		if err != nil {
			log.Println("DB error fetching users:", err) // log no servidor
			http.Error(w, "error fetching users: ", http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		var users []models.User
		for rows.Next() {
			var u models.User
			rows.Scan(&u.ID, &u.Name)
			users = append(users, u)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(users)
	}
}

// --- GET BY ID ---
func GetUser(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var u models.User
		query := `SELECT id, name FROM users WHERE id = $1`
		err = database.Pool().QueryRow(ctx, query, id).Scan(&u.ID, &u.Name)
		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		} else if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(u)
	}
}

// --- UPDATE (PATCH) ---
func UpdateUser(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extrair o ID da URL
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		// Parse do corpo da requisição
		var input map[string]any
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Campos e valores para o update
		fields := []string{}
		values := []any{}
		i := 1

		// Criar uma instância de User para validar o status
		var u models.User

		// Itera sobre os campos do input para verificar as mudanças
		for key, value := range input {
			switch key {
			case "name", "setor", "cargo", "role":
				fields = append(fields, fmt.Sprintf("%s = $%d", key, i))
				values = append(values, value)
				i++

			case "senha":
				hashed, _ := bcrypt.GenerateFromPassword([]byte(value.(string)), bcrypt.DefaultCost)
				fields = append(fields, fmt.Sprintf("senha = $%d", i))
				values = append(values, string(hashed))
				i++

			case "status":
				// Valida o status com o método da struct
				status := models.StatusUser(value.(string))
				u.Status = status
				if !u.IsValidStatus(status) {
					http.Error(w, "invalid status", http.StatusBadRequest)
					return
				}
				fields = append(fields, fmt.Sprintf("status = $%d", i))
				values = append(values, status)
				i++

			case "birth":
				fields = append(fields, fmt.Sprintf("nascimento = $%d", i))
				values = append(values, value)
				i++
			}
		}

		if len(fields) == 0 {
			http.Error(w, "no valid fields to update", http.StatusBadRequest)
			return
		}

		// Adicionar o ID como valor para o WHERE
		values = append(values, id)

		// Criar a query dinâmica
		query := fmt.Sprintf(
			`UPDATE users SET %s WHERE id = $%d`,
			strings.Join(fields, ", "),
			len(values),
		)

		// Executar a query
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(ctx, query, values...)
		if err != nil || cmd.RowsAffected() == 0 {
			http.Error(w, "user not found or not updated", http.StatusNotFound)
			return
		}

		// Retornar resposta de sucesso
		json.NewEncoder(w).Encode(map[string]string{"status": "updated"})
	}
}

// --- DELETE ---
func DeleteUser(database *db.Database) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.PathValue("id")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		cmd, err := database.Pool().Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
		if err != nil {
			http.Error(w, "error deleting user", http.StatusInternalServerError)
			return
		}

		if cmd.RowsAffected() == 0 {
			http.Error(w, "user not found", http.StatusNotFound)
			return
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
	}
}
