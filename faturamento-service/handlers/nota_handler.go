package handlers

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"

	"faturamento-service/database"
	"faturamento-service/models"
)

// Cliente HTTP com timeout para chamadas ao serviço de estoque
var httpClient = &http.Client{
	Timeout: 5 * time.Second,
}

func getEstoqueURL() string {
	url := os.Getenv("ESTOQUE_SERVICE_URL")
	if url == "" {
		url = "http://localhost:8081"
	}
	return url
}

// CriarNota - POST /notas
func CriarNota(c *gin.Context) {
	var req models.CriarNotaRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Dados inválidos",
			Details: err.Error(),
		})
		return
	}

	// Inicia transação
	tx, err := database.DB.Begin()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao iniciar transação"})
		return
	}
	defer tx.Rollback()

	// Pega o próximo número sequencial
	var numero int
	err = tx.QueryRow("SELECT nextval('nota_numero_seq')").Scan(&numero)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao gerar número da nota"})
		return
	}

	// Insere a nota fiscal
	var nota models.NotaFiscal
	err = tx.QueryRow(`
		INSERT INTO notas_fiscais (numero, status)
		VALUES ($1, 'Aberta')
		RETURNING id, numero, status, created_at, updated_at
	`, numero).Scan(&nota.ID, &nota.Numero, &nota.Status, &nota.CreatedAt, &nota.UpdatedAt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao criar nota fiscal"})
		return
	}

	// Insere os itens da nota
	nota.Itens = []models.NotaItem{}
	for _, item := range req.Itens {
		var notaItem models.NotaItem
		err = tx.QueryRow(`
			INSERT INTO nota_itens (nota_id, produto_id, quantidade)
			VALUES ($1, $2, $3)
			RETURNING id, nota_id, produto_id, quantidade
		`, nota.ID, item.ProdutoID, item.Quantidade).Scan(
			&notaItem.ID, &notaItem.NotaID, &notaItem.ProdutoID, &notaItem.Quantidade,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao adicionar item à nota"})
			return
		}
		nota.Itens = append(nota.Itens, notaItem)
	}

	if err = tx.Commit(); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao confirmar transação"})
		return
	}

	c.JSON(http.StatusCreated, nota)
}

// ListarNotas - GET /notas
func ListarNotas(c *gin.Context) {
	rows, err := database.DB.Query(`
		SELECT id, numero, status, created_at, updated_at
		FROM notas_fiscais
		ORDER BY numero DESC
	`)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao listar notas"})
		return
	}
	defer rows.Close()

	notas := []models.NotaFiscal{}
	for rows.Next() {
		var n models.NotaFiscal
		if err := rows.Scan(&n.ID, &n.Numero, &n.Status, &n.CreatedAt, &n.UpdatedAt); err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao ler nota"})
			return
		}
		// Busca os itens da nota
		n.Itens, _ = buscarItensDaNota(n.ID)
		notas = append(notas, n)
	}

	c.JSON(http.StatusOK, notas)
}

// BuscarNota - GET /notas/:id
func BuscarNota(c *gin.Context) {
	id := c.Param("id")

	var n models.NotaFiscal
	err := database.DB.QueryRow(`
		SELECT id, numero, status, created_at, updated_at
		FROM notas_fiscais WHERE id = $1
	`, id).Scan(&n.ID, &n.Numero, &n.Status, &n.CreatedAt, &n.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Nota fiscal não encontrada"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao buscar nota"})
		return
	}

	n.Itens, _ = buscarItensDaNota(n.ID)
	c.JSON(http.StatusOK, n)
}

// ImprimirNota - POST /notas/:id/imprimir
// Este é o handler mais importante: muda status e debita estoque via microsserviço
func ImprimirNota(c *gin.Context) {
	id := c.Param("id")

	// 1. Busca a nota e valida status
	var nota models.NotaFiscal
	err := database.DB.QueryRow(`
		SELECT id, numero, status, created_at, updated_at
		FROM notas_fiscais WHERE id = $1
	`, id).Scan(&nota.ID, &nota.Numero, &nota.Status, &nota.CreatedAt, &nota.UpdatedAt)

	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "Nota fiscal não encontrada"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao buscar nota"})
		return
	}

	// Só permite imprimir notas com status "Aberta"
	if nota.Status != "Aberta" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error:   "Nota fiscal não pode ser impressa",
			Details: "Apenas notas com status 'Aberta' podem ser impressas",
		})
		return
	}

	// 2. Busca os itens da nota
	itens, err := buscarItensDaNota(nota.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao buscar itens da nota"})
		return
	}

	// 3. Debita o estoque de cada produto via microsserviço de estoque
	for _, item := range itens {
		err := debitarEstoque(item.ProdutoID, item.Quantidade)
		if err != nil {
			// TRATAMENTO DE FALHA: se o serviço de estoque falhar,
			// não altera o status da nota e informa o usuário
			c.JSON(http.StatusServiceUnavailable, models.ErrorResponse{
				Error: "Falha ao comunicar com o serviço de estoque",
				Details: fmt.Sprintf(
					"Não foi possível debitar o produto %d. Tente novamente mais tarde. Detalhe: %s",
					item.ProdutoID, err.Error(),
				),
			})
			return
		}
	}

	// 4. Atualiza o status da nota para "Fechada"
	_, err = database.DB.Exec(`
		UPDATE notas_fiscais
		SET status = 'Fechada', updated_at = NOW()
		WHERE id = $1
	`, nota.ID)

	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "Erro ao atualizar status da nota"})
		return
	}

	nota.Status = "Fechada"
	nota.Itens = itens
	c.JSON(http.StatusOK, gin.H{
		"message": "Nota fiscal impressa com sucesso",
		"nota":    nota,
	})
}

// ---------- Funções auxiliares ----------

// buscarItensDaNota retorna os itens de uma nota fiscal
func buscarItensDaNota(notaID int64) ([]models.NotaItem, error) {
	rows, err := database.DB.Query(`
		SELECT id, nota_id, produto_id, quantidade
		FROM nota_itens WHERE nota_id = $1
	`, notaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	itens := []models.NotaItem{}
	for rows.Next() {
		var item models.NotaItem
		if err := rows.Scan(&item.ID, &item.NotaID, &item.ProdutoID, &item.Quantidade); err != nil {
			return nil, err
		}
		itens = append(itens, item)
	}
	return itens, nil
}

// debitarEstoque chama o microsserviço de estoque para debitar o saldo
func debitarEstoque(produtoID int64, quantidade int) error {
	url := fmt.Sprintf("%s/produtos/%d/debitar", getEstoqueURL(), produtoID)

	body, _ := json.Marshal(map[string]int{"quantidade": quantidade})

	resp, err := httpClient.Post(url, "application/json", bytes.NewBuffer(body))
	if err != nil {
		// Serviço de estoque fora do ar ou timeout
		return fmt.Errorf("serviço de estoque indisponível: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Lê a resposta de erro do serviço de estoque
		respBody, _ := io.ReadAll(resp.Body)
		var errResp models.ErrorResponse
		json.Unmarshal(respBody, &errResp)
		return fmt.Errorf("erro do serviço de estoque (HTTP %d): %s", resp.StatusCode, errResp.Error)
	}

	return nil
}
