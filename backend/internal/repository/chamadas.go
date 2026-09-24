package repository

import (
	"context"
	"time"
)

type ChamadaComDetalhe struct {
	ChamadaID     string
	AgendamentoID string
	Protocolo     string
	NomeCidadao   string
	GuicheNome    string
	Status        string
	ChamadoEm     time.Time
	EhOrfao       bool
}

// UltimasChamadas alimenta a tabela "últimas chamadas" do painel de TV — escopo por grade
// (22/09), já que cada grade é uma fila/painel independente.
//
// Uma linha POR AGENDAMENTO, não por chamada (23/09, corrigindo bug real relatado: "se eu
// chamei alguém e ele entrou na tabela, se eu chamar de novo, ele não sobe pra tabela, ele
// cria uma nova linha, ficando a mesma pessoa duas vezes"). `chamadas` é um log
// intencionalmente append-only (uma linha por chamar/rechamar, ver skill modelo-dados) — o
// bug era mostrar esse log cru na tabela em vez de deduplicar por cidadão. Reaproveita
// juncoesUltimaChamada/condicaoOrfao (agendamentos.go) pra pegar só a chamada mais recente de
// cada agendamento e marcar ehOrfao com a MESMA lógica usada no resto do sistema (bloco
// "Chamando agora", chamar-próximo) — sem isso, o ícone cinza novo (23/09, ver painel.go)
// ficaria inconsistente com o que o atendente vê na própria tela.
func (r *Repo) UltimasChamadas(ctx context.Context, gradeID string, limite int) ([]ChamadaComDetalhe, error) {
	rows, err := r.Pool.Query(ctx, `
		select ultima.chamada_id, a.id, a.protocolo, a.nome_cidadao, g.nome, a.status, ultima.chamado_em,
		       (`+condicaoOrfao+`) as eh_orfao
		from agendamentos a
		`+juncoesUltimaChamada+`
		join guiches g on g.id = ultima.guiche_id
		where a.grade_id = $1
		order by ultima.chamado_em desc
		limit $2
	`, gradeID, limite)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ChamadaComDetalhe
	for rows.Next() {
		var c ChamadaComDetalhe
		if err := rows.Scan(&c.ChamadaID, &c.AgendamentoID, &c.Protocolo, &c.NomeCidadao, &c.GuicheNome, &c.Status, &c.ChamadoEm, &c.EhOrfao); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}

// HistoricoDoAgendamento é a "janela do cidadão": todas as chamadas/rechamadas de um
// agendamento específico, mais antiga primeiro.
func (r *Repo) HistoricoDoAgendamento(ctx context.Context, agendamentoID string) ([]ChamadaComDetalhe, error) {
	rows, err := r.Pool.Query(ctx, `
		select c.id, c.agendamento_id, a.protocolo, a.nome_cidadao, g.nome, a.status, c.chamado_em
		from chamadas c
		join agendamentos a on a.id = c.agendamento_id
		join guiches g on g.id = c.guiche_id
		where c.agendamento_id = $1
		order by c.chamado_em asc
	`, agendamentoID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var lista []ChamadaComDetalhe
	for rows.Next() {
		var c ChamadaComDetalhe
		if err := rows.Scan(&c.ChamadaID, &c.AgendamentoID, &c.Protocolo, &c.NomeCidadao, &c.GuicheNome, &c.Status, &c.ChamadoEm); err != nil {
			return nil, err
		}
		lista = append(lista, c)
	}
	return lista, rows.Err()
}
