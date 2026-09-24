package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
)

var ErrNaoEncontrado = errors.New("nao encontrado")
var ErrEmailJaExiste = errors.New("email ja existe")

const colunasUsuario = `id, secretaria_id, unidade_id, nome, email, senha_hash, papel, external_sub, ativo, ultimo_acesso_em`

func escanearLinhaUsuario(row interface {
	Scan(dest ...any) error
}, u *domain.Usuario) error {
	return row.Scan(&u.ID, &u.SecretariaID, &u.UnidadeID, &u.Nome, &u.Email, &u.SenhaHash, &u.Papel, &u.ExternalSub, &u.Ativo, &u.UltimoAcessoEm)
}

// Login busca por email direto, sem escopo nenhum — usuarios.email é único globalmente
// desde a introdução do admin (22/09, ver skill auth-login-mvp).
func (r *Repo) BuscarUsuarioPorEmail(ctx context.Context, email string) (*domain.Usuario, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasUsuario+` from usuarios where email = $1 limit 1`, email)
	return escanearUsuario(row)
}

// ListarGradesDoUsuario — as grades em que um atendente/recepcionista está alocado (23/09,
// "atendente ou recepção podem ser alocados em mais de uma grade"). Grades excluídas não
// aparecem (não faz sentido oferecer pra trabalhar numa fila que nem existe mais na tela de
// gestão). Usado tanto pra montar a sessão quanto pra mostrar a "etiqueta" de cada grade
// alocada na tela "Usuários internos".
//
// Devolve `GradeComUnidade` (com `UnidadeNome` junto, 23/09 — auditoria de multi-grade):
// como um atendente pode legitimamente ter grades de UNIDADES DIFERENTES (ver exemplo real
// do dono do produto — "João" com grades na Unidade A e B), o `usuario.UnidadeID` (singular)
// deixou de ser suficiente pra saber "onde" cada grade fica — precisa vir por grade mesmo,
// pra o operacional/painel poderem agrupar por local corretamente.
func (r *Repo) ListarGradesDoUsuario(ctx context.Context, usuarioID string) ([]GradeComUnidade, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasGradeComAlias+`, un.nome
		from grades g
		join usuario_grades ug on ug.grade_id = g.id
		join unidades un on un.id = g.unidade_id
		where ug.usuario_id = $1 and g.excluido_em is null
		order by un.nome, g.nome
	`, usuarioID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []GradeComUnidade
	for rows.Next() {
		var g GradeComUnidade
		if err := rows.Scan(&g.ID, &g.UnidadeID, &g.Servico, &g.Nome, &g.SLAChegadaMinutos, &g.SLAAtendimentoMinutos,
			&g.DuracaoChamadaPainelSegundos, &g.DuracaoChamadaPainelFilaSegundos,
			&g.RepeticoesChamada, &g.IntervaloRepeticaoSegundos, &g.CooldownRechamadaMinutos,
			&g.PermiteEncaixeRecepcao, &g.Ativo, &g.CriadoEm, &g.Excluida, &g.UnidadeNome); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, rows.Err()
}

// GradesPorUsuario resolve as grades alocadas de VÁRIOS usuários numa passada só (evita N+1
// na listagem "Usuários internos", que já mostra dezenas de linhas de uma vez) — devolve um
// mapa usuarioID -> grades dele (com UnidadeNome, mesmo motivo de ListarGradesDoUsuario).
func (r *Repo) GradesPorUsuario(ctx context.Context, usuarioIDs []string) (map[string][]GradeComUnidade, error) {
	out := map[string][]GradeComUnidade{}
	if len(usuarioIDs) == 0 {
		return out, nil
	}
	rows, err := r.Pool.Query(ctx, `
		select ug.usuario_id, `+colunasGradeComAlias+`, un.nome
		from usuario_grades ug
		join grades g on g.id = ug.grade_id
		join unidades un on un.id = g.unidade_id
		where ug.usuario_id = any($1) and g.excluido_em is null
		order by un.nome, g.nome
	`, usuarioIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var usuarioID string
		var g GradeComUnidade
		if err := rows.Scan(&usuarioID, &g.ID, &g.UnidadeID, &g.Servico, &g.Nome, &g.SLAChegadaMinutos, &g.SLAAtendimentoMinutos,
			&g.DuracaoChamadaPainelSegundos, &g.DuracaoChamadaPainelFilaSegundos,
			&g.RepeticoesChamada, &g.IntervaloRepeticaoSegundos, &g.CooldownRechamadaMinutos,
			&g.PermiteEncaixeRecepcao, &g.Ativo, &g.CriadoEm, &g.Excluida, &g.UnidadeNome); err != nil {
			return nil, err
		}
		out[usuarioID] = append(out[usuarioID], g)
	}
	return out, rows.Err()
}

// ExcluirUsuario (23/09) — "excluir" reaproveita `ativo` (já bloqueava login, nunca tinha
// botão de UI antes) em vez de uma coluna nova só pra isso, ver nota na migration 000008.
func (r *Repo) ExcluirUsuario(ctx context.Context, usuarioID string) error {
	_, err := r.Pool.Exec(ctx, `update usuarios set ativo = false where id = $1`, usuarioID)
	return err
}

// ReativarUsuario (23/09, auditoria de reativação) — desfaz ExcluirUsuario. As alocações de
// grade (`usuario_grades`) NUNCA foram tocadas pela exclusão (só `ExcluirGrade` limpa esse
// vínculo, e só da grade que foi excluída) — um atendente reativado volta a ver exatamente as
// grades que já tinha antes de ser desativado, sem precisar realocar nada.
func (r *Repo) ReativarUsuario(ctx context.Context, usuarioID string) error {
	_, err := r.Pool.Exec(ctx, `update usuarios set ativo = true where id = $1`, usuarioID)
	return err
}

func (r *Repo) BuscarUsuarioPorID(ctx context.Context, id string) (*domain.Usuario, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasUsuario+` from usuarios where id = $1`, id)
	return escanearUsuario(row)
}

func escanearUsuario(row pgx.Row) (*domain.Usuario, error) {
	var u domain.Usuario
	err := escanearLinhaUsuario(row, &u)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &u, nil
}

// AtualizarUltimoAcesso grava o instante do login — base da tela "Usuários internos" do
// gestor pra distinguir "pendente" (nunca logou, ultimo_acesso_em nulo) de "último acesso
// em X" (22/09).
func (r *Repo) AtualizarUltimoAcesso(ctx context.Context, usuarioID string) error {
	_, err := r.Pool.Exec(ctx, `update usuarios set ultimo_acesso_em = now() where id = $1`, usuarioID)
	return err
}

// ListarUsuariosPorUnidade é a base da tela "Usuários internos" antes do admin existir —
// mantida pra listar quem trabalha numa unidade específica (recepcionista + atendentes
// dela). Não inclui gestor (que não tem mais unidade_id, ver ListarUsuariosPorSecretaria).
func (r *Repo) ListarUsuariosPorUnidade(ctx context.Context, unidadeID string) ([]domain.Usuario, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasUsuario+` from usuarios where unidade_id = $1 order by nome`, unidadeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		if err := escanearLinhaUsuario(rows, &u); err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	return lista, rows.Err()
}

// ListarUsuariosPorSecretaria (22/09): a tela "Usuários internos" do gestor agora precisa
// mostrar todo mundo da secretaria inteira — o gestor em si, e todo recepcionista/atendente
// de QUALQUER unidade daquela secretaria (join, já que esses dois papéis só têm unidade_id,
// não secretaria_id direto). Inclui usuários desativados (23/09, auditoria de reativação —
// antes somiam sem deixar rastro; agora aparecem numa aba "Inativos" com botão de reativar,
// ver Usuarios.tsx). O campo `Ativo` no retorno é o que o frontend usa pra decidir a aba.
func (r *Repo) ListarUsuariosPorSecretaria(ctx context.Context, secretariaID string) ([]domain.Usuario, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasUsuario+` from usuarios where secretaria_id = $1
		union all
		select u.id, u.secretaria_id, u.unidade_id, u.nome, u.email, u.senha_hash, u.papel, u.external_sub, u.ativo, u.ultimo_acesso_em
		from usuarios u
		join unidades un on un.id = u.unidade_id
		where un.secretaria_id = $1
		order by nome
	`, secretariaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []domain.Usuario
	for rows.Next() {
		var u domain.Usuario
		if err := escanearLinhaUsuario(rows, &u); err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	return lista, rows.Err()
}

// NovoUsuario — campos pra criar um usuário interno (admin cria gestor/atendente/
// recepcionista; ver handlers/admin.go). Exatamente um de SecretariaID/UnidadeID é
// preenchido dependendo do papel (gestor: secretaria; atendente/recepcionista: unidade) —
// ver comentário em domain.Usuario. GradeIDs (23/09) substitui a antiga FK única — pode vir
// vazia (recepcionista sem restrição de grade) ou com uma ou mais.
type NovoUsuario struct {
	Nome         string
	Email        string
	SenhaHash    string
	Papel        domain.Papel
	SecretariaID *string
	UnidadeID    *string
	GradeIDs     []string
}

func (r *Repo) CriarUsuario(ctx context.Context, n NovoUsuario) (*domain.Usuario, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		insert into usuarios (secretaria_id, unidade_id, nome, email, senha_hash, papel, ativo)
		values ($1, $2, $3, $4, $5, $6, true)
		returning `+colunasUsuario,
		n.SecretariaID, n.UnidadeID, strings.TrimSpace(n.Nome), strings.TrimSpace(n.Email), n.SenhaHash, n.Papel)
	u, err := escanearUsuario(row)
	if err != nil {
		if strings.Contains(err.Error(), "usuarios_email_key") {
			return nil, ErrEmailJaExiste
		}
		return nil, err
	}
	for _, gradeID := range n.GradeIDs {
		if _, err := tx.Exec(ctx, `insert into usuario_grades (usuario_id, grade_id) values ($1, $2)`, u.ID, gradeID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return u, nil
}

// UsuarioComContexto é a linha enriquecida usada pela tela "Usuários internos" do admin/
// gestor (22/09) — junta o nome da prefeitura/secretaria/unidade numa consulta só, pra não
// obrigar o frontend a fazer N+1 requisições resolvendo cada vínculo. `Grades` (23/09) é
// preenchido À PARTE via GradesPorUsuario (não dá mais pra vir no mesmo join — é N:N agora).
type UsuarioComContexto struct {
	domain.Usuario
	PrefeituraNome *string
	SecretariaNome *string
	UnidadeNome    *string
	Grades         []GradeComUnidade
}

const colunasUsuarioComContexto = `
	u.id, u.secretaria_id, u.unidade_id, u.nome, u.email, u.senha_hash, u.papel,
	u.external_sub, u.ativo, u.ultimo_acesso_em,
	coalesce(p1.nome, p2.nome), coalesce(s1.nome, s2.nome), un.nome
`

const juncoesUsuarioComContexto = `
	from usuarios u
	left join secretarias s1 on s1.id = u.secretaria_id
	left join prefeituras p1 on p1.id = s1.prefeitura_id
	left join unidades un on un.id = u.unidade_id
	left join secretarias s2 on s2.id = un.secretaria_id
	left join prefeituras p2 on p2.id = s2.prefeitura_id
`

func escanearUsuarioComContexto(rows interface {
	Scan(dest ...any) error
}) (UsuarioComContexto, error) {
	var u UsuarioComContexto
	err := rows.Scan(&u.ID, &u.SecretariaID, &u.UnidadeID, &u.Nome, &u.Email, &u.SenhaHash, &u.Papel,
		&u.ExternalSub, &u.Ativo, &u.UltimoAcessoEm, &u.PrefeituraNome, &u.SecretariaNome, &u.UnidadeNome)
	return u, err
}

// ListarTodosUsuariosInternos: tela "Usuários internos" consolidada do admin (22/09,
// pedido explícito — "cria gestores e outros funcionarios na mesma tela") — gestor,
// atendente e recepcionista da plataforma inteira, numa lista só. Inclui usuários
// desativados (23/09, auditoria de reativação, mesma nota de ListarUsuariosPorSecretaria).
func (r *Repo) ListarTodosUsuariosInternos(ctx context.Context) ([]UsuarioComContexto, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasUsuarioComContexto+juncoesUsuarioComContexto+`
		where u.papel in ('gestor', 'atendente', 'recepcionista') order by u.nome`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lista []UsuarioComContexto
	for rows.Next() {
		u, err := escanearUsuarioComContexto(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	rows.Close()

	ids := make([]string, len(lista))
	for i, u := range lista {
		ids[i] = u.ID
	}
	gradesPorUsuario, err := r.GradesPorUsuario(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range lista {
		lista[i].Grades = gradesPorUsuario[lista[i].ID]
	}
	return lista, nil
}

// ListarUsuariosInternosPorPrefeitura (23/09) — mesma coisa que ListarTodosUsuariosInternos,
// mas escopada a UMA prefeitura (aba "Usuários" dentro da configuração de uma prefeitura
// específica, no admin — "dashboard e usuarios internos... dentro da configuração de cada
// prefeitura").
func (r *Repo) ListarUsuariosInternosPorPrefeitura(ctx context.Context, prefeituraID string) ([]UsuarioComContexto, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasUsuarioComContexto+juncoesUsuarioComContexto+`
		where u.papel in ('gestor', 'atendente', 'recepcionista') and u.ativo = true
		and coalesce(p1.id, p2.id) = $1
		order by u.nome`, prefeituraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lista []UsuarioComContexto
	for rows.Next() {
		u, err := escanearUsuarioComContexto(rows)
		if err != nil {
			return nil, err
		}
		lista = append(lista, u)
	}
	rows.Close()

	ids := make([]string, len(lista))
	for i, u := range lista {
		ids[i] = u.ID
	}
	gradesPorUsuario, err := r.GradesPorUsuario(ctx, ids)
	if err != nil {
		return nil, err
	}
	for i := range lista {
		lista[i].Grades = gradesPorUsuario[lista[i].ID]
	}
	return lista, nil
}

// AtualizacaoUsuario — campos que o gestor/admin pode editar na tela "Usuários internos"
// (22/09): nome, email, senha (opcional — só troca se vier preenchida) e as grades em que
// atendente/recepcionista estão alocados (23/09, substitui a FK única por uma lista).
type AtualizacaoUsuario struct {
	Nome          string
	Email         string
	Papel         domain.Papel
	NovaSenhaHash *string
	GradeIDs      []string
}

func (r *Repo) AtualizarUsuario(ctx context.Context, usuarioID string, a AtualizacaoUsuario) (*domain.Usuario, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	var row pgx.Row
	if a.NovaSenhaHash != nil {
		row = tx.QueryRow(ctx, `
			update usuarios set nome = $2, email = $3, papel = $4, senha_hash = $5
			where id = $1
			returning `+colunasUsuario,
			usuarioID, strings.TrimSpace(a.Nome), strings.TrimSpace(a.Email), a.Papel, *a.NovaSenhaHash)
	} else {
		row = tx.QueryRow(ctx, `
			update usuarios set nome = $2, email = $3, papel = $4
			where id = $1
			returning `+colunasUsuario,
			usuarioID, strings.TrimSpace(a.Nome), strings.TrimSpace(a.Email), a.Papel)
	}
	u, err := escanearUsuario(row)
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(ctx, `delete from usuario_grades where usuario_id = $1`, usuarioID); err != nil {
		return nil, err
	}
	for _, gradeID := range a.GradeIDs {
		if _, err := tx.Exec(ctx, `insert into usuario_grades (usuario_id, grade_id) values ($1, $2)`, usuarioID, gradeID); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return u, nil
}
