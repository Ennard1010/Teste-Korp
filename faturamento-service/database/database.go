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
	dbname := getEnv("DB_NAME", "korp_faturamento")

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

	log.Println("Conectado ao PostgreSQL (faturamento)")
}

// Migrate cria as tabelas necessárias
func Migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS notas_fiscais (
			id         SERIAL PRIMARY KEY,
			numero     INTEGER   NOT NULL UNIQUE,
			status     VARCHAR(10) NOT NULL DEFAULT 'Aberta' CHECK (status IN ('Aberta', 'Fechada')),
			created_at TIMESTAMP NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS nota_itens (
			id          SERIAL PRIMARY KEY,
			nota_id     INTEGER NOT NULL REFERENCES notas_fiscais(id) ON DELETE CASCADE,
			produto_id  INTEGER NOT NULL,
			quantidade  INTEGER NOT NULL CHECK (quantidade > 0)
		)`,
		// Sequence para numeração sequencial
		`CREATE SEQUENCE IF NOT EXISTS nota_numero_seq START 1`,
	}

	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("Erro na migração: %v", err)
		}
	}

	log.Println("Migração do faturamento concluída")
}

func getEnv(key, fallback string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return fallback
}
