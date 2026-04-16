package main

import (
	"log"
	"os"

	"faturamento-service/database"
	"faturamento-service/handlers"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	// Conecta ao banco e roda migrações
	database.Connect()
	database.Migrate()

	// Configura o router
	r := gin.Default()

	// CORS para o frontend Angular
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "faturamento"})
	})

	// Rotas de notas fiscais
	r.POST("/notas", handlers.CriarNota)
	r.GET("/notas", handlers.ListarNotas)
	r.GET("/notas/:id", handlers.BuscarNota)
	r.POST("/notas/:id/imprimir", handlers.ImprimirNota)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
	}

	log.Printf("Serviço de Faturamento rodando na porta %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
