package models

import "time"

// NotaFiscal representa uma nota fiscal
type NotaFiscal struct {
	ID        int64      `json:"id"`
	Numero    int        `json:"numero"`
	Status    string     `json:"status"` // "Aberta" ou "Fechada"
	Itens     []NotaItem `json:"itens,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// NotaItem representa um produto dentro da nota fiscal
type NotaItem struct {
	ID         int64 `json:"id"`
	NotaID     int64 `json:"nota_id"`
	ProdutoID  int64 `json:"produto_id"`
	Quantidade int   `json:"quantidade"`
}

// CriarNotaRequest é o payload para criar uma nota
type CriarNotaRequest struct {
	Itens []ItemRequest `json:"itens" binding:"required,min=1,dive"`
}

// ItemRequest é um item no payload de criação
type ItemRequest struct {
	ProdutoID  int64 `json:"produto_id" binding:"required"`
	Quantidade int   `json:"quantidade" binding:"required,min=1"`
}

// ErrorResponse padroniza respostas de erro
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
