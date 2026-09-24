package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/util"
)

const colunasPrefeitura = `id, nome, slug, issuer_jwt, cor_destaque, cor_clara`

func escanearLinhaPrefeitura(row interface {
	Scan(dest ...any) error
}, p *domain.Prefeitura) error {
	return row.Scan(&p.ID, &p.Nome, &p.Slug, &p.IssuerJWT, &p.CorDestaque, &p.CorClara)
}

func escanearPrefeitura(row pgx.Row) (*domain.Prefeitura, error) {
	var p domain.Prefeitura
	err := escanearLinhaPrefeitura(row, &p)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repo) ListarPrefeituras(ctx context.Context) ([]domain.Prefeitura, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasPrefeitura+` from prefeituras order by nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []domain.Prefeitura
	for rows.Next() {
		var p domain.Prefeitura
		if err := escanearLinhaPrefeitura(rows, &p); err != nil {
			return nil, err
		}
		lista = append(lista, p)
	}
	return lista, rows.Err()
}

func (r *Repo) BuscarPrefeituraPorID(ctx context.Context, id string) (*domain.Prefeitura, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasPrefeitura+` from prefeituras where id = $1`, id)
	return escanearPrefeitura(row)
}

// slugificar deriva um slug simples a partir do nome (minúsculas, espaço vira hífen) — só
// pra dar um identificador legível de URL/log; não precisa ser sofisticado no MVP.
func slugificar(nome string) string {
	s := util.NormalizarNome(nome)
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "-")
}

func (r *Repo) CriarPrefeitura(ctx context.Context, nome string) (*domain.Prefeitura, error) {
	row := r.Pool.QueryRow(ctx, `
		insert into prefeituras (nome, slug)
		values ($1, $2)
		returning `+colunasPrefeitura,
		strings.TrimSpace(nome), slugificar(nome))
	return escanearPrefeitura(row)
}

// AtualizarPrefeitura grava nome e as duas cores de identidade (destaque/clara,
// configuráveis pelo admin — ver skill identidade-visual).
func (r *Repo) AtualizarPrefeitura(ctx context.Context, id, nome, corDestaque, corClara string) (*domain.Prefeitura, error) {
	row := r.Pool.QueryRow(ctx, `
		update prefeituras set nome = $2, cor_destaque = $3, cor_clara = $4
		where id = $1
		returning `+colunasPrefeitura,
		id, strings.TrimSpace(nome), corDestaque, corClara)
	return escanearPrefeitura(row)
}
