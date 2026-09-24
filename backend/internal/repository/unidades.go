package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/util"
)

const colunasUnidade = `id, secretaria_id, nome`

func (r *Repo) BuscarUnidadePorID(ctx context.Context, id string) (*domain.Unidade, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasUnidade+` from unidades where id = $1`, id)
	return escanearUnidade(row)
}

// Resolve uma unidade pelo nome (normalizado) dentro de uma secretaria, criando
// automaticamente se não existir ainda — decisão do dono do produto (19/09, ver skill
// importacao-planilha): "Local no mapa" é a chave, e unidade nova não deve travar o
// import esperando cadastro manual prévio.
func (r *Repo) ResolverOuCriarUnidade(ctx context.Context, secretariaID, nome string) (*domain.Unidade, error) {
	existentes, err := r.ListarUnidades(ctx, secretariaID)
	if err != nil {
		return nil, err
	}
	alvo := util.NormalizarNome(nome)
	for _, u := range existentes {
		if util.NormalizarNome(u.Nome) == alvo {
			return &u, nil
		}
	}

	row := r.Pool.QueryRow(ctx, `
		insert into unidades (secretaria_id, nome)
		values ($1, $2)
		returning `+colunasUnidade, secretariaID, strings.TrimSpace(nome))
	return escanearUnidade(row)
}

func (r *Repo) ListarUnidades(ctx context.Context, secretariaID string) ([]domain.Unidade, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasUnidade+` from unidades where secretaria_id = $1`, secretariaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var unidades []domain.Unidade
	for rows.Next() {
		var u domain.Unidade
		if err := escanearLinhaUnidade(rows, &u); err != nil {
			return nil, err
		}
		unidades = append(unidades, u)
	}
	return unidades, rows.Err()
}

// escanearLinhaUnidade existe pra reaproveitar a mesma ordem de colunas entre pgx.Row
// (QueryRow) e pgx.Rows (Query) sem duplicar a lista de campos do Scan.
func escanearLinhaUnidade(row interface {
	Scan(dest ...any) error
}, u *domain.Unidade) error {
	return row.Scan(&u.ID, &u.SecretariaID, &u.Nome)
}

func escanearUnidade(row pgx.Row) (*domain.Unidade, error) {
	var u domain.Unidade
	err := escanearLinhaUnidade(row, &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}
