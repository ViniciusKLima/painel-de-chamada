package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"painel-chamada-backend/internal/domain"
)

// Erros específicos da regra de posse/cooldown de chamada (20/09) — ver skill
// regras-negocio-fila.
var (
	ErrChamadaPendente = errors.New("usuario tem chamada pendente ainda nao rechamada")
	ErrExistemOrfaos   = errors.New("existem chamados orfaos aguardando serem assumidos")
	ErrNaoEhOrfao      = errors.New("agendamento nao esta em estado de chamado orfao")
	ErrCooldownAtivo   = errors.New("cooldown de rechamada ainda ativo")
)

const colunasAgendamento = `
	id, unidade_id, grade_id, lote_id, protocolo, nome_cidadao, cpf, telefone, servico, grade_horario,
	tipo, horario_previsto, chegada_em, prioridade, guiche_id, status,
	atendido_em, ausente_em, cancelado_em, motivo_cancelamento, criado_em
`

// colunasAgendamentoComAlias é a mesma lista prefixada com "a." — necessária só quando a
// query junta `agendamentos` com outras tabelas que têm colunas de mesmo nome (guiches e
// usuarios também têm `id`, grades também tem `unidade_id`), senão o Postgres recusa por
// ambiguidade.
const colunasAgendamentoComAlias = `
	a.id, a.unidade_id, a.grade_id, a.lote_id, a.protocolo, a.nome_cidadao, a.cpf, a.telefone, a.servico, a.grade_horario,
	a.tipo, a.horario_previsto, a.chegada_em, a.prioridade, a.guiche_id, a.status,
	a.atendido_em, a.ausente_em, a.cancelado_em, a.motivo_cancelamento, a.criado_em
`

func escanearAgendamento(row pgx.Row) (*domain.Agendamento, error) {
	var a domain.Agendamento
	err := row.Scan(
		&a.ID, &a.UnidadeID, &a.GradeID, &a.LoteID, &a.Protocolo, &a.NomeCidadao, &a.CPF, &a.Telefone, &a.Servico, &a.GradeHorario,
		&a.Tipo, &a.HorarioPrevisto, &a.ChegadaEm, &a.Prioridade, &a.GuicheID, &a.Status,
		&a.AtendidoEm, &a.AusenteEm, &a.CanceladoEm, &a.MotivoCancelamento, &a.CriadoEm,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	return &a, err
}

func escanearAgendamentos(rows pgx.Rows) ([]domain.Agendamento, error) {
	defer rows.Close()
	var lista []domain.Agendamento
	for rows.Next() {
		var a domain.Agendamento
		if err := rows.Scan(
			&a.ID, &a.UnidadeID, &a.GradeID, &a.LoteID, &a.Protocolo, &a.NomeCidadao, &a.CPF, &a.Telefone, &a.Servico, &a.GradeHorario,
			&a.Tipo, &a.HorarioPrevisto, &a.ChegadaEm, &a.Prioridade, &a.GuicheID, &a.Status,
			&a.AtendidoEm, &a.AusenteEm, &a.CanceladoEm, &a.MotivoCancelamento, &a.CriadoEm,
		); err != nil {
			return nil, err
		}
		lista = append(lista, a)
	}
	return lista, rows.Err()
}

// ProtocoloExisteNaSecretaria checa duplicidade no escopo da SECRETARIA inteira (não só
// da unidade) — protocolo vem do sistema de origem e deve ser único ali. A constraint do
// banco é só (unidade_id, protocolo) por simplicidade de schema; esta checagem cobre o
// caso de duplicidade entre unidades diferentes da mesma secretaria. Ver skill
// importacao-planilha.
func (r *Repo) ProtocoloExisteNaSecretaria(ctx context.Context, secretariaID, protocolo string) (bool, error) {
	var existe bool
	err := r.Pool.QueryRow(ctx, `
		select exists(
			select 1 from agendamentos a
			join unidades u on u.id = a.unidade_id
			where u.secretaria_id = $1 and a.protocolo = $2
		)
	`, secretariaID, protocolo).Scan(&existe)
	return existe, err
}

type NovoAgendamento struct {
	UnidadeID          string
	GradeID            string
	LoteID             *string
	Protocolo          string
	NomeCidadao        string
	CPF                *string
	Telefone           *string
	Servico            *string
	GradeHorario       *string
	Tipo               domain.TipoAgendamento
	HorarioPrevisto    *time.Time
	Status             domain.StatusAgendamento
	CanceladoEm        *time.Time
	MotivoCancelamento *string
}

func (r *Repo) CriarAgendamento(ctx context.Context, tx pgx.Tx, n NovoAgendamento) error {
	_, err := tx.Exec(ctx, `
		insert into agendamentos (
			unidade_id, grade_id, lote_id, protocolo, nome_cidadao, cpf, telefone, servico, grade_horario,
			tipo, horario_previsto, status, cancelado_em, motivo_cancelamento
		) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
	`, n.UnidadeID, n.GradeID, n.LoteID, n.Protocolo, n.NomeCidadao, n.CPF, n.Telefone, n.Servico, n.GradeHorario,
		n.Tipo, n.HorarioPrevisto, n.Status, n.CanceladoEm, n.MotivoCancelamento)
	return err
}

// ListarParaRecepcao traz agendado (aguardando_chegada + sala_espera) do dia, mais quem
// faltou HOJE (status ausente — decisão 21/09, pedido do dono do produto: se a pessoa
// chegar atrasada depois de já ter sido marcada ausente automaticamente, a recepção precisa
// saber disso de cara). Ausente de dias anteriores fica de fora, senão a aba "Faltaram"
// acumularia o histórico inteiro do piloto em vez de só o dia corrente.
func (r *Repo) ListarParaRecepcao(ctx context.Context, unidadeID string) ([]domain.Agendamento, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasAgendamento+`
		from agendamentos
		where unidade_id = $1
		  and (
		    status in ('aguardando_chegada','sala_espera')
		    or (status = 'ausente' and coalesce(horario_previsto::date, criado_em::date) = current_date)
		  )
		order by horario_previsto asc nulls last, criado_em asc
	`, unidadeID)
	if err != nil {
		return nil, err
	}
	return escanearAgendamentos(rows)
}

// ListarSalaEspera aplica a regra de prioridade confirmada em 19/09: no_horario primeiro
// (por horario_previsto); atrasado e encaixe competem juntos depois, por chegada_em.
// Escopo por grade (22/09) — cada grade tem sua própria fila.
func (r *Repo) ListarSalaEspera(ctx context.Context, gradeID string) ([]domain.Agendamento, error) {
	return r.ListarSalaEsperaPorGrades(ctx, []string{gradeID})
}

// ListarSalaEsperaPorGrades (23/09, "atendente pode ser alocado em mais de uma grade — deve
// ver todas consolidadas, sem escolher uma") — mesma regra de prioridade de ListarSalaEspera,
// só que numa passada só pra várias grades ao mesmo tempo (evita N chamadas separadas — uma
// por grade do atendente — quando o operacional já sabe de antemão quais são). A ordenação
// de prioridade continua por linha (dentro da grade dela); quando o resultado cobre mais de
// uma grade, o frontend agrupa por unidade/grade — a ordem aqui é só a prioridade "crua".
func (r *Repo) ListarSalaEsperaPorGrades(ctx context.Context, gradeIDs []string) ([]domain.Agendamento, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasAgendamento+`
		from agendamentos
		where grade_id = any($1) and status = 'sala_espera'
		order by
			grade_id,
			case when prioridade = 'no_horario' then 0 else 1 end,
			case when prioridade = 'no_horario' then horario_previsto else chegada_em end asc
	`, gradeIDs)
	if err != nil {
		return nil, err
	}
	return escanearAgendamentos(rows)
}

// ChamadoCompartilhado é uma linha do bloco "Chamando agora" — visível a TODOS os
// atendentes da unidade (decisão do dono do produto, 20/09), com quem chamou e se já virou
// "órfão" (chamado, já rechamado pelo dono, e o dono já seguiu em frente pra outra pessoa —
// qualquer atendente ocioso deve assumir esses antes de puxar alguém novo da sala de
// espera). Ver skill regras-negocio-fila.
type ChamadoCompartilhado struct {
	Agendamento         domain.Agendamento
	GuicheNome          string
	ChamadoPorUsuarioID string
	ChamadoPorNome      string
	PrimeiraChamadaEm   time.Time
	UltimaChamadaEm     time.Time
	TotalChamadasDoDono int
	EhOrfao             bool
}

// condicaoOrfao (fragmento reaproveitado): dado que "ultima" é a última chamada (usuario_id,
// chamado_em) e "cnt" é quantas dessas chamadas pertencem a esse mesmo usuario_id, um
// agendamento é órfão quando o dono já rechamou pelo menos uma vez (cnt.total >= 2) E esse
// mesmo dono já fez uma chamada MAIS RECENTE em outro agendamento (ou seja, seguiu em
// frente e abandonou este).
const condicaoOrfao = `
	cnt.total >= 2
	and exists (
		select 1 from chamadas c_mov
		where c_mov.usuario_id = ultima.usuario_id and c_mov.chamado_em > ultima.chamado_em
	)
`

// juncoesUltimaChamada também expõe chamada_id/guiche_id (23/09) — necessário pra
// UltimasChamadas dar uma linha só por agendamento (dedup) em vez de uma por chamada.
const juncoesUltimaChamada = `
	join lateral (
		select c.id as chamada_id, c.guiche_id, c.usuario_id, c.chamado_em
		from chamadas c where c.agendamento_id = a.id
		order by c.chamado_em desc limit 1
	) ultima on true
	join lateral (
		select count(*) as total from chamadas c2
		where c2.agendamento_id = a.id and c2.usuario_id = ultima.usuario_id
	) cnt on true
`

// ListarChamadosCompartilhados alimenta o bloco "Chamando agora" da tela do atendente —
// TODOS os status='chamado' da grade, não só os de quem está olhando a tela. Escopo por
// grade (22/09) — cada grade é uma fila independente.
func (r *Repo) ListarChamadosCompartilhados(ctx context.Context, gradeID string) ([]ChamadoCompartilhado, error) {
	return r.ListarChamadosCompartilhadosPorGrades(ctx, []string{gradeID})
}

// ListarChamadosCompartilhadosPorGrades (23/09) — mesma consulta de ListarChamadosCompartilhados,
// consolidada pra várias grades numa passada só (ver nota em ListarSalaEsperaPorGrades).
func (r *Repo) ListarChamadosCompartilhadosPorGrades(ctx context.Context, gradeIDs []string) ([]ChamadoCompartilhado, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasAgendamentoComAlias+`, g.nome, ultima.usuario_id, u.nome,
		       primeira.chamado_em, ultima.chamado_em, cnt.total,
		       (`+condicaoOrfao+`) as eh_orfao
		from agendamentos a
		join guiches g on g.id = a.guiche_id
		join usuarios u on u.id = (select c3.usuario_id from chamadas c3 where c3.agendamento_id = a.id order by c3.chamado_em desc limit 1)
		`+juncoesUltimaChamada+`
		join lateral (
			select min(c4.chamado_em) as chamado_em from chamadas c4 where c4.agendamento_id = a.id
		) primeira on true
		where a.grade_id = any($1) and a.status = 'chamado'
		order by primeira.chamado_em asc
	`, gradeIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ChamadoCompartilhado
	for rows.Next() {
		var item ChamadoCompartilhado
		a := &item.Agendamento
		if err := rows.Scan(
			&a.ID, &a.UnidadeID, &a.GradeID, &a.LoteID, &a.Protocolo, &a.NomeCidadao, &a.CPF, &a.Telefone, &a.Servico, &a.GradeHorario,
			&a.Tipo, &a.HorarioPrevisto, &a.ChegadaEm, &a.Prioridade, &a.GuicheID, &a.Status,
			&a.AtendidoEm, &a.AusenteEm, &a.CanceladoEm, &a.MotivoCancelamento, &a.CriadoEm,
			&item.GuicheNome, &item.ChamadoPorUsuarioID, &item.ChamadoPorNome,
			&item.PrimeiraChamadaEm, &item.UltimaChamadaEm, &item.TotalChamadasDoDono, &item.EhOrfao,
		); err != nil {
			return nil, err
		}
		lista = append(lista, item)
	}
	return lista, rows.Err()
}

// AguardandoExibicao é uma linha do "modo aguardando" do painel de TV (23/09) — gente já
// chamada, ainda não atendida/ausente, mas que já saiu do destaque normal do painel (a
// exibição de 30s/15s expirou) e ainda não virou órfã. Ver ListarAguardandoExibicao.
type AguardandoExibicao struct {
	Protocolo         string
	Nome              string
	GuicheNome        string
	PrimeiraChamadaEm time.Time
}

// ListarAguardandoExibicao alimenta o "modo aguardando" do painel de TV: pedido do dono do
// produto (23/09) pra parar de esconder na tabela quem ainda está sendo esperado por um
// atendente. Reaproveita exatamente a mesma condicaoOrfao usada em ChamarProximo/
// ListarChamadosCompartilhados/UltimasChamadas — quem já é órfão (dono já seguiu em frente
// pra outra pessoa) explicitamente NÃO entra aqui ("se estiver orfão nao fica no painel"),
// só quem ainda está genuinamente dentro do que o próprio atendente está esperando.
// Ordenado pela chamada mais antiga primeiro (quem espera há mais tempo aparece primeiro),
// limitado a `limite` (painel mostra no máximo 4 blocos, 23/09). Chamado pelo handler só quando não
// há nenhuma chamada nova ativa em destaque (ver GetPainel) — nunca compete com a exibição
// normal, só ocupa o espaço quando ele está livre.
func (r *Repo) ListarAguardandoExibicao(ctx context.Context, gradeID string, limite int) ([]AguardandoExibicao, error) {
	rows, err := r.Pool.Query(ctx, `
		select a.protocolo, a.nome_cidadao, g.nome, primeira.chamado_em
		from agendamentos a
		`+juncoesUltimaChamada+`
		join guiches g on g.id = ultima.guiche_id
		join lateral (
			select min(c4.chamado_em) as chamado_em from chamadas c4 where c4.agendamento_id = a.id
		) primeira on true
		where a.grade_id = $1 and a.status = 'chamado' and not (`+condicaoOrfao+`)
		order by primeira.chamado_em asc
		limit $2
	`, gradeID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []AguardandoExibicao
	for rows.Next() {
		var ag AguardandoExibicao
		if err := rows.Scan(&ag.Protocolo, &ag.Nome, &ag.GuicheNome, &ag.PrimeiraChamadaEm); err != nil {
			return nil, err
		}
		lista = append(lista, ag)
	}
	return lista, rows.Err()
}

// TemChamadaPendenteNaoRechamada: true se esse usuário tem uma chamada ativa (status
// 'chamado') cujo cooldown, contado da ÚLTIMA chamada que ele fez nela, ainda não passou —
// enquanto isso for true, ele não pode chamar mais ninguém.
//
// Correção de regra (23/09, pedido explícito do dono do produto): antes essa checagem só
// olhava pra cnt.total = 1 (ou seja, só bloqueava ANTES da primeira rechamada — depois de
// rechamar uma vez, o atendente já ficava livre pra chamar outra pessoa na hora, mesmo com
// o cooldown da rechamada seguinte ainda contando na tela). Relatado como errado: "se eu
// estou chamando alguém nesse momento e contando os 3min, eu não posso chamar outra pessoa
// até o prazo terminar" — ou seja, o gate certo é o COOLDOWN em si (o mesmo que já governa
// quando dá pra rechamar), não quantas vezes já rechamou. Agora: bloqueado enquanto
// `now() < ultima_chamada + cooldown`, não importa se já foi a 1ª ou a 3ª chamada nesse
// agendamento. Só depois que o cooldown da última chamada passar é que o atendente fica
// livre — nesse ponto ele escolhe entre rechamar (reabre um novo cooldown) ou chamar outra
// pessoa (abandona esta, que vira órfã — mecânica inalterada, ver condicaoOrfao). Escopo por
// grade (22/09).
func (r *Repo) TemChamadaPendenteNaoRechamada(ctx context.Context, gradeID, usuarioID string, cooldownMinutos int) (bool, error) {
	var existe bool
	err := r.Pool.QueryRow(ctx, `
		select exists(
			select 1 from agendamentos a
			`+juncoesUltimaChamada+`
			where a.grade_id = $1 and a.status = 'chamado' and ultima.usuario_id = $2
			and now() < ultima.chamado_em + ($3 * interval '1 minute')
		)
	`, gradeID, usuarioID, cooldownMinutos).Scan(&existe)
	return existe, err
}

// ExistemChamadosOrfaos: true se existe ao menos um chamado abandonado na grade — nesse
// caso, ninguém pode puxar gente nova da sala de espera até assumir os órfãos.
func (r *Repo) ExistemChamadosOrfaos(ctx context.Context, gradeID string) (bool, error) {
	var existe bool
	err := r.Pool.QueryRow(ctx, `
		select exists(
			select 1 from agendamentos a
			`+juncoesUltimaChamada+`
			where a.grade_id = $1 and a.status = 'chamado' and (`+condicaoOrfao+`)
		)
	`, gradeID).Scan(&existe)
	return existe, err
}

// UltimaChamadaInfo é o que os handlers precisam pra checar posse (só o dono rechama/marca
// presente) e o cooldown (esperar N minutos desde a última chamada antes de rechamar).
type UltimaChamadaInfo struct {
	UsuarioID   string
	ChamadoEm   time.Time
	TotalDoDono int
}

func (r *Repo) UltimaChamadaDe(ctx context.Context, agendamentoID string) (*UltimaChamadaInfo, error) {
	var info UltimaChamadaInfo
	err := r.Pool.QueryRow(ctx, `
		select ultima.usuario_id, ultima.chamado_em,
		       (select count(*) from chamadas c2 where c2.agendamento_id = $1 and c2.usuario_id = ultima.usuario_id)
		from (
			select usuario_id, chamado_em from chamadas c
			where c.agendamento_id = $1
			order by c.chamado_em desc limit 1
		) ultima
	`, agendamentoID).Scan(&info.UsuarioID, &info.ChamadoEm, &info.TotalDoDono)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}
	return &info, nil
}

// ListarAtendidosHoje é o bloco de consulta (só leitura) do lado direito da tela do
// atendente — quem ELE já atendeu hoje, sem ação nenhuma associada. Escopo por grade (22/09).
func (r *Repo) ListarAtendidosHoje(ctx context.Context, gradeID, usuarioID string) ([]domain.Agendamento, error) {
	return r.ListarAtendidosHojePorGrades(ctx, []string{gradeID}, usuarioID)
}

// ListarAtendidosHojePorGrades (23/09) — consolidado pra várias grades, ver nota em
// ListarSalaEsperaPorGrades.
func (r *Repo) ListarAtendidosHojePorGrades(ctx context.Context, gradeIDs []string, usuarioID string) ([]domain.Agendamento, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasAgendamento+`
		from agendamentos a
		where a.grade_id = any($1) and a.status = 'atendido'
		and a.atendido_em::date = current_date
		and (select c.usuario_id from chamadas c where c.agendamento_id = a.id order by c.chamado_em desc limit 1) = $2
		order by a.atendido_em desc
	`, gradeIDs, usuarioID)
	if err != nil {
		return nil, err
	}
	return escanearAgendamentos(rows)
}

// ListarAusentesHoje é a aba "Ausentes" do bloco compartilhado "Sala de espera" do
// atendente (22/09, movida de "meus atendimentos" — decisão do dono do produto: "vai ser
// uma tela compartilhada", já que Sala de espera já é vista por todos os atendentes da
// grade, não só por quem chamou). Lista TODA a grade, não filtra por quem chamou. Só entra
// quem teve pelo menos uma chamada de verdade — um agendamento que nunca chegou a ser
// chamado (ausente já na etapa de chegada, sem passar pela recepção) não é um "ausente de
// atendimento", é problema da recepção, não do atendente.
func (r *Repo) ListarAusentesHoje(ctx context.Context, gradeID string) ([]domain.Agendamento, error) {
	return r.ListarAusentesHojePorGrades(ctx, []string{gradeID})
}

// ListarAusentesHojePorGrades (23/09) — consolidado pra várias grades, ver nota em
// ListarSalaEsperaPorGrades.
func (r *Repo) ListarAusentesHojePorGrades(ctx context.Context, gradeIDs []string) ([]domain.Agendamento, error) {
	rows, err := r.Pool.Query(ctx, `
		select `+colunasAgendamento+`
		from agendamentos a
		where a.grade_id = any($1) and a.status = 'ausente'
		and a.ausente_em::date = current_date
		and exists (select 1 from chamadas c where c.agendamento_id = a.id)
		order by a.ausente_em desc
	`, gradeIDs)
	if err != nil {
		return nil, err
	}
	return escanearAgendamentos(rows)
}

func (r *Repo) BuscarAgendamentoPorID(ctx context.Context, id string) (*domain.Agendamento, error) {
	row := r.Pool.QueryRow(ctx, `select `+colunasAgendamento+` from agendamentos where id = $1`, id)
	return escanearAgendamento(row)
}

// ConfirmarChegada grava chegada_em e calcula a prioridade UMA VEZ, comparando contra o
// SLA da unidade NO MOMENTO da confirmação — não recalculado depois se o SLA mudar.
func (r *Repo) ConfirmarChegada(ctx context.Context, agendamentoID string, agora time.Time, prioridade domain.Prioridade) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `
		update agendamentos
		set chegada_em = $2, prioridade = $3, status = 'sala_espera'
		where id = $1 and status = 'aguardando_chegada'
	`, agendamentoID, agora, prioridade)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// DesfazerChegada reverte uma "confirmar chegada" feita por engano — só permitido enquanto
// o agendamento ainda está em sala_espera (ninguém chamou ainda). Limpa chegada_em/
// prioridade, como se a chegada nunca tivesse sido confirmada (decisão 21/09, pedido do
// dono do produto: recepção precisa poder "puxar de volta" alguém da sala de espera).
func (r *Repo) DesfazerChegada(ctx context.Context, agendamentoID string) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `
		update agendamentos
		set chegada_em = null, prioridade = null, status = 'aguardando_chegada'
		where id = $1 and status = 'sala_espera'
	`, agendamentoID)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ChamarProximo usa FOR UPDATE SKIP LOCKED pra garantir que, sob chamadas simultâneas de
// atendentes diferentes, cada agendamento seja entregue a exatamente um atendente.
//
// Duas travas checadas ANTES de puxar alguém novo da sala de espera (decisão 20/09, ajustada
// 23/09 — ver skill regras-negocio-fila e o comentário de TemChamadaPendenteNaoRechamada):
// (1) o próprio usuário não pode ter uma chamada pendente cujo cooldown da última chamada
// ainda não passou; (2) se existir QUALQUER chamado órfão na unidade (alguém que o dono já
// rechamou e seguiu em frente), ninguém pode chamar gente nova da sala de espera até assumir
// os órfãos primeiro. Checagem feita fora da transação principal — uma pequena janela de
// corrida é aceitável aqui (mesma simplificação já usada na checagem de protocolo duplicado
// da importação), o pior caso é vez ou outra deixar passar uma chamada que devia ter sido
// bloqueada, não perder dado.
func (r *Repo) ChamarProximo(ctx context.Context, gradeID, guicheID, usuarioID string, cooldownMinutos int) (*domain.Agendamento, *domain.Chamada, error) {
	pendente, err := r.TemChamadaPendenteNaoRechamada(ctx, gradeID, usuarioID, cooldownMinutos)
	if err != nil {
		return nil, nil, err
	}
	if pendente {
		return nil, nil, ErrChamadaPendente
	}
	orfaos, err := r.ExistemChamadosOrfaos(ctx, gradeID)
	if err != nil {
		return nil, nil, err
	}
	if orfaos {
		return nil, nil, ErrExistemOrfaos
	}

	var resultado *domain.Agendamento
	var chamada *domain.Chamada

	err = pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		row := tx.QueryRow(ctx, `
			select id from agendamentos
			where grade_id = $1 and status = 'sala_espera'
			order by
				case when prioridade = 'no_horario' then 0 else 1 end,
				case when prioridade = 'no_horario' then horario_previsto else chegada_em end asc
			limit 1
			for update skip locked
		`, gradeID)

		var proximoID string
		if err := row.Scan(&proximoID); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrNaoEncontrado
			}
			return err
		}

		aRow := tx.QueryRow(ctx, `
			update agendamentos set status = 'chamado', guiche_id = $2
			where id = $1
			returning `+colunasAgendamento, proximoID, guicheID)
		a, err := escanearAgendamento(aRow)
		if err != nil {
			return err
		}

		cRow := tx.QueryRow(ctx, `
			insert into chamadas (agendamento_id, guiche_id, usuario_id)
			values ($1, $2, $3)
			returning id, agendamento_id, guiche_id, usuario_id, chamado_em
		`, a.ID, guicheID, usuarioID)
		var c domain.Chamada
		if err := cRow.Scan(&c.ID, &c.AgendamentoID, &c.GuicheID, &c.UsuarioID, &c.ChamadoEm); err != nil {
			return err
		}

		resultado = a
		chamada = &c
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return resultado, chamada, nil
}

// Rechamar cria uma nova linha em chamadas (mesmo guichê da última chamada), sem mudar
// o status — o painel passa a exibir essa como a mais recente.
func (r *Repo) Rechamar(ctx context.Context, agendamentoID, usuarioID string) (*domain.Chamada, error) {
	var guicheID string
	err := r.Pool.QueryRow(ctx, `
		select guiche_id from chamadas where agendamento_id = $1 order by chamado_em desc limit 1
	`, agendamentoID).Scan(&guicheID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNaoEncontrado
	}
	if err != nil {
		return nil, err
	}

	row := r.Pool.QueryRow(ctx, `
		insert into chamadas (agendamento_id, guiche_id, usuario_id)
		values ($1, $2, $3)
		returning id, agendamento_id, guiche_id, usuario_id, chamado_em
	`, agendamentoID, guicheID, usuarioID)
	var c domain.Chamada
	if err := row.Scan(&c.ID, &c.AgendamentoID, &c.GuicheID, &c.UsuarioID, &c.ChamadoEm); err != nil {
		return nil, err
	}
	return &c, nil
}

// AssumirChamada transfere um chamado órfão pra outro atendente/guichê — valida de novo no
// banco que é realmente órfão (não confia só na tela) antes de gravar. Atualiza o
// guiche_id do agendamento pro novo guichê (a pessoa deve ir pro balcão de quem assumiu) e
// registra uma nova linha em `chamadas` com o novo dono, do mesmo jeito que uma chamada
// normal — o painel de TV também anuncia essa transferência.
func (r *Repo) AssumirChamada(ctx context.Context, gradeID, agendamentoID, guicheID, usuarioID string) (*domain.Agendamento, *domain.Chamada, error) {
	var resultado *domain.Agendamento
	var chamada *domain.Chamada

	err := pgx.BeginFunc(ctx, r.Pool, func(tx pgx.Tx) error {
		var ehOrfao bool
		err := tx.QueryRow(ctx, `
			select exists(
				select 1 from agendamentos a
				`+juncoesUltimaChamada+`
				where a.id = $1 and a.grade_id = $2 and a.status = 'chamado' and (`+condicaoOrfao+`)
			)
		`, agendamentoID, gradeID).Scan(&ehOrfao)
		if err != nil {
			return err
		}
		if !ehOrfao {
			return ErrNaoEhOrfao
		}

		aRow := tx.QueryRow(ctx, `
			update agendamentos set guiche_id = $2
			where id = $1
			returning `+colunasAgendamento, agendamentoID, guicheID)
		a, err := escanearAgendamento(aRow)
		if err != nil {
			return err
		}

		cRow := tx.QueryRow(ctx, `
			insert into chamadas (agendamento_id, guiche_id, usuario_id)
			values ($1, $2, $3)
			returning id, agendamento_id, guiche_id, usuario_id, chamado_em
		`, a.ID, guicheID, usuarioID)
		var c domain.Chamada
		if err := cRow.Scan(&c.ID, &c.AgendamentoID, &c.GuicheID, &c.UsuarioID, &c.ChamadoEm); err != nil {
			return err
		}

		resultado = a
		chamada = &c
		return nil
	})
	if err != nil {
		return nil, nil, err
	}
	return resultado, chamada, nil
}

func (r *Repo) MarcarAtendido(ctx context.Context, agendamentoID string, agora time.Time) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `
		update agendamentos set status = 'atendido', atendido_em = $2
		where id = $1 and status = 'chamado'
	`, agendamentoID, agora)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// MarcarAusenteManual (23/09, pedido do dono do produto): botão de ausência manual pro
// atendente, pro caso de alguém avisar que o cidadão já foi embora (ou o próprio cidadão
// avisar) — sem precisar esperar o sweeper automático (SweepAusenciaAutomatica) rodar até o
// fim do SLA de atendimento. Mesma transição de estado que a ausência automática (status +
// ausente_em), só que disparada manualmente por quem é dono da chamada (checado no handler,
// mesmo padrão de PostAtendido/PostRechamar) em vez de por tempo.
func (r *Repo) MarcarAusenteManual(ctx context.Context, agendamentoID string, agora time.Time) (int64, error) {
	tag, err := r.Pool.Exec(ctx, `
		update agendamentos set status = 'ausente', ausente_em = $2
		where id = $1 and status = 'chamado'
	`, agendamentoID, agora)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// SweepAusenciaAutomatica varre as duas janelas de timeout (confirmado 19/09: os dois
// estágios viram ausente automático, limite sempre exclusivo — < estrito) e retorna
// quantos agendamentos mudaram em cada estágio. Ver skill regras-negocio-fila.
//
// Ajuste 20/09: o prazo de atendimento conta a partir da PRIMEIRA chamada (min), não mais
// da última (max) — antes, cada rechamada "resetava" o relógio pra mais 20min inteiros;
// agora os 20min são um orçamento fixo desde a primeira chamada (os 3min de cooldown de
// rechamada são só uma fatia inicial desse mesmo orçamento, não algo à parte). Rechamar ou
// um outro atendente assumir não dão mais tempo extra, só usam o tempo que já tinha.
func (r *Repo) SweepAusenciaAutomatica(ctx context.Context, agora time.Time) (chegada int64, atendimento int64, err error) {
	tagChegada, err := r.Pool.Exec(ctx, `
		update agendamentos a
		set status = 'ausente', ausente_em = $1
		from grades g
		where a.grade_id = g.id
		and a.status = 'aguardando_chegada'
		and a.tipo = 'agendado'
		and a.horario_previsto + (g.sla_chegada_minutos || ' minutes')::interval < $1
	`, agora)
	if err != nil {
		return 0, 0, err
	}

	tagAtendimento, err := r.Pool.Exec(ctx, `
		update agendamentos a
		set status = 'ausente', ausente_em = $1
		from grades g
		where a.grade_id = g.id
		and a.status = 'chamado'
		and (
			select min(c.chamado_em) from chamadas c where c.agendamento_id = a.id
		) + (g.sla_atendimento_minutos || ' minutes')::interval < $1
	`, agora)
	if err != nil {
		return 0, 0, err
	}

	return tagChegada.RowsAffected(), tagAtendimento.RowsAffected(), nil
}
