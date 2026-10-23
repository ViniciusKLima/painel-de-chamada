package views

import (
	"time"

	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/painel"
	"painel-chamada-backend/internal/repository"
)

// NomeExibicaoPainel decide o título mostrado no painel de TV — pedido do dono do produto
// (23/09): "tire o nome do serviço do título do painel, deixe somente a grade de horário".
// A unidade só é usada como fallback pra grade "Geral" (auto-criada pra encaixe sem serviço,
// sem nome próprio de verdade).
func NomeExibicaoPainel(nomeUnidade, nomeGrade string) string {
	if nomeGrade == "" || nomeGrade == "Geral" {
		return nomeUnidade
	}
	return nomeGrade
}

// PainelResponse monta a resposta pública de `GET /painel/{gradeId}` a partir de dado JÁ
// RESOLVIDO pelo controller (grade, unidade, chamada atual da fila de exibição em memória,
// últimas chamadas, cores da prefeitura) — pura decisão de formato, sem nenhuma consulta
// própria. Ver skill painel-tv pro desenho completo (modo ativo/descanso, barra de contagem,
// voz).
//
// Removido o modo "aguardando" (23/09, revertido 20/10): mantinha quem já tinha sido chamado
// visível em destaque indefinidamente, até ser marcado ou outro atendente chamar alguém —
// passava a impressão de tela travada quando os 30s normais da exibição já tinham passado há
// muito. Agora a tabela de últimas chamadas simplesmente mostra todo mundo, sem exclusão —
// quem ainda está "chamado" (não órfão) continua distinguível ali pelo ícone cinza
// (`ehOrfao=false`, ver skill painel-tv), só não ocupa mais o destaque pra sempre.
func PainelResponse(
	grade *domain.Grade,
	nomeUnidade string,
	atual *painel.ItemExibicao,
	expiraEm time.Time,
	duracaoTotal time.Duration,
	ultimas []repository.ChamadaComDetalhe,
	corDestaque, corClara any,
) map[string]any {
	var atualJSON map[string]any
	if atual != nil {
		atualJSON = map[string]any{
			"protocolo": atual.Protocolo,
			"nome":      atual.Nome,
			"guiche":    atual.GuicheNome,
			// expiraEm/duracaoTotalSegundos — só pra desenhar a barra de contagem regressiva no
			// painel; quem decide de verdade quando trocar de exibição continua sendo só o
			// backend, isso aqui é decorativo.
			"expiraEm":             expiraEm,
			"duracaoTotalSegundos": int(duracaoTotal.Seconds()),
		}
	}

	listaJSON := []map[string]any{}
	for _, c := range ultimas {
		listaJSON = append(listaJSON, map[string]any{
			"id":        c.ChamadaID,
			"protocolo": c.Protocolo,
			"nome":      c.NomeCidadao,
			"guiche":    c.GuicheNome,
			"status":    c.Status, // atendido | chamado (esperando/órfão) | ausente — ver skill painel-tv
			"ehOrfao":   c.EhOrfao,
			"chamadoEm": c.ChamadoEm,
		})
	}

	return map[string]any{
		"unidade": map[string]any{
			"id":                         grade.ID,
			"nome":                       NomeExibicaoPainel(nomeUnidade, grade.Nome),
			"repeticoesChamada":          grade.RepeticoesChamada,
			"intervaloRepeticaoSegundos": grade.IntervaloRepeticaoSegundos,
		},
		"corDestaque":     corDestaque,
		"corClara":        corClara,
		"chamadaAtual":    atualJSON,
		"ultimasChamadas": listaJSON,
	}
}

// PaineisPublicoDaUnidadeResponse monta a resposta do kiosk público por unidade (23/09) — só
// id/nome de cada grade ATIVA (uma grade inativa não tem fila rodando).
func PaineisPublicoDaUnidadeResponse(nomeUnidade string, grades []domain.Grade) map[string]any {
	ativas := []map[string]any{}
	for _, g := range grades {
		if g.Ativo {
			ativas = append(ativas, map[string]any{"id": g.ID, "nome": g.Nome})
		}
	}
	return map[string]any{"unidadeNome": nomeUnidade, "grades": ativas}
}
