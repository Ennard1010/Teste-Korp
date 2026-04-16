package models

import "time"

// Produto representa um item no estoque
type Produto struct {
	ID        int64     `json:"id"`
	Codigo    string    `json:"codigo"`
	Descricao string    `json:"descricao"`
	Saldo     int       `json:"saldo"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ProdutoRequest é o payload de criação/atualização
type ProdutoRequest struct {
	Codigo    string `json:"codigo" binding:"required"`
	Descricao string `json:"descricao" binding:"required"`
	Saldo     int    `json:"saldo" binding:"required,min=0"`
}

// DebitarEstoqueRequest é o payload para debitar saldo
type DebitarEstoqueRequest struct {
	Quantidade int `json:"quantidade" binding:"required,min=1"`
}

// ErrorResponse padroniza respostas de erro
type ErrorResponse struct {
	Error   string `json:"error"`
	Details string `json:"details,omitempty"`
}
