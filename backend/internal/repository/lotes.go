package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
)

// CriarLote começa por SECRETARIA (corrigido 19/09) — um único upload gera agendamentos
// em várias unidades ao mesmo tempo, então o lote não podia pertencer a uma unidade só.
func (r *Repo) CriarLote(ctx context.Context, tx pgx.Tx, secretariaID, usuarioID, arquivoNome string) (string, error) {
	var id string
	err := tx.QueryRow(ctx, `
		insert into lotes_importacao (secretaria_id, usuario_id, arquivo_nome, status)
		values ($1, $2, $3, 'processando')
		returning id
	`, secretariaID, usuarioID, arquivoNome).Scan(&id)
	return id, err
}

func (r *Repo) FinalizarLote(ctx context.Context, tx pgx.Tx, loteID, status string, totalLinhas, totalErros int) error {
	_, err := tx.Exec(ctx, `
		update lotes_importacao set status = $2, total_linhas = $3, total_erros = $4 where id = $1
	`, loteID, status, totalLinhas, totalErros)
	return err
}
