# Korp Teste - Sistema de Emissão de Notas Fiscais

## Arquitetura de Microsserviços

```
                    ┌──────────────────┐
                    │   Frontend       │
                    │   Angular 17     │
                    │   :4200          │
                    └────┬────────┬────┘
                         │        │
              ┌──────────▼──┐  ┌──▼──────────────┐
              │  Serviço de  │  │  Serviço de      │
              │  Estoque     │  │  Faturamento     │
              │  Go + Gin    │  │  Go + Gin        │
              │  :8081       │◄─┤  :8082           │
              └──────┬───────┘  └──────┬───────────┘
                     │                 │
              ┌──────▼───────┐  ┌──────▼───────────┐
              │ korp_estoque │  │ korp_faturamento  │
              │ PostgreSQL   │  │ PostgreSQL        │
              └──────────────┘  └───────────────────┘
```

## Tecnologias Utilizadas

| Camada    | Tecnologia                                      |
|-----------|--------------------------------------------------|
| Frontend  | Angular 17, Angular Material, RxJS               |
| Backend   | Go 1.22, Gin (HTTP framework), lib/pq (driver PG)|
| Banco     | PostgreSQL 16                                    |
| Infra     | Docker, Docker Compose                           |

## Como Rodar

### Opção 1: Docker Compose (recomendado)

```bash
git clone https://github.com/seunome/Korp_Teste_GiovanniSantos.git
cd Korp_Teste_GiovanniSantos
docker-compose up --build
```

Acesse: http://localhost:4200

### Opção 2: Desenvolvimento local

**1. Banco de dados:**
```bash
docker run -d --name korp_pg \
  -e POSTGRES_USER=korp \
  -e POSTGRES_PASSWORD=korp123 \
  -p 5432:5432 postgres:16-alpine

docker exec -it korp_pg psql -U korp -c "CREATE DATABASE korp_estoque;"
docker exec -it korp_pg psql -U korp -c "CREATE DATABASE korp_faturamento;"
```

**2. Serviço de Estoque:**
```bash
cd estoque-service
go mod tidy
go run main.go
# Rodando em http://localhost:8081
```

**3. Serviço de Faturamento (outro terminal):**
```bash
cd faturamento-service
go mod tidy
go run main.go
# Rodando em http://localhost:8082
```

**4. Frontend Angular (outro terminal):**
```bash
cd frontend
npm install
ng serve
# Rodando em http://localhost:4200
```

## Endpoints da API

### Estoque (:8081)

| Método | Rota                  | Descrição                   |
|--------|-----------------------|-----------------------------|
| GET    | /health               | Health check                |
| POST   | /produtos             | Criar produto               |
| GET    | /produtos             | Listar produtos             |
| GET    | /produtos/:id         | Buscar produto por ID       |
| PUT    | /produtos/:id         | Atualizar produto           |
| POST   | /produtos/:id/debitar | Debitar saldo (uso interno) |

### Faturamento (:8082)

| Método | Rota                  | Descrição              |
|--------|-----------------------|------------------------|
| GET    | /health               | Health check           |
| POST   | /notas                | Criar nota fiscal      |
| GET    | /notas                | Listar notas           |
| GET    | /notas/:id            | Buscar nota por ID     |
| POST   | /notas/:id/imprimir   | Imprimir nota fiscal   |

## Testando Cenário de Falha

```bash
# 1. Crie um produto e uma nota fiscal pela interface

# 2. Derrube o serviço de estoque
docker stop korp_estoque

# 3. Tente imprimir a nota → erro amigável na tela

# 4. Suba novamente e reimprima
docker start korp_estoque
```
