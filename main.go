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

	uploadDir := "./uploads"

	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Fatalf("Error creating upload dir: %v", err)
	}

	// Serve Files Route
	mux.Handle(
		"GET /uploads/",
		http.StripPrefix("/uploads/", http.FileServer(http.Dir(uploadDir))),
	)

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
	mux.HandleFunc("POST /api/users/{id}/upload", routes.UploadUserFile)

	// Rotas de CRUD Ponto Funcionário
	mux.Handle("POST /api/points", routes.CreatePoint(pool))
	mux.Handle("GET /api/points", routes.ListPoints(pool))
	mux.Handle("GET /api/points/{id}", routes.GetPoint(pool))       // /users/{id}
	mux.Handle("PATCH /api/points/{id}", routes.UpdatePoint(pool))  // /users/{id}
	mux.Handle("DELETE /api/points/{id}", routes.DeletePoint(pool)) // /users/{id}

	// Rotas de CRUD Setores
	mux.Handle("POST /api/setor", routes.CreateSetor(pool))
	mux.Handle("GET /api/setor", routes.ListSetores(pool))
	mux.Handle("GET /api/setor/{id}", routes.GetSetor(pool))       // /users/{id}
	mux.Handle("PATCH /api/setor/{id}", routes.UpdateSetor(pool))  // /users/{id}
	mux.Handle("DELETE /api/setor/{id}", routes.DeleteSetor(pool)) // /users/{id}

	// Relatórios
	mux.Handle("GET /api/reports/points", routes.ReportPoints(pool))
	mux.Handle("GET /api/reports/frequency", routes.ReportFrequency(pool))

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
