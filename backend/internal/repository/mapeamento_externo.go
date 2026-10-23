package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
)

// mapeamento_externo traduz os IDs numéricos que a Conecta Cidades manda no token
// (`department_ids`, `unit_ids`) pros nossos próprios UUIDs de secretaria/unidade — ver
// skill modelo-dados e auth-conecta-cidades, seção 6. Cadastrado manualmente pelo gestor/
// admin (não existe tela pra isso ainda — fora de escopo até o SSO sair do piloto, ver
// pendências da skill).
//
// ESCOPADO POR PREFEITURA (22/10, corrigido por revisão de segurança): `department_id`/
// `unit_id` são IDs internos POR MUNICÍPIO na Conecta Cidades — sem o escopo, um
// department_id=3 de uma prefeitura podia resolver pro mapeamento cadastrado por OUTRA
// prefeitura (vazamento de acesso entre tenants). `prefeituraID` vem SEMPRE do emissor já
// verificado (BuscarPrefeituraPorIssuer), nunca de dado do token em si.
//
// Um usuário só tem UMA secretaria/unidade no nosso schema, mas o token pode trazer VÁRIOS
// department_ids/unit_ids (um servidor pode estar ligado a mais de um setor na plataforma) —
// resolução simplificada: usa o PRIMEIRO id da lista que tiver mapeamento cadastrado aqui,
// ignora o resto. Documentado como simplificação deliberada, não bug — revisar se algum dia
// um usuário precisar de fato de múltiplos vínculos.

// ResolverSecretariaPorExternalIDs devolve o id da primeira secretaria mapeada (dentro da
// prefeitura dada) entre os `departmentIDs` dados, ou nil se nenhum tiver mapeamento.
func (r *Repo) ResolverSecretariaPorExternalIDs(ctx context.Context, prefeituraID string, departmentIDs []int) (*string, error) {
	// Defesa em profundidade: além do mapeamento em si já estar escopado por prefeitura_id
	// (índice único composto, migration 000011), o join contra secretarias confirma que a
	// secretaria apontada REALMENTE pertence a essa prefeitura — protege contra um erro de
	// cadastro manual (ninguém tem UI pra isso ainda) que gravasse prefeitura_id certo mas
	// secretaria_id de outro tenant.
	query := `
		select m.secretaria_id
		from mapeamento_externo m
		join secretarias s on s.id = m.secretaria_id
		where m.prefeitura_id = $1 and m.tipo = 'department' and m.external_id = $2 and s.prefeitura_id = $1
	`
	return resolverPorExternalIDs(ctx, r, query, prefeituraID, departmentIDs)
}

// ResolverUnidadePorExternalIDs devolve o id da primeira unidade mapeada (dentro da
// prefeitura dada) entre os `unitIDs` dados, ou nil se nenhum tiver mapeamento.
func (r *Repo) ResolverUnidadePorExternalIDs(ctx context.Context, prefeituraID string, unitIDs []int) (*string, error) {
	query := `
		select m.unidade_id
		from mapeamento_externo m
		join unidades u on u.id = m.unidade_id
		join secretarias s on s.id = u.secretaria_id
		where m.prefeitura_id = $1 and m.tipo = 'unit' and m.external_id = $2 and s.prefeitura_id = $1
	`
	return resolverPorExternalIDs(ctx, r, query, prefeituraID, unitIDs)
}

func resolverPorExternalIDs(ctx context.Context, r *Repo, query, prefeituraID string, ids []int) (*string, error) {
	for _, id := range ids {
		var valor string
		err := r.Pool.QueryRow(ctx, query, prefeituraID, strconv.Itoa(id)).Scan(&valor)
		if err == nil {
			return &valor, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}
	return nil, nil
}
