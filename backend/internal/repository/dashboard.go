package repository

import (
	"context"
	"time"
)

type ResumoDashboard struct {
	TotalDia     int                     `json:"totalDia"`
	Atendidos    int                     `json:"atendidos"`
	Ausentes     int                     `json:"ausentes"`
	PorAtendente []AtendidosPorAtendente `json:"porAtendente"`
	SerieDiaria  []PontoSerieDiaria      `json:"serieDiaria"`
	PorGrade     []AtendimentosPorGrade  `json:"porGrade"`
}

type AtendidosPorAtendente struct {
	UsuarioID string `json:"usuarioId"`
	Nome      string `json:"nome"`
	Total     int    `json:"total"`
}

// PontoSerieDiaria alimenta o gráfico de linha "atendidos vs. ausentes" do dashboard do
// gestor — um ponto por dia dos últimos 7, mesmo os dias sem nenhum movimento (contagem
// zero), pra não quebrar a continuidade do eixo X.
type PontoSerieDiaria struct {
	Data      string `json:"data"`
	Atendidos int    `json:"atendidos"`
	Ausentes  int    `json:"ausentes"`
}

// AtendimentosPorGrade — gráfico extra (22/09, "se tiver potencial pra mais gráficos,
// coloque"): dá pra ver de cara qual serviço/grade concentra mais atendimento hoje, algo que
// só passou a fazer sentido depois que uma unidade ganhou a possibilidade de ter várias
// grades. Ajustado no mesmo dia (segunda rodada) pra trazer também os ausentes — a
// proporção atendido-vs-ausente de cada grade é mais informativa que só o volume absoluto.
type AtendimentosPorGrade struct {
	GradeID   string `json:"gradeId"`
	Nome      string `json:"nome"`
	Atendidos int    `json:"atendidos"`
	Ausentes  int    `json:"ausentes"`
}

// ResumoDoDia: métricas pro dashboard do gestor — "hoje" é a data do horario_previsto (ou
// de criado_em, pra encaixe, que não tem horario_previsto). Escopado por SECRETARIA (22/09,
// gestor deixou de ser vinculado a uma unidade só — ver skill modelo-dados), agregando
// todas as unidades daquela secretaria via subquery; no piloto (uma unidade por secretaria)
// o resultado é idêntico ao que era antes por unidade.
func (r *Repo) ResumoDoDia(ctx context.Context, secretariaID string) (*ResumoDashboard, error) {
	// PorAtendente/PorGrade inicializados vazios, não nil — um slice nil vira `null` no
	// JSON (achado real ao testar a tela de importação, mesma classe de bug, ver
	// importacao_preview.go), e "ninguém atendeu ainda hoje" é o caso mais comum de todos.
	resumo := ResumoDashboard{PorAtendente: []AtendidosPorAtendente{}, PorGrade: []AtendimentosPorGrade{}}
	err := r.Pool.QueryRow(ctx, `
		select
			count(*) filter (where coalesce(horario_previsto::date, criado_em::date) = current_date),
			count(*) filter (where status = 'atendido' and coalesce(horario_previsto::date, criado_em::date) = current_date),
			count(*) filter (where status = 'ausente' and coalesce(horario_previsto::date, criado_em::date) = current_date)
		from agendamentos where unidade_id in (select id from unidades where secretaria_id = $1)
	`, secretariaID).Scan(&resumo.TotalDia, &resumo.Atendidos, &resumo.Ausentes)
	if err != nil {
		return nil, err
	}

	rows, err := r.Pool.Query(ctx, `
		select u.id, u.nome, count(*)
		from agendamentos a
		join usuarios u on u.id = (
			select c.usuario_id from chamadas c where c.agendamento_id = a.id order by c.chamado_em desc limit 1
		)
		where a.unidade_id in (select id from unidades where secretaria_id = $1) and a.status = 'atendido'
		and coalesce(a.horario_previsto::date, a.criado_em::date) = current_date
		group by u.id, u.nome
		order by count(*) desc
	`, secretariaID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p AtendidosPorAtendente
		if err := rows.Scan(&p.UsuarioID, &p.Nome, &p.Total); err != nil {
			rows.Close()
			return nil, err
		}
		resumo.PorAtendente = append(resumo.PorAtendente, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	serie, err := r.serieDiaria(ctx, secretariaID)
	if err != nil {
		return nil, err
	}
	resumo.SerieDiaria = serie

	rowsGrade, err := r.Pool.Query(ctx, `
		select g.id, g.nome,
		       count(*) filter (where a.status = 'atendido'),
		       count(*) filter (where a.status = 'ausente')
		from agendamentos a
		join grades g on g.id = a.grade_id
		where a.unidade_id in (select id from unidades where secretaria_id = $1) and a.status in ('atendido', 'ausente')
		and coalesce(a.horario_previsto::date, a.criado_em::date) = current_date
		group by g.id, g.nome
		order by count(*) desc
	`, secretariaID)
	if err != nil {
		return nil, err
	}
	for rowsGrade.Next() {
		var p AtendimentosPorGrade
		if err := rowsGrade.Scan(&p.GradeID, &p.Nome, &p.Atendidos, &p.Ausentes); err != nil {
			rowsGrade.Close()
			return nil, err
		}
		resumo.PorGrade = append(resumo.PorGrade, p)
	}
	rowsGrade.Close()
	if err := rowsGrade.Err(); err != nil {
		return nil, err
	}

	return &resumo, nil
}

// ResumoDoDiaPrefeitura (23/09) — mesma métrica de ResumoDoDia, agregada pra TODAS as
// secretarias de uma prefeitura de uma vez (a aba "Dashboard" da configuração da prefeitura,
// no admin — "dashboard e usuarios internos... dentro da configuração de cada prefeitura").
// Mesma estrutura de query, só o subquery de unidades muda (via secretarias->prefeitura em
// vez de secretaria direta) — duplicado em vez de generalizado de propósito, pra não criar
// uma abstração só pra economizar 3 linhas repetidas (ver constituição do projeto).
func (r *Repo) ResumoDoDiaPrefeitura(ctx context.Context, prefeituraID string) (*ResumoDashboard, error) {
	resumo := ResumoDashboard{PorAtendente: []AtendidosPorAtendente{}, PorGrade: []AtendimentosPorGrade{}}
	subUnidades := `select u.id from unidades u join secretarias s on s.id = u.secretaria_id where s.prefeitura_id = $1`

	err := r.Pool.QueryRow(ctx, `
		select
			count(*) filter (where coalesce(horario_previsto::date, criado_em::date) = current_date),
			count(*) filter (where status = 'atendido' and coalesce(horario_previsto::date, criado_em::date) = current_date),
			count(*) filter (where status = 'ausente' and coalesce(horario_previsto::date, criado_em::date) = current_date)
		from agendamentos where unidade_id in (`+subUnidades+`)
	`, prefeituraID).Scan(&resumo.TotalDia, &resumo.Atendidos, &resumo.Ausentes)
	if err != nil {
		return nil, err
	}

	rows, err := r.Pool.Query(ctx, `
		select u.id, u.nome, count(*)
		from agendamentos a
		join usuarios u on u.id = (
			select c.usuario_id from chamadas c where c.agendamento_id = a.id order by c.chamado_em desc limit 1
		)
		where a.unidade_id in (`+subUnidades+`) and a.status = 'atendido'
		and coalesce(a.horario_previsto::date, a.criado_em::date) = current_date
		group by u.id, u.nome
		order by count(*) desc
	`, prefeituraID)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var p AtendidosPorAtendente
		if err := rows.Scan(&p.UsuarioID, &p.Nome, &p.Total); err != nil {
			rows.Close()
			return nil, err
		}
		resumo.PorAtendente = append(resumo.PorAtendente, p)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}

	serieRows, err := r.Pool.Query(ctx, `
		select coalesce(horario_previsto::date, criado_em::date) as dia,
		       count(*) filter (where status = 'atendido'),
		       count(*) filter (where status = 'ausente')
		from agendamentos
		where unidade_id in (`+subUnidades+`)
		and coalesce(horario_previsto::date, criado_em::date) between current_date - interval '6 days' and current_date
		group by dia
	`, prefeituraID)
	if err != nil {
		return nil, err
	}
	type contagem struct{ atendidos, ausentes int }
	porDia := map[string]contagem{}
	for serieRows.Next() {
		var dia time.Time
		var c contagem
		if err := serieRows.Scan(&dia, &c.atendidos, &c.ausentes); err != nil {
			serieRows.Close()
			return nil, err
		}
		porDia[dia.Format("2006-01-02")] = c
	}
	serieRows.Close()
	if err := serieRows.Err(); err != nil {
		return nil, err
	}
	hoje := time.Now()
	for i := 6; i >= 0; i-- {
		chave := hoje.AddDate(0, 0, -i).Format("2006-01-02")
		c := porDia[chave]
		resumo.SerieDiaria = append(resumo.SerieDiaria, PontoSerieDiaria{Data: chave, Atendidos: c.atendidos, Ausentes: c.ausentes})
	}

	rowsGrade, err := r.Pool.Query(ctx, `
		select g.id, g.nome,
		       count(*) filter (where a.status = 'atendido'),
		       count(*) filter (where a.status = 'ausente')
		from agendamentos a
		join grades g on g.id = a.grade_id
		where a.unidade_id in (`+subUnidades+`) and a.status in ('atendido', 'ausente')
		and coalesce(a.horario_previsto::date, a.criado_em::date) = current_date
		group by g.id, g.nome
		order by count(*) desc
	`, prefeituraID)
	if err != nil {
		return nil, err
	}
	for rowsGrade.Next() {
		var p AtendimentosPorGrade
		if err := rowsGrade.Scan(&p.GradeID, &p.Nome, &p.Atendidos, &p.Ausentes); err != nil {
			rowsGrade.Close()
			return nil, err
		}
		resumo.PorGrade = append(resumo.PorGrade, p)
	}
	rowsGrade.Close()
	if err := rowsGrade.Err(); err != nil {
		return nil, err
	}

	return &resumo, nil
}

// ResumoAdmin — contadores simples pro dashboard do admin (22/09): quantas prefeituras,
// secretarias e usuários (por papel) existem na plataforma inteira. Deliberadamente sem
// gráfico/série temporal nesta rodada — a prioridade era a estrutura de acesso, não
// dashboards de negócio pro admin (ver CLAUDE.md).
type ResumoAdmin struct {
	TotalPrefeituras    int `json:"totalPrefeituras"`
	TotalSecretarias    int `json:"totalSecretarias"`
	TotalGestores       int `json:"totalGestores"`
	TotalAtendentes     int `json:"totalAtendentes"`
	TotalRecepcionistas int `json:"totalRecepcionistas"`
}

func (r *Repo) ResumoAdmin(ctx context.Context) (*ResumoAdmin, error) {
	var res ResumoAdmin
	err := r.Pool.QueryRow(ctx, `
		select
			(select count(*) from prefeituras),
			(select count(*) from secretarias),
			(select count(*) from usuarios where papel = 'gestor'),
			(select count(*) from usuarios where papel = 'atendente'),
			(select count(*) from usuarios where papel = 'recepcionista')
	`).Scan(&res.TotalPrefeituras, &res.TotalSecretarias, &res.TotalGestores, &res.TotalAtendentes, &res.TotalRecepcionistas)
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// serieDiaria monta os últimos 7 dias (hoje incluído) com contagem de atendidos/ausentes,
// preenchendo com zero os dias sem nenhum registro — o gráfico de linha precisa de um ponto
// por dia, não só dos dias com movimento.
func (r *Repo) serieDiaria(ctx context.Context, secretariaID string) ([]PontoSerieDiaria, error) {
	rows, err := r.Pool.Query(ctx, `
		select coalesce(horario_previsto::date, criado_em::date) as dia,
		       count(*) filter (where status = 'atendido'),
		       count(*) filter (where status = 'ausente')
		from agendamentos
		where unidade_id in (select id from unidades where secretaria_id = $1)
		and coalesce(horario_previsto::date, criado_em::date) between current_date - interval '6 days' and current_date
		group by dia
	`, secretariaID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type contagem struct{ atendidos, ausentes int }
	porDia := map[string]contagem{}
	for rows.Next() {
		var dia time.Time
		var c contagem
		if err := rows.Scan(&dia, &c.atendidos, &c.ausentes); err != nil {
			return nil, err
		}
		porDia[dia.Format("2006-01-02")] = c
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	hoje := time.Now()
	serie := make([]PontoSerieDiaria, 0, 7)
	for i := 6; i >= 0; i-- {
		chave := hoje.AddDate(0, 0, -i).Format("2006-01-02")
		c := porDia[chave]
		serie = append(serie, PontoSerieDiaria{Data: chave, Atendidos: c.atendidos, Ausentes: c.ausentes})
	}
	return serie, nil
}
