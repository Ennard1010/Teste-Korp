package database

import (
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

var DB *sql.DB

// Connect inicializa a conexão com o PostgreSQL
func Connect() {
	host := getEnv("DB_HOST", "localhost")
	port := getEnv("DB_PORT", "5432")
	user := getEnv("DB_USER", "korp")
	password := getEnv("DB_PASSWORD", "korp123")
	dbname := getEnv("DB_NAME", "korp_estoque")

	connStr := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname,
	)

	var err error
	DB, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("Erro ao conectar ao banco: %v", err)
	}

	if err = DB.Ping(); err != nil {
		log.Fatalf("Banco não respondeu: %v", err)
	}

	log.Println("Conectado ao PostgreSQL (estoque)")
}

// Migrate cria as tabelas necessárias
func Migrate() {
	query := `
	CREATE TABLE IF NOT EXISTS produtos (
		id          SERIAL PRIMARY KEY,
		codigo      VARCHAR(50)  NOT NULL UNIQUE,
		descricao   VARCHAR(255) NOT NULL,
		saldo       INTEGER      NOT NULL DEFAULT 0 CHECK (saldo >= 0),
		created_at  TIMESTAMP    NOT NULL DEFAULT NOW(),
		updated_at  TIMESTAMP    NOT NULL DEFAULT NOW()
	);
	`

	if _, err := DB.Exec(query); err != nil {
		log.Fatalf("Erro na migração: %v", err)
	}

	log.Println("Migração do estoque concluída")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
