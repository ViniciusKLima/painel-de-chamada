package views

import "painel-chamada-backend/internal/domain"

// GradeParaJSON expõe a configuração completa de uma grade — usado nas telas de gestão
// (Grade de horário, configurações da grade). `excluida` (23/09) distingue soft-delete
// (`excluido_em`) do toggle reversível `ativo`, ver skill modelo-dados.
func GradeParaJSON(g *domain.Grade) map[string]any {
	return map[string]any{
		"id":                               g.ID,
		"unidadeId":                        g.UnidadeID,
		"servico":                          g.Servico,
		"nome":                             g.Nome,
		"slaChegadaMinutos":                g.SLAChegadaMinutos,
		"slaAtendimentoMinutos":            g.SLAAtendimentoMinutos,
		"duracaoChamadaPainelSegundos":     g.DuracaoChamadaPainelSegundos,
		"duracaoChamadaPainelFilaSegundos": g.DuracaoChamadaPainelFilaSegundos,
		"repeticoesChamada":                g.RepeticoesChamada,
		"intervaloRepeticaoSegundos":       g.IntervaloRepeticaoSegundos,
		"cooldownRechamadaMinutos":         g.CooldownRechamadaMinutos,
		"permiteEncaixeRecepcao":           g.PermiteEncaixeRecepcao,
		"ativo":                            g.Ativo,
		"excluida":                         g.Excluida,
	}
}

func GradesParaJSON(lista []domain.Grade) []map[string]any {
	out := make([]map[string]any, len(lista))
	for i, g := range lista {
		gg := g
		out[i] = GradeParaJSON(&gg)
	}
	return out
}

// GuicheParaJSON expõe um guichê/sala, incluindo ocupação em tempo real (já resolvida pelo
// repository com a janela de heartbeat) e `excluida` (23/09, mesma distinção de GradeParaJSON).
func GuicheParaJSON(g domain.Guiche) map[string]any {
	return map[string]any{
		"id":                  g.ID,
		"gradeId":             g.GradeID,
		"nome":                g.Nome,
		"ativo":               g.Ativo,
		"tipo":                g.Tipo,
		"andar":               g.Andar,
		"capacidade":          g.Capacidade,
		"ocupadoPorUsuarioId": g.OcupadoPorUsuarioID,
		"ocupadoPorNome":      g.OcupadoPorNome,
		"excluida":            g.Excluida,
	}
}
