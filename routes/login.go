package routes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"

	"github.com/Rafhael-Viana/m/db"     // ajuste conforme o seu path real
	"github.com/Rafhael-Viana/m/models" // ajuste conforme o seu path real
)

type LoginResponse struct {
	Status string `json:"status"`
	Token  string `json:"token,omitempty"` // caso queira adicionar JWT depois
}

func (l *LoginResponse) setToken(token string) {
	l.Token = token
}

// geraToken cria um JWT com expiração
func geraToken(userID string, username string, role string) (string, error) {
	godotenv.Load()
	var jwtSecret = []byte(os.Getenv("JWT_SECRET")) // defina JWT_SECRET no ambiente

	claims := jwt.MapClaims{
		"user_id":  userID,
		"username": username,
		"role":     role,
		"exp":      time.Now().Add(time.Hour * 24).Unix(), // expira em 24h
		"iat":      time.Now().Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtSecret)
}

// Login realiza autenticação de um usuário
func Login(database *db.Database) http.HandlerFunc {
	// fmt.Println(jwtSecret)
	return func(w http.ResponseWriter, r *http.Request) {
		var u models.User
		defer r.Body.Close()

		dec := json.NewDecoder(r.Body)
		if err := dec.Decode(&u); err != nil {
			var syntaxErr *json.SyntaxError
			var unmarshalTypeErr *json.UnmarshalTypeError

			switch {
			case errors.Is(err, io.EOF):
				http.Error(w, "Empty body", http.StatusBadRequest)
			case errors.As(err, &syntaxErr):
				http.Error(w, "Malformed JSON", http.StatusBadRequest)
			case errors.As(err, &unmarshalTypeErr):
				http.Error(w, "Wrong type for a field", http.StatusBadRequest)
			default:
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
			}
			return
		}

		if u.Username == "" || u.Senha == "" {
			http.Error(w, "username and senha are required", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		// busca no banco o hash da senha do usuário
		var hashedPassword string
		var userID string

		query := `SELECT user_id,senha FROM "users" WHERE username = $1 LIMIT 1`
		err := database.Pool().QueryRow(ctx, query, u.Username).Scan(&userID, &hashedPassword)

		if errors.Is(err, pgx.ErrNoRows) {
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		} else if err != nil {
			http.Error(w, "database error", http.StatusInternalServerError)
			fmt.Println("DB Error:", err)
			return
		}

		// compara bcrypt
		err = bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(u.Senha))
		if err != nil {
			http.Error(w, "invalid username or password", http.StatusUnauthorized)
			return
		}

		// gera jwt token
		token, err := geraToken(userID, u.Username, u.Role)
		if err != nil {
			log.Println(err)
			return
		}

		// sucesso
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(LoginResponse{Status: "Logged", Token: token})

		fmt.Printf("Usuário logado: %s (ID %s)\n", u.Name, userID)
	}
}
