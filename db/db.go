package db

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"time"

	// "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
)

type Database struct {
	pool *pgxpool.Pool
}

func NewPool() (*Database, error) {

	godotenv.Load()

	user := os.Getenv("DB_USER")
	pass := os.Getenv("DB_PASS")
	dbname := os.Getenv("DB_NAME")
	host := os.Getenv("DB_HOST")
	port := os.Getenv("DB_PORT")

	// encode para evitar erro de URL
	user = url.QueryEscape(user)
	pass = url.QueryEscape(pass)
	host = url.QueryEscape(host)
	dbname = url.QueryEscape(dbname)

	log.Printf("Conectando em %s:%s/%s ...", host, port, dbname)

	// urlExample := "postgres://username:password@localhost:5432/database_name"
	url := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", user, pass, host, port, dbname)

	// db, err := sql.Open("mysql", nome+":"+senha+"@/"+dbname)
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect to database: %v\n", err)
		os.Exit(1)
	}

	// (opcional) ajustes de pool
	cfg.MaxConns = 10
	cfg.MinConns = 1
	cfg.MaxConnIdleTime = 5 * time.Minute

	fmt.Println("Banco conectado")

	pool, err := pgxpool.NewWithConfig(context.Background(), cfg)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	return &Database{pool: pool}, nil
}

func (d *Database) Pool() *pgxpool.Pool {
	return d.pool
}

func (d *Database) Close() {
	log.Printf("Encerrando conexão....")
	d.pool.Close()
}

func (d *Database) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}
