package main

import (
	"log"
	"os"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"estoque-service/database"
	"estoque-service/handlers"
)

func main() {
	// Conecta ao banco e roda migrações
	database.Connect()
	database.Migrate()

	// Configura o router
	r := gin.Default()

	// CORS para o frontend Angular acessar
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:4200"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	// Health check (útil para docker-compose e para testar se o serviço está vivo)
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok", "service": "estoque"})
	})

	// Rotas de produtos
	r.POST("/produtos", handlers.CriarProduto)
	r.GET("/produtos", handlers.ListarProdutos)
	r.GET("/produtos/:id", handlers.BuscarProduto)
	r.PUT("/produtos/:id", handlers.AtualizarProduto)

	// Rota interna: chamada pelo serviço de faturamento
	r.POST("/produtos/:id/debitar", handlers.DebitarEstoque)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}

	log.Printf("Serviço de Estoque rodando na porta %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Erro ao iniciar servidor: %v", err)
	}
}
