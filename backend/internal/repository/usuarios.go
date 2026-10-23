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

// senha_hash não é mais selecionada (22/10) — nada no código lê esse campo desde a remoção
// do login por senha; a coluna em si fica no banco (migration de drop não vale a pena só
// por isso), só a aplicação parou de tocar nela.
const colunasUsuario = `id, secretaria_id, unidade_id, nome, email, papel, external_sub, external_issuer, ativo, ultimo_acesso_em`

func escanearLinhaUsuario(row interface {
	Scan(dest ...any) error
}, u *domain.Usuario) error {
	return row.Scan(&u.ID, &u.SecretariaID, &u.UnidadeID, &u.Nome, &u.Email, &u.Papel, &u.ExternalSub, &u.ExternalIssuer, &u.Ativo, &u.UltimoAcessoEm)
}

// buscarUsuarioPorEmailNaPrefeitura: usada só pelo fallback por e-mail do login via SSO
// (UpsertUsuarioPortal) — ACHADO POR REVISÃO DE SEGURANÇA (22/10, mesmo dia da implementação):
// `usuarios.email` é único globalmente, mas isso NÃO significa que qualquer conta com aquele
// e-mail pode ser vinculada a qualquer token — sem escopar por prefeitura, um e-mail
// verificado por QUALQUER emissor confiado (ex: a Smartândia de teste) conseguia rebindar
// `external_sub`/`secretaria_id`/`unidade_id` de uma conta pré-provisionada de OUTRA
// prefeitura pra si, sequestrando o registro e travando o dono legítimo fora na próxima vez
// (vira `ErrVinculoExternoConflitante` pra ele). Por isso aqui: (1) `papel <> 'admin'` — conta
// global nunca tem prefeitura pra comparar, e nunca deve ser alvo de bind automático; (2) o
// `coalesce` junta os dois caminhos possíveis até uma prefeitura (gestor via secretaria_id
// direto, atendente/recepcionista via unidade_id -> secretaria) e exige que seja EXATAMENTE a
// prefeitura já resolvida pelo `iss` do token atual — uma conta de outra prefeitura (ou uma
// pré-provisionada sem nenhum vínculo ainda, prefeitura_id nulo nos dois) nunca bate.
func (r *Repo) buscarUsuarioPorEmailNaPrefeitura(ctx context.Context, email, prefeituraID string) (*domain.Usuario, error) {
	// Falha fechado (mesmo padrão já usado pra CONECTA_AUDIENCIA vazio): sem uma prefeitura
	// resolvida não há com o que comparar, então o fallback simplesmente não acha nada — nunca
	// o contrário (reinterpretar "sem escopo" como "qualquer escopo serve").
	if prefeituraID == "" {
		return nil, ErrNaoEncontrado
	}
	row := r.Pool.QueryRow(ctx, `
		select u.id, u.secretaria_id, u.unidade_id, u.nome, u.email, u.papel, u.external_sub, u.external_issuer, u.ativo, u.ultimo_acesso_em
		from usuarios u
		left join secretarias s1 on s1.id = u.secretaria_id
		left join unidades un on un.id = u.unidade_id
		left join secretarias s2 on s2.id = un.secretaria_id
		where u.email = $1 and u.papel <> 'admin' and coalesce(s1.prefeitura_id, s2.prefeitura_id) = $2
		limit 1
	`, email, prefeituraID)
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
		select u.id, u.secretaria_id, u.unidade_id, u.nome, u.email, u.papel, u.external_sub, u.external_issuer, u.ativo, u.ultimo_acesso_em
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

// NovoUsuario — campos pra PRÉ-PROVISIONAR um usuário interno (admin/gestor cria gestor/
// atendente/recepcionista com nome/email/papel/grade, mas SEM senha — 22/10, login por senha
// removido). O registro criado aqui funciona como uma "reserva de lugar": quando a pessoa
// loga de verdade pela primeira vez via SSO, `UpsertUsuarioPortal` acha esse registro pelo
// e-mail (se `email_verified` no token) e completa o vínculo externo nele, em vez de criar
// um usuário duplicado. Exatamente um de SecretariaID/UnidadeID é preenchido dependendo do
// papel (gestor: secretaria; atendente/recepcionista: unidade) — ver comentário em
// domain.Usuario. GradeIDs (23/09) substitui a antiga FK única — pode vir vazia
// (recepcionista sem restrição de grade) ou com uma ou mais.
type NovoUsuario struct {
	Nome         string
	Email        string
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
		insert into usuarios (secretaria_id, unidade_id, nome, email, papel, ativo)
		values ($1, $2, $3, $4, $5, true)
		returning `+colunasUsuario,
		n.SecretariaID, n.UnidadeID, strings.TrimSpace(n.Nome), strings.TrimSpace(n.Email), n.Papel)
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
	u.id, u.secretaria_id, u.unidade_id, u.nome, u.email, u.papel,
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
	err := rows.Scan(&u.ID, &u.SecretariaID, &u.UnidadeID, &u.Nome, &u.Email, &u.Papel,
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
// (22/09): nome, email, papel e as grades em que atendente/recepcionista estão alocados
// (23/09, substitui a FK única por uma lista). Senha removida (22/10, login por senha
// removido por completo — ver skill auth-conecta-cidades).
type AtualizacaoUsuario struct {
	Nome     string
	Email    string
	Papel    domain.Papel
	GradeIDs []string
}

func (r *Repo) AtualizarUsuario(ctx context.Context, usuarioID string, a AtualizacaoUsuario) (*domain.Usuario, error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		update usuarios set nome = $2, email = $3, papel = $4
		where id = $1
		returning `+colunasUsuario,
		usuarioID, strings.TrimSpace(a.Nome), strings.TrimSpace(a.Email), a.Papel)
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

// ErrVinculoExternoConflitante: o usuário achado por e-mail já tem um (external_issuer,
// external_sub) DIFERENTE gravado — recusa o login em vez de rebindar silenciosamente
// (achado por revisão de segurança, 22/10: sem essa checagem, um token com o e-mail de
// outra pessoa — mesmo que legitimamente assinado pelo emissor confiado — conseguiria
// sequestrar a conta local dela trocando o vínculo externo por baixo dos panos).
var ErrVinculoExternoConflitante = errors.New("usuario ja vinculado a outra identidade externa")

// BuscarUsuarioPorExternalSub resolve pelo `(issuer, sub)` do token do portal — identidade
// preferida no login via SSO (skill auth-conecta-cidades: "use sub ou preferred_username
// como identificador", já que `cpf` é opcional e normalmente ausente nos tokens de
// servidor). ESCOPADO POR EMISSOR (22/10, corrigido por revisão de segurança): `sub` é um ID
// interno POR MUNICÍPIO na Conecta Cidades — duas prefeituras diferentes podem ter um
// servidor com o mesmo `sub` cada uma; comparar sem o emissor juntava contas de tenants
// diferentes.
func (r *Repo) BuscarUsuarioPorExternalSub(ctx context.Context, issuer, externalSub string) (*domain.Usuario, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasUsuario+` from usuarios where external_issuer = $1 and external_sub = $2`, issuer, externalSub)
	return escanearUsuario(row)
}

// DadosPortal — o que o token do portal (já verificado) resolve sobre a pessoa, pronto pra
// gravar/atualizar. SecretariaID/UnidadeID já vêm resolvidos (via mapeamento_externo) pelo
// controller — este repository não decide isso. EmailVerificado (22/10, revisão de
// segurança) controla se o fallback de busca por e-mail pode rodar — ver UpsertUsuarioPortal.
// PrefeituraID (22/10, revisão de segurança) é a prefeitura já resolvida pelo `iss` do token —
// escopa o próprio fallback por e-mail, pra não rebindar uma conta de outra prefeitura (ver
// buscarUsuarioPorEmailNaPrefeitura).
type DadosPortal struct {
	ExternalIssuer  string
	ExternalSub     string
	Nome            string
	Email           string
	EmailVerificado bool
	Papel           domain.Papel
	SecretariaID    *string
	UnidadeID       *string
	PrefeituraID    string
}

// UpsertUsuarioPortal implementa o passo 6 da skill auth-conecta-cidades (seção 4): busca por
// `(issuer, external_sub)` primeiro; existe = atualiza nome/papel/vínculo (papel é
// REAPLICADO a cada entrada — uma promoção/rebaixamento na plataforma vale no próximo
// acesso, sem esperar nada expirar); não existe = tenta por e-mail (só se `EmailVerificado`
// — ver aviso abaixo); continua sem achar = cria, sem aprovação prévia, com `senha_hash`
// nulo (login por SSO não usa senha).
//
// Travas de segurança no fallback por e-mail (22/10, revisão de segurança — a 3ª veio de uma
// segunda rodada de revisão, depois do commit/push da primeira):
//  1. só roda se `d.EmailVerificado` — um e-mail não confirmado pelo próprio emissor não é
//     base confiável pra herdar o acesso de uma conta local já existente.
//  2. se o usuário achado por e-mail já tiver um vínculo externo DIFERENTE gravado, a função
//     recusa (`ErrVinculoExternoConflitante`) em vez de sobrescrever silenciosamente —
//     nunca "rebinda" uma identidade externa já estabelecida.
//  3. a própria busca por e-mail é escopada pela prefeitura já resolvida (ver
//     `buscarUsuarioPorEmailNaPrefeitura`) e nunca acha conta `admin` — sem isso, um e-mail
//     verificado por QUALQUER prefeitura cadastrada conseguia sequestrar/realocar uma conta
//     pré-provisionada de OUTRA prefeitura (trancando o dono legítimo fora na vez seguinte).
//
// Usuário desativado não é recriado nem reativado automaticamente aqui — a função só
// atualiza os dados; quem decide se o acesso é permitido (checando `Ativo`) é o controller,
// igual já faz pro login por senha.
func (r *Repo) UpsertUsuarioPortal(ctx context.Context, d DadosPortal) (*domain.Usuario, error) {
	existente, err := r.BuscarUsuarioPorExternalSub(ctx, d.ExternalIssuer, d.ExternalSub)
	// Fallback por e-mail só roda com EmailVerificado=true (ver aviso de segurança acima) —
	// sem isso, `existente` continua nil e `err` continua ErrNaoEncontrado, caindo direto no
	// INSERT abaixo, exatamente como se o fallback nem existisse.
	if errors.Is(err, ErrNaoEncontrado) && d.EmailVerificado {
		existente, err = r.buscarUsuarioPorEmailNaPrefeitura(ctx, d.Email, d.PrefeituraID)
		if err == nil && existente.ExternalSub != nil &&
			(*existente.ExternalSub != d.ExternalSub || existente.ExternalIssuer == nil || *existente.ExternalIssuer != d.ExternalIssuer) {
			return nil, ErrVinculoExternoConflitante
		}
	}
	if err != nil && !errors.Is(err, ErrNaoEncontrado) {
		return nil, err
	}

	if existente != nil {
		row := r.Pool.QueryRow(ctx, `
			update usuarios
			set nome = $2, papel = $3, secretaria_id = $4, unidade_id = $5, external_sub = $6, external_issuer = $7, ultimo_acesso_em = now()
			where id = $1
			returning `+colunasUsuario,
			existente.ID, strings.TrimSpace(d.Nome), d.Papel, d.SecretariaID, d.UnidadeID, d.ExternalSub, d.ExternalIssuer)
		return escanearUsuario(row)
	}

	row := r.Pool.QueryRow(ctx, `
		insert into usuarios (secretaria_id, unidade_id, nome, email, papel, external_sub, external_issuer, ativo, ultimo_acesso_em)
		values ($1, $2, $3, $4, $5, $6, $7, true, now())
		returning `+colunasUsuario,
		d.SecretariaID, d.UnidadeID, strings.TrimSpace(d.Nome), strings.TrimSpace(d.Email), d.Papel, d.ExternalSub, d.ExternalIssuer)
	return escanearUsuario(row)
}
