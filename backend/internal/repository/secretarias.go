package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
)

const colunasSecretaria = `id, prefeitura_id, nome, sigla`

func escanearLinhaSecretaria(row interface {
	Scan(dest ...any) error
}, s *domain.Secretaria) error {
	return row.Scan(&s.ID, &s.PrefeituraID, &s.Nome, &s.Sigla)
}

func escanearSecretaria(row pgx.Row) (*domain.Secretaria, error) {
	var s domain.Secretaria
	err := escanearLinhaSecretaria(row, &s)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *Repo) BuscarSecretariaPorID(ctx context.Context, id string) (*domain.Secretaria, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasSecretaria+` from secretarias where id = $1`, id)
	return escanearSecretaria(row)
}

func (r *Repo) ListarSecretariasPorPrefeitura(ctx context.Context, prefeituraID string) ([]domain.Secretaria, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasSecretaria+` from secretarias where prefeitura_id = $1 order by nome`, prefeituraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []domain.Secretaria
	for rows.Next() {
		var s domain.Secretaria
		if err := escanearLinhaSecretaria(rows, &s); err != nil {
			return nil, err
		}
		lista = append(lista, s)
	}
	return lista, rows.Err()
}

func (r *Repo) CriarSecretaria(ctx context.Context, prefeituraID, nome, sigla string) (*domain.Secretaria, error) {
	row := r.Pool.QueryRow(ctx, `
		insert into secretarias (prefeitura_id, nome, sigla)
		values ($1, $2, $3)
		returning `+colunasSecretaria,
		prefeituraID, strings.TrimSpace(nome), strings.TrimSpace(sigla))
	return escanearSecretaria(row)
}
