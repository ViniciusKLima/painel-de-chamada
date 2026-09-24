package views

import (
	"time"

	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
)

func AgendamentoParaJSON(a domain.Agendamento) map[string]any {
	return map[string]any{
		"id":                 a.ID,
		"unidadeId":          a.UnidadeID,
		"gradeId":            a.GradeID,
		"protocolo":          a.Protocolo,
		"nomeCidadao":        a.NomeCidadao,
		"cpf":                a.CPF,
		"telefone":           a.Telefone,
		"servico":            a.Servico,
		"gradeHorario":       a.GradeHorario,
		"tipo":               a.Tipo,
		"horarioPrevisto":    a.HorarioPrevisto,
		"chegadaEm":          a.ChegadaEm,
		"prioridade":         a.Prioridade,
		"guicheId":           a.GuicheID,
		"status":             a.Status,
		"atendidoEm":         a.AtendidoEm,
		"ausenteEm":          a.AusenteEm,
		"canceladoEm":        a.CanceladoEm,
		"motivoCancelamento": a.MotivoCancelamento,
	}
}

func AgendamentosParaJSON(lista []domain.Agendamento) []map[string]any {
	out := make([]map[string]any, len(lista))
	for i, a := range lista {
		out[i] = AgendamentoParaJSON(a)
	}
	return out
}

// ChamadoCompartilhadoParaJSON monta a linha do bloco "Chamando agora" — visível a todos os
// atendentes da grade, com "souDono" indicando se É este usuário quem pode agir nela (marcar
// presente / rechamar) e "ehOrfao" indicando se qualquer atendente ocioso pode assumir. Ver
// skill regras-negocio-fila.
func ChamadoCompartilhadoParaJSON(item repository.ChamadoCompartilhado, grade *domain.Grade, usuarioID string) map[string]any {
	j := AgendamentoParaJSON(item.Agendamento)
	j["guicheNome"] = item.GuicheNome
	j["chamadoPorUsuarioId"] = item.ChamadoPorUsuarioID
	j["chamadoPorNome"] = item.ChamadoPorNome
	j["souDono"] = item.ChamadoPorUsuarioID == usuarioID
	j["ehOrfao"] = item.EhOrfao
	j["totalChamadasDoDono"] = item.TotalChamadasDoDono
	j["podeRechamarEm"] = item.UltimaChamadaEm.Add(time.Duration(grade.CooldownRechamadaMinutos) * time.Minute)
	j["prazoAusenciaEm"] = item.PrimeiraChamadaEm.Add(time.Duration(grade.SLAAtendimentoMinutos) * time.Minute)
	return j
}
