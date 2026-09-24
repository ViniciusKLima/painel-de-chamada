package repository

import (
	"context"
	"errors"
	"strings"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/util"
)

const colunasGrade = `
	id, unidade_id, servico, nome, sla_chegada_minutos, sla_atendimento_minutos,
	duracao_chamada_painel_segundos, duracao_chamada_painel_fila_segundos,
	repeticoes_chamada, intervalo_repeticao_segundos, cooldown_rechamada_minutos,
	permite_encaixe_recepcao, ativo, criado_em, (excluido_em is not null)
`

// colunasGradeComAlias — mesma lista, prefixada com `g.`, pra usar em joins onde outra
// tabela também tem colunas homônimas (ex. `usuario_grades`/`agendamentos` também têm `id`).
const colunasGradeComAlias = `
	g.id, g.unidade_id, g.servico, g.nome, g.sla_chegada_minutos, g.sla_atendimento_minutos,
	g.duracao_chamada_painel_segundos, g.duracao_chamada_painel_fila_segundos,
	g.repeticoes_chamada, g.intervalo_repeticao_segundos, g.cooldown_rechamada_minutos,
	g.permite_encaixe_recepcao, g.ativo, g.criado_em, (g.excluido_em is not null)
`

const colunasGuiche = `id, grade_id, nome, ativo, tipo, andar, capacidade, ocupado_por_usuario_id, ocupado_em`

// janelaOcupacaoGuiche: um guichê sem heartbeat mais recente que isso é tratado como livre
// de novo automaticamente nas consultas de listagem, mesmo que a coluna ainda não tenha sido
// limpa — cobre aba fechada/crash sem logout, sem precisar de job de limpeza (22/09, ver
// skill regras-negocio-fila).
const janelaOcupacaoGuiche = `interval '15 seconds'`

func escanearLinhaGrade(row interface {
	Scan(dest ...any) error
}, g *domain.Grade) error {
	return row.Scan(&g.ID, &g.UnidadeID, &g.Servico, &g.Nome, &g.SLAChegadaMinutos, &g.SLAAtendimentoMinutos,
		&g.DuracaoChamadaPainelSegundos, &g.DuracaoChamadaPainelFilaSegundos,
		&g.RepeticoesChamada, &g.IntervaloRepeticaoSegundos, &g.CooldownRechamadaMinutos,
		&g.PermiteEncaixeRecepcao, &g.Ativo, &g.CriadoEm, &g.Excluida)
}

func escanearGrade(row pgx.Row) (*domain.Grade, error) {
	var g domain.Grade
	err := escanearLinhaGrade(row, &g)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) BuscarGradePorID(ctx context.Context, id string) (*domain.Grade, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasGrade+` from grades where id = $1`, id)
	return escanearGrade(row)
}

// ListarGradesPorUnidade é a tela "Grade de horário" do gestor — todas as filas/serviços
// daquela unidade, ativas ou não (excluídas, essas sim, somem — ver ExcluirGrade).
func (r *Repo) ListarGradesPorUnidade(ctx context.Context, unidadeID string) ([]domain.Grade, error) {
	rows, err := r.Pool.Query(ctx, `select `+colunasGrade+` from grades where unidade_id = $1 and excluido_em is null order by nome`, unidadeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []domain.Grade
	for rows.Next() {
		var g domain.Grade
		if err := escanearLinhaGrade(rows, &g); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, rows.Err()
}

// ListarGradesPorSecretaria (22/09): a tela "Grade de horário" do gestor lista todas as
// filas/serviços de TODAS as unidades da secretaria (gestor deixou de ser vinculado a uma
// unidade só, ver skill modelo-dados) — join com `unidades` pra filtrar pela secretaria.
//
// Inclui grades EXCLUÍDAS (23/09, auditoria de reativação) — diferente de todo outro
// consumidor de grade (recepção, painel, importação, `ResolverOuCriarGrade`), que continuam
// filtrando `excluido_em is null` porque uma grade excluída nunca deveria voltar a receber
// agendamento/aparecer pra operação sozinha. Esta é a ÚNICA função que devolve excluídas de
// propósito — é a tela de gestão que precisa mostrá-las (numa aba "Excluídas" separada) pra
// ter como reativar; o campo `Excluida` no retorno é o que o frontend usa pra decidir a aba.
func (r *Repo) ListarGradesPorSecretaria(ctx context.Context, secretariaID string) ([]domain.Grade, error) {
	rows, err := r.Pool.Query(ctx, `
		select g.id, g.unidade_id, g.servico, g.nome, g.sla_chegada_minutos, g.sla_atendimento_minutos,
			g.duracao_chamada_painel_segundos, g.duracao_chamada_painel_fila_segundos,
			g.repeticoes_chamada, g.intervalo_repeticao_segundos, g.cooldown_rechamada_minutos,
			g.permite_encaixe_recepcao, g.ativo, g.criado_em, (g.excluido_em is not null)
		from grades g
		join unidades u on u.id = g.unidade_id
		where u.secretaria_id = $1
		order by g.nome
	`, secretariaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var grades []domain.Grade
	for rows.Next() {
		var g domain.Grade
		if err := escanearLinhaGrade(rows, &g); err != nil {
			return nil, err
		}
		grades = append(grades, g)
	}
	return grades, rows.Err()
}

// GradeComUnidade — usada na aba "Painéis" da configuração da prefeitura (admin, 23/09):
// precisa do nome da unidade junto (um pai só de grades não diz onde cada uma fica).
type GradeComUnidade struct {
	domain.Grade
	UnidadeNome string
}

// ListarGradesAtivasPorPrefeitura (23/09) — todas as grades ATIVAS de TODAS as unidades de
// TODAS as secretarias de uma prefeitura, pra central de painéis do admin dentro da
// configuração daquela prefeitura ("dashboard e usuarios internos... dentro da configuração
// de cada prefeitura" — o mesmo vale pra painéis, item 4 do pedido).
func (r *Repo) ListarGradesAtivasPorPrefeitura(ctx context.Context, prefeituraID string) ([]GradeComUnidade, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasGradeComAlias+`, un.nome
		from grades g
		join unidades un on un.id = g.unidade_id
		join secretarias s on s.id = un.secretaria_id
		where s.prefeitura_id = $1 and g.ativo = true and g.excluido_em is null
		order by un.nome, g.nome
	`, prefeituraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []GradeComUnidade
	for rows.Next() {
		var g GradeComUnidade
		if err := rows.Scan(&g.ID, &g.UnidadeID, &g.Servico, &g.Nome, &g.SLAChegadaMinutos, &g.SLAAtendimentoMinutos,
			&g.DuracaoChamadaPainelSegundos, &g.DuracaoChamadaPainelFilaSegundos,
			&g.RepeticoesChamada, &g.IntervaloRepeticaoSegundos, &g.CooldownRechamadaMinutos,
			&g.PermiteEncaixeRecepcao, &g.Ativo, &g.CriadoEm, &g.UnidadeNome); err != nil {
			return nil, err
		}
		lista = append(lista, g)
	}
	return lista, rows.Err()
}

// ResolverOuCriarGrade casa por serviço (normalizado) dentro da unidade, criando
// automaticamente se ainda não existir — mesmo padrão já usado pra unidade em
// ResolverOuCriarUnidade (skill importacao-planilha): a grade nova não deve travar o
// import esperando cadastro manual prévio. Serviço vazio cai no bucket "Geral".
//
// `nome` (22/09, ajustado): vem da coluna "Grade de horários" da planilha — é o nome de
// exibição de verdade, DIFERENTE do serviço (que é só a chave de roteamento). Se vier vazio,
// cai no próprio serviço como nome — mas isso só acontece pra planilhas sem essa coluna
// preenchida, não é o caminho normal.
func (r *Repo) ResolverOuCriarGrade(ctx context.Context, unidadeID, servico, nome string) (*domain.Grade, error) {
	servico = strings.TrimSpace(servico)
	if servico == "" {
		servico = "Geral"
	}
	nome = strings.TrimSpace(nome)
	if nome == "" {
		nome = servico
	}
	existentes, err := r.ListarGradesPorUnidade(ctx, unidadeID)
	if err != nil {
		return nil, err
	}
	alvo := util.NormalizarNome(servico)
	for _, g := range existentes {
		if util.NormalizarNome(g.Servico) == alvo {
			return &g, nil
		}
	}

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		insert into grades (unidade_id, servico, nome)
		values ($1, $2, $3)
		returning `+colunasGrade, unidadeID, servico, nome)
	g, err := escanearGrade(row)
	if err != nil {
		return nil, err
	}
	// Toda grade nova já nasce com 1 guichê (23/09, pedido explícito: "por padrão já quero
	// que a grade venha com 1 guichê") — sem isso, o atendente alocado nela não teria nenhum
	// guichê/sala pra escolher até o gestor lembrar de criar um manualmente.
	if _, err := tx.Exec(ctx, `insert into guiches (grade_id, nome, ativo, tipo, capacidade) values ($1, 'Guichê 1', true, 'guiche', 1)`, g.ID); err != nil {
		return nil, err
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return g, nil
}

// ExcluirGrade (23/09) — soft-delete (ver nota na migration 000008): some da tela "Grade de
// horário" e de qualquer roteamento de importação futuro, mas o histórico de agendamentos/
// chamadas daquela grade continua intacto no banco.
// ExcluirGrade marca a grade como excluída E limpa as alocações `usuario_grades` que
// apontavam pra ela (23/09, auditoria de integridade — "verifique... relacionamentos
// órfãos"). Diferente de `ativo=false` (reversível, tem botão de reativar), exclusão de
// grade não tem volta na UI — deixar o vínculo morto no banco só acumularia lixo que todo
// mundo (ListarGradesDoUsuario/GradesPorUsuario) já filtra manualmente de qualquer jeito;
// mais simples limpar na hora do que confiar em filtro em N lugares diferentes pra sempre.
func (r *Repo) ExcluirGrade(ctx context.Context, gradeID string) error {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `update grades set excluido_em = now() where id = $1`, gradeID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `delete from usuario_grades where grade_id = $1`, gradeID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ReativarGrade (23/09, auditoria de reativação) — desfaz `ExcluirGrade`. Diferente da
// exclusão, não precisa limpar/restaurar `usuario_grades`: os vínculos de quem estava alocado
// na grade já foram removidos na hora da exclusão (não dá pra "adivinhar" quem devia voltar a
// ser alocado), então uma grade reativada volta com zero atendentes/recepcionistas — o gestor
// aloca de novo quem precisar pela tela "Usuários internos", como faria com uma grade nova.
func (r *Repo) ReativarGrade(ctx context.Context, gradeID string) (*domain.Grade, error) {
	row := r.Pool.QueryRow(ctx, `
		update grades set excluido_em = null where id = $1
		returning `+colunasGrade, gradeID)
	return escanearGrade(row)
}

// AtualizarConfiguracaoGrade grava os SLAs, duração de exibição, repetição de chamada,
// cooldown de rechamada e a permissão de encaixe da recepção configurados pelo gestor na
// tela de configurações da grade.
func (r *Repo) AtualizarConfiguracaoGrade(ctx context.Context, gradeID string, slaChegada, slaAtendimento, duracaoNormal, duracaoFila, repeticoesChamada, intervaloRepeticaoSegundos, cooldownRechamadaMinutos int, permiteEncaixeRecepcao bool) (*domain.Grade, error) {
	row := r.Pool.QueryRow(ctx, `
		update grades set
			sla_chegada_minutos = $2,
			sla_atendimento_minutos = $3,
			duracao_chamada_painel_segundos = $4,
			duracao_chamada_painel_fila_segundos = $5,
			repeticoes_chamada = $6,
			intervalo_repeticao_segundos = $7,
			cooldown_rechamada_minutos = $8,
			permite_encaixe_recepcao = $9
		where id = $1
		returning `+colunasGrade,
		gradeID, slaChegada, slaAtendimento, duracaoNormal, duracaoFila, repeticoesChamada, intervaloRepeticaoSegundos, cooldownRechamadaMinutos, permiteEncaixeRecepcao)
	return escanearGrade(row)
}

// AtualizarNomeEAtivoGrade — o gestor pode renomear a grade (o nome de exibição pode
// divergir do texto bruto do serviço) e ativar/desativar (uma grade desativada não recebe
// mais agendamentos novos via import, mas o histórico continua intacto).
func (r *Repo) AtualizarNomeEAtivoGrade(ctx context.Context, gradeID, nome string, ativo bool) (*domain.Grade, error) {
	row := r.Pool.QueryRow(ctx, `
		update grades set nome = $2, ativo = $3 where id = $1
		returning `+colunasGrade, gradeID, strings.TrimSpace(nome), ativo)
	return escanearGrade(row)
}

type NovoGuicheOuSala struct {
	Nome       string
	Tipo       domain.TipoGuiche
	Andar      *string
	Capacidade int
}

func (r *Repo) CriarGuiche(ctx context.Context, gradeID string, n NovoGuicheOuSala) (*domain.Guiche, error) {
	row := r.Pool.QueryRow(ctx, `
		insert into guiches (grade_id, nome, ativo, tipo, andar, capacidade)
		values ($1, $2, true, $3, $4, $5)
		returning `+colunasGuiche,
		gradeID, strings.TrimSpace(n.Nome), n.Tipo, n.Andar, n.Capacidade)
	var g domain.Guiche
	if err := escanearLinhaGuiche(row, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// AtualizarAtivoGuiche liga/desliga um guichê — não existe exclusão de verdade, um guichê
// com chamadas no histórico não pode sumir da tabela `chamadas` (integridade referencial).
func (r *Repo) AtualizarAtivoGuiche(ctx context.Context, gradeID, guicheID string, ativo bool) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `
		update guiches set ativo = $3 where id = $1 and grade_id = $2
	`, guicheID, gradeID, ativo)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type AtualizacaoGuiche struct {
	Nome       string
	Tipo       domain.TipoGuiche
	Andar      *string
	Capacidade int
	Ativo      bool
}

// AtualizarGuiche grava a edição completa do bloco (nome, tipo guichê/sala, andar,
// capacidade, ativo) feita pelo gestor na tela de configurações da grade.
func (r *Repo) AtualizarGuiche(ctx context.Context, gradeID, guicheID string, a AtualizacaoGuiche) (*domain.Guiche, error) {
	row := r.Pool.QueryRow(ctx, `
		update guiches set nome = $3, tipo = $4, andar = $5, capacidade = $6, ativo = $7
		where id = $1 and grade_id = $2
		returning `+colunasGuiche,
		guicheID, gradeID, strings.TrimSpace(a.Nome), a.Tipo, a.Andar, a.Capacidade, a.Ativo)
	var g domain.Guiche
	if err := escanearLinhaGuiche(row, &g); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNaoEncontrado
		}
		return nil, err
	}
	return &g, nil
}

var ErrGuicheOcupado = errors.New("guiche ocupado por outro atendente")

// OcuparGuiche registra que `usuarioID` está atendendo naquele guichê agora — chamado ao
// escolher o guichê no modal (22/09) e reafirmado periodicamente (heartbeat) enquanto o
// atendente segue nele, pra outros atendentes verem em tempo real quais guichês já têm
// alguém. Sucede se o guichê estiver livre, já for do próprio usuário (heartbeat/reconexão),
// ou se o heartbeat anterior estiver velho demais (`janelaOcupacaoGuiche` — dono anterior
// sumiu sem liberar). Falha com ErrGuicheOcupado se outro atendente está lá agora de
// verdade.
func (r *Repo) OcuparGuiche(ctx context.Context, gradeID, guicheID, usuarioID string) (*domain.Guiche, error) {
	row := r.Pool.QueryRow(ctx, `
		update guiches
		set ocupado_por_usuario_id = $3, ocupado_em = now()
		where id = $1 and grade_id = $2
		  and (ocupado_por_usuario_id is null or ocupado_por_usuario_id = $3 or ocupado_em < now() - `+janelaOcupacaoGuiche+`)
		returning `+colunasGuiche,
		guicheID, gradeID, usuarioID)
	var g domain.Guiche
	err := escanearLinhaGuiche(row, &g)
	if errors.Is(err, pgx.ErrNoRows) {
		if _, errBusca := r.BuscarGuichePorID(ctx, gradeID, guicheID); errBusca != nil {
			return nil, ErrNaoEncontrado
		}
		return nil, ErrGuicheOcupado
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

// LiberarGuiche solta o guichê explicitamente (botão "Sair"/trocar de guichê) — só o próprio
// dono libera; alguém tentando liberar o guichê de outra pessoa é um no-op silencioso.
func (r *Repo) LiberarGuiche(ctx context.Context, gradeID, guicheID, usuarioID string) error {
	_, err := r.Pool.Exec(ctx, `
		update guiches set ocupado_por_usuario_id = null, ocupado_em = null
		where id = $1 and grade_id = $2 and ocupado_por_usuario_id = $3
	`, guicheID, gradeID, usuarioID)
	return err
}

func escanearLinhaGuiche(row interface {
	Scan(dest ...any) error
}, g *domain.Guiche) error {
	return row.Scan(&g.ID, &g.GradeID, &g.Nome, &g.Ativo, &g.Tipo, &g.Andar, &g.Capacidade, &g.OcupadoPorUsuarioID, &g.OcupadoEm)
}

// colunasGuicheComOcupacao é usada só nas listagens (não em insert/update returning): junta
// o nome de quem ocupa e aplica a janela de heartbeat — um registro sem heartbeat recente
// aparece como livre (usuário/nome/horário nulos), mesmo que a coluna bruta ainda não tenha
// sido limpa. `g.*` corresponde 1:1 a colunasGuiche, então dá pra reaproveitar
// escanearLinhaGuiche pros 7 primeiros campos + mais um scan manual do nome.
func escanearLinhaGuicheComOcupacao(row interface {
	Scan(dest ...any) error
}, g *domain.Guiche) error {
	return row.Scan(&g.ID, &g.GradeID, &g.Nome, &g.Ativo, &g.Tipo, &g.Andar, &g.Capacidade,
		&g.OcupadoPorUsuarioID, &g.OcupadoEm, &g.OcupadoPorNome)
}

// escanearLinhaGuicheComOcupacaoEExclusao — mesma coisa, mais o campo `Excluida` (23/09,
// auditoria de reativação) — só usada por `selectGuichesComOcupacaoIncluindoExcluidos`.
func escanearLinhaGuicheComOcupacaoEExclusao(row interface {
	Scan(dest ...any) error
}, g *domain.Guiche) error {
	return row.Scan(&g.ID, &g.GradeID, &g.Nome, &g.Ativo, &g.Tipo, &g.Andar, &g.Capacidade,
		&g.OcupadoPorUsuarioID, &g.OcupadoEm, &g.OcupadoPorNome, &g.Excluida)
}

const selectGuichesComOcupacao = `
	select g.id, g.grade_id, g.nome, g.ativo, g.tipo, g.andar, g.capacidade,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then g.ocupado_por_usuario_id end,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then g.ocupado_em end,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then u.nome end
	from guiches g
	left join usuarios u on u.id = g.ocupado_por_usuario_id
	where g.excluido_em is null
`

// selectGuichesComOcupacaoIncluindoExcluidos — mesma consulta, SEM o filtro de exclusão, mais
// a coluna `excluida` no fim (23/09, auditoria de reativação). Usada só por
// `ListarTodosGuiches` (a tela de gestão precisa mostrar guichês excluídos pra ter como
// reativar) — todo outro consumidor de guichê continua filtrando excluídos, ver nota em
// `ListarGradesPorSecretaria` sobre o mesmo princípio pra grades.
const selectGuichesComOcupacaoIncluindoExcluidos = `
	select g.id, g.grade_id, g.nome, g.ativo, g.tipo, g.andar, g.capacidade,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then g.ocupado_por_usuario_id end,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then g.ocupado_em end,
		case when g.ocupado_em > now() - ` + janelaOcupacaoGuiche + ` then u.nome end,
		(g.excluido_em is not null)
	from guiches g
	left join usuarios u on u.id = g.ocupado_por_usuario_id
	where true
`

func (r *Repo) ListarGuichesAtivos(ctx context.Context, gradeID string) ([]domain.Guiche, error) {
	return r.ListarGuichesAtivosPorGrades(ctx, []string{gradeID})
}

// ListarGuichesAtivosPorGrades (23/09) — consolidado pra várias grades numa passada só, ver
// nota em ListarSalaEsperaPorGrades. Cada guichê continua pertencendo a UMA grade só (não
// existe "guichê compartilhado entre grades") — isso só evita N requisições separadas
// quando o atendente tem N grades.
func (r *Repo) ListarGuichesAtivosPorGrades(ctx context.Context, gradeIDs []string) ([]domain.Guiche, error) {
	rows, err := r.Pool.Query(ctx, selectGuichesComOcupacao+`
		and g.grade_id = any($1) and g.ativo = true order by g.nome
	`, gradeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guiches []domain.Guiche
	for rows.Next() {
		var g domain.Guiche
		if err := escanearLinhaGuicheComOcupacao(rows, &g); err != nil {
			return nil, err
		}
		guiches = append(guiches, g)
	}
	return guiches, rows.Err()
}

// ListarTodosGuiches inclui os inativos E os excluídos (23/09) — a listagem pra atendente usa
// ListarGuichesAtivos; esta é pra tela do gestor, que precisa poder ligar/desligar um guichê
// (`ativo`) E reativar um excluído (`excluido_em`).
func (r *Repo) ListarTodosGuiches(ctx context.Context, gradeID string) ([]domain.Guiche, error) {
	rows, err := r.Pool.Query(ctx, selectGuichesComOcupacaoIncluindoExcluidos+`
		and g.grade_id = $1 order by g.nome
	`, gradeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var guiches []domain.Guiche
	for rows.Next() {
		var g domain.Guiche
		if err := escanearLinhaGuicheComOcupacaoEExclusao(rows, &g); err != nil {
			return nil, err
		}
		guiches = append(guiches, g)
	}
	return guiches, rows.Err()
}

var ErrUltimoGuiche = errors.New("grade precisa de pelo menos um guiche ou sala")

// ExcluirGuiche (23/09) — soft-delete, mesma lógica de ExcluirGrade. Bloqueado (ErrUltimoGuiche)
// se for o único guichê/sala não-excluído da grade — pedido explícito: "é obrigatório ter
// pelo menos 1 guichê ou sala". A checagem de contagem e o update acontecem na MESMA query
// (atômico) pra não ter condição de corrida entre "contar quantos existem" e "excluir".
func (r *Repo) ExcluirGuiche(ctx context.Context, gradeID, guicheID string) error {
	tag, err := r.Pool.Exec(ctx, `
		update guiches set excluido_em = now()
		where id = $1 and grade_id = $2 and excluido_em is null
		  and (select count(*) from guiches where grade_id = $2 and excluido_em is null) > 1
	`, guicheID, gradeID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() > 0 {
		return nil
	}
	if _, err := r.BuscarGuichePorID(ctx, gradeID, guicheID); err != nil {
		return err
	}
	return ErrUltimoGuiche
}

// ReativarGuiche (23/09, auditoria de reativação) — desfaz `ExcluirGuiche`. Volta com
// `ativo` do jeito que estava antes de ser excluído (a exclusão não mexe em `ativo`, só em
// `excluido_em`) — se o gestor tinha desativado o guichê antes de excluir, ele volta
// desativado; precisa ligar separadamente se quiser usá-lo de novo.
func (r *Repo) ReativarGuiche(ctx context.Context, gradeID, guicheID string) (*domain.Guiche, error) {
	row := r.Pool.QueryRow(ctx, `
		update guiches set excluido_em = null where id = $1 and grade_id = $2
		returning `+colunasGuiche, guicheID, gradeID)
	var g domain.Guiche
	err := escanearLinhaGuiche(row, &g)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (r *Repo) BuscarGuichePorID(ctx context.Context, gradeID, guicheID string) (*domain.Guiche, error) {
	row := r.Pool.QueryRow(ctx, `
		select `+colunasGuiche+` from guiches where id = $1 and grade_id = $2
	`, guicheID, gradeID)
	var g domain.Guiche
	err := escanearLinhaGuiche(row, &g)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}
