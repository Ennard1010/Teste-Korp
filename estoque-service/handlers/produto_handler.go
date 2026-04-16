package handlers

import (
	"database/sql"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"estoque-service/database"
	"estoque-service/models"
)

// CriarProduto - POST /produtos
func CriarProduto(c *gin.Context) {
	var req models.ProdutoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Dados inválidos",
			Details: err.Error(),
		})
		return
	}

	var produto models.Produto
	err := database.DB.QueryRow(`
		INSERT INTO produtos (codigo, descricao, saldo)
		VALUES ($1, $2, $3)
		RETURNING id, codigo, descricao, saldo, created_at, updated_at
	`, req.Codigo, req.Descricao, req.Saldo).Scan(
		&produto.ID, &produto.Codigo, &produto.Descricao,
		&produto.Saldo, &produto.CreatedAt, &produto.UpdatedAt,
	)

	if err != nil {
		// Verifica se é violação de unique (código duplicado)
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error:   "Erro ao criar produto",
			Details: err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, produto)
}

// ListarProdutos - GET /produtos
func ListarProdutos(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, codigo, descricao, saldo, created_at, updated_at
		FROM produtos
		ORDER BY id
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: "Erro ao listar produtos",
		})
		return
	}
	defer rows.Close()

	produtos := []models.Produto{}
	for rows.Next() {
		var p models.Produto
		if err := rows.Scan(&p.ID, &p.Codigo, &p.Descricao, &p.Saldo, &p.CreatedAt, &p.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: "Erro ao ler produto",
			})
			return
		}
		produtos = append(produtos, p)
	}

	c.JSON(http.StatusOK, produtos)
}

// BuscarProduto - GET /produtos/:id
func BuscarProduto(c *gin.Context) {
	id := c.Param("id")

	var p models.Produto
	err := database.DB.QueryRow(`
		SELECT id, codigo, descricao, saldo, created_at, updated_at
		FROM produtos WHERE id = $1
	`, id).Scan(&p.ID, &p.Codigo, &p.Descricao, &p.Saldo, &p.CreatedAt, &p.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Produto não encontrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao buscar produto"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// AtualizarProduto - PUT /produtos/:id
func AtualizarProduto(c *gin.Context) {
	id := c.Param("id")
	var req models.ProdutoRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Dados inválidos",
			Details: err.Error(),
		})
		return
	}

	var p models.Produto
	err := database.DB.QueryRow(`
		UPDATE produtos
		SET codigo = $1, descricao = $2, saldo = $3, updated_at = NOW()
		WHERE id = $4
		RETURNING id, codigo, descricao, saldo, created_at, updated_at
	`, req.Codigo, req.Descricao, req.Saldo, id).Scan(
		&p.ID, &p.Codigo, &p.Descricao, &p.Saldo, &p.CreatedAt, &p.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Produto não encontrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao atualizar produto"})
		return
	}

	c.JSON(http.StatusOK, p)
}

// DebitarEstoque - POST /produtos/:id/debitar
// Usado internamente pelo serviço de faturamento
func DebitarEstoque(c *gin.Context) {
	id := c.Param("id")
	var req models.DebitarEstoqueRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Dados inválidos",
			Details: err.Error(),
		})
		return
	}

	// Usa SELECT FOR UPDATE para evitar race condition (requisito opcional de concorrência)
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao iniciar transação"})
		return
	}
	defer tx.Rollback()

	var saldoAtual int
	err = tx.QueryRow(`
		SELECT saldo FROM produtos WHERE id = $1 FOR UPDATE
	`, id).Scan(&saldoAtual)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Produto não encontrado"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao buscar produto"})
		return
	}

	// Verifica se há saldo suficiente
	if saldoAtual < req.Quantidade {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Saldo insuficiente",
			Details: fmt.Sprintf("Saldo atual: %d, quantidade solicitada: %d", saldoAtual, req.Quantidade),
		})
		return
	}

	// Debita o saldo
	var p models.Produto
	err = tx.QueryRow(`
		UPDATE produtos
		SET saldo = saldo - $1, updated_at = NOW()
		WHERE id = $2
		RETURNING id, codigo, descricao, saldo, created_at, updated_at
	`, req.Quantidade, id).Scan(
		&p.ID, &p.Codigo, &p.Descricao, &p.Saldo, &p.CreatedAt, &p.UpdatedAt,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao debitar estoque"})
		return
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao confirmar transação"})
		return
	}

	c.JSON(http.StatusOK, p)
}
