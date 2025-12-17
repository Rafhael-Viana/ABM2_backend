package main

import (
	// "context"
	"fmt"
	"log"
	"net/http"
	"os"

	// mid "github.com/Rafhael-Viana/Gaart/middleware"
	"github.com/Rafhael-Viana/m/cors"
	"github.com/Rafhael-Viana/m/db"
	"github.com/Rafhael-Viana/m/routes"
	"github.com/joho/godotenv"
)

func main() {

	godotenv.Load()

	pool, err := db.NewPool()
	if err != nil {
		log.Fatalf("Error connecting database: %v", err)
	}

	// pool.Ping(context.Background())

	mux := http.NewServeMux()

	// Health Check Route
	mux.Handle("GET /api/hello", http.HandlerFunc(routes.Hello))

	// Rotas de Login
	mux.HandleFunc("POST /api/login", routes.Login(pool))

	// Rotas de CRUD usuários
	mux.Handle("POST /api/users", routes.CreateUser(pool))
	mux.Handle("GET /api/users", routes.ListUsers(pool))
	mux.Handle("GET /api/users/{id}", routes.GetUser(pool))
	mux.Handle("PATCH /api/users/{id}", routes.UpdateUser(pool))
	mux.Handle("DELETE /api/users/{id}", routes.DeleteUser(pool)) // /users/{id}

	// Rotas de CRUD Ponto Funcionário
	mux.Handle("POST /api/points", routes.CreatePoint(pool))
	mux.Handle("GET /api/points", routes.ListPoints(pool))
	mux.Handle("GET /api/points/", routes.GetPoint(pool))       // /users/{id}
	mux.Handle("PATCH /api/points/", routes.UpdatePoint(pool))  // /users/{id}
	mux.Handle("DELETE /api/points/", routes.DeletePoint(pool)) // /users/{id}

	// rotas permitidas
	allowedOrigins := []string{
		"http://localhost:3000", // dev
	}
	handler := cors.Cors(allowedOrigins, true /* usa cookies/credenciais? */)(mux)

	port := os.Getenv("PORT")
	// fmt.Println(port)
	fmt.Printf("Server listening on port %s...\n", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Error start server: %v", err)
	}
}
