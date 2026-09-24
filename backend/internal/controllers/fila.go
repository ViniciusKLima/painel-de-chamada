package controllers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/painel"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

// usuarioPodeAcessarUnidade resolve a secretaria dona da unidade (pra gestor, cuja
// autoridade é sobre a secretaria inteira) e delega a decisão pra auth.PodeAcessarUnidade —
// central de permissões, ver internal/auth/permissoes.go. Usado tanto por exigirUnidade
// quanto por qualquer handler que já tenha um agendamento/recurso carregado e precise
// checar se ele pertence a uma unidade acessível pelo usuário logado.
func (c *Controllers) usuarioPodeAcessarUnidade(ctx context.Context, usuario *domain.Usuario, unidadeID string) bool {
	if auth.EhAdmin(usuario) {
		return true
	}
	unidade, err := c.Repo.BuscarUnidadePorID(ctx, unidadeID)
	if err != nil {
		return false
	}
	return auth.PodeAcessarUnidade(usuario, unidade.SecretariaID, unidade.ID)
}

// exigirSecretaria garante que o usuário autenticado pode acessar a secretaria do path
// (ver auth.PodeAcessarSecretaria): admin sempre pode, gestor só a própria secretaria.
// Usada pelas telas de gestor que passaram a ser escopadas pela secretaria inteira em vez
// de uma unidade só (dashboard, grade de horário, usuários internos — 22/09/admin).
func (c *Controllers) exigirSecretaria(w http.ResponseWriter, r *http.Request) (usuario *domain.Usuario, secretariaID string, ok bool) {
	usuario = auth.UsuarioDoContexto(r.Context())
	secretariaID = chi.URLParam(r, "secretariaId")
	if !auth.PodeAcessarSecretaria(usuario, secretariaID) {
		views.ResponderErro(w, http.StatusForbidden, "secretaria_incorreta", "usuario nao pertence a esta secretaria")
		return nil, "", false
	}
	return usuario, secretariaID, true
}

// exigirUnidade garante que o usuário autenticado pode acessar a unidade do path —
// isolamento de tenant central (ver auth.PodeAcessarUnidade): admin sempre pode, gestor
// pode qualquer unidade da própria secretaria, recepcionista só a própria unidade.
func (c *Controllers) exigirUnidade(w http.ResponseWriter, r *http.Request) (usuario *domain.Usuario, unidadeID string, ok bool) {
	usuario = auth.UsuarioDoContexto(r.Context())
	unidadeID = chi.URLParam(r, "unidadeId")
	if !c.usuarioPodeAcessarUnidade(r.Context(), usuario, unidadeID) {
		views.ResponderErro(w, http.StatusForbidden, "unidade_incorreta", "usuario nao pertence a esta unidade")
		return nil, "", false
	}
	return usuario, unidadeID, true
}

// exigirGrade garante que o usuário autenticado pode acessar a grade do path (ver
// auth.PodeAcessarGrade): admin sempre pode, gestor qualquer grade de qualquer unidade da
// própria secretaria, atendente só a grade em que está alocado.
func (c *Controllers) exigirGrade(w http.ResponseWriter, r *http.Request) (usuario *domain.Usuario, grade *domain.Grade, ok bool) {
	usuario = auth.UsuarioDoContexto(r.Context())
	gradeID := chi.URLParam(r, "gradeId")
	g, err := c.Repo.BuscarGradePorID(r.Context(), gradeID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "grade nao encontrada")
		return nil, nil, false
	}
	if !auth.EhAdmin(usuario) {
		unidade, err := c.Repo.BuscarUnidadePorID(r.Context(), g.UnidadeID)
		if err != nil {
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
			return nil, nil, false
		}
		var gradesDoUsuario []string
		if usuario.Papel == domain.PapelAtendente {
			grades, err := c.Repo.ListarGradesDoUsuario(r.Context(), usuario.ID)
			if err != nil {
				views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
				return nil, nil, false
			}
			for _, gr := range grades {
				gradesDoUsuario = append(gradesDoUsuario, gr.ID)
			}
		}
		if !auth.PodeAcessarGrade(usuario, unidade.SecretariaID, g.UnidadeID, g.ID, gradesDoUsuario) {
			views.ResponderErro(w, http.StatusForbidden, "unidade_incorreta", "grade nao pertence a esta unidade")
			return nil, nil, false
		}
	}
	return usuario, g, true
}

// GetRecepcao: GET /api/v1/unidades/{unidadeId}/recepcao — lista do dia (aguardando
// chegada + sala de espera), papel recepcionista/gestor.
func (c *Controllers) GetRecepcao(w http.ResponseWriter, r *http.Request) {
	usuario, unidadeID, ok := c.exigirUnidade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelRecepcionista && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas recepcao, gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarParaRecepcao(r.Context(), unidadeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// PostConfirmarChegada: POST /api/v1/agendamentos/{id}/confirmar-chegada
func (c *Controllers) PostConfirmarChegada(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelRecepcionista && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas recepcao, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")

	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado")
		return
	}
	if !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusForbidden, "unidade_incorreta", "agendamento de outra unidade")
		return
	}
	if a.HorarioPrevisto == nil {
		views.ResponderErro(w, http.StatusConflict, "sem_horario", "agendamento sem horario previsto")
		return
	}

	grade, err := c.Repo.BuscarGradePorID(r.Context(), a.GradeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}

	agora := time.Now()
	// Limite sempre EXCLUSIVO (confirmado 19/09): < estrito, não <=.
	limiteNoHorario := a.HorarioPrevisto.Add(time.Duration(grade.SLAAtendimentoMinutos) * time.Minute)
	prioridade := domain.PrioridadeNoHorario
	if !agora.Before(limiteNoHorario) {
		prioridade = domain.PrioridadeAtrasado
	}

	linhas, err := c.Repo.ConfirmarChegada(r.Context(), agendamentoID, agora, prioridade)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if linhas == 0 {
		views.ResponderErro(w, http.StatusConflict, "estado_invalido", "agendamento nao esta aguardando chegada")
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true, "prioridade": prioridade})
}

// PostDesfazerChegada: POST /api/v1/agendamentos/{id}/desfazer-chegada — recepção "puxa de
// volta" alguém que confirmou chegada por engano, enquanto ainda está em sala_espera
// (decisão 21/09). Não se aplica a encaixe, que nunca teve etapa de aguardando_chegada.
func (c *Controllers) PostDesfazerChegada(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelRecepcionista && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas recepcao, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")

	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado")
		return
	}
	if !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusForbidden, "unidade_incorreta", "agendamento de outra unidade")
		return
	}
	if a.Tipo != domain.TipoAgendado {
		views.ResponderErro(w, http.StatusConflict, "tipo_invalido", "encaixe nao tem etapa de aguardando chegada")
		return
	}

	linhas, err := c.Repo.DesfazerChegada(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if linhas == 0 {
		views.ResponderErro(w, http.StatusConflict, "estado_invalido", "agendamento nao esta na sala de espera")
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type corpoEncaixe struct {
	NomeCidadao string `json:"nomeCidadao"`
	Telefone    string `json:"telefone"`
	GradeID     string `json:"gradeId"`
}

// PostEncaixe: POST /api/v1/unidades/{unidadeId}/encaixe — walk-in sem agendamento
// prévio, já entra direto em sala_espera. Precisa saber pra qual grade (serviço) o walk-in
// é — decisão 22/09: com mais de um serviço na mesma unidade, a recepção escolhe.
func (c *Controllers) PostEncaixe(w http.ResponseWriter, r *http.Request) {
	usuario, unidadeID, ok := c.exigirUnidade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelRecepcionista && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas recepcao, gestor ou admin")
		return
	}
	var corpo corpoEncaixe
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.NomeCidadao == "" || corpo.GradeID == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nomeCidadao e gradeId sao obrigatorios")
		return
	}
	grade, err := c.Repo.BuscarGradePorID(r.Context(), corpo.GradeID)
	if err != nil || grade.UnidadeID != unidadeID {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "grade nao encontrada nesta unidade")
		return
	}
	// Gestor/admin sempre podem cadastrar encaixe, não importa a configuração da grade
	// (23/09, pedido explícito do dono do produto) — a restrição vale só pra recepcionista.
	if usuario.Papel == domain.PapelRecepcionista && !grade.PermiteEncaixeRecepcao {
		views.ResponderErro(w, http.StatusForbidden, "encaixe_bloqueado", "esta grade nao permite encaixe pela recepcao — fale com o gestor")
		return
	}

	protocolo := "ENC-" + time.Now().Format("20060102-150405.000")
	n := repository.NovoAgendamento{
		UnidadeID:   unidadeID,
		GradeID:     grade.ID,
		Protocolo:   protocolo,
		NomeCidadao: corpo.NomeCidadao,
		Tipo:        domain.TipoEncaixe,
		Status:      domain.StatusSalaEspera,
	}
	if corpo.Telefone != "" {
		n.Telefone = &corpo.Telefone
	}

	err = c.Repo.Pool.QueryRow(r.Context(), `
		insert into agendamentos (unidade_id, grade_id, protocolo, nome_cidadao, telefone, tipo, status, chegada_em)
		values ($1,$2,$3,$4,$5,$6,$7, now())
		returning id
	`, n.UnidadeID, n.GradeID, n.Protocolo, n.NomeCidadao, n.Telefone, n.Tipo, n.Status).Scan(new(string))
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, map[string]any{"ok": true, "protocolo": protocolo})
}

// GetGuiches: GET /api/v1/grades/{gradeId}/guiches — lista pro atendente escolher em qual
// guichê está chamando (evita colar UUID à mão).
func (c *Controllers) GetGuiches(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	guiches, err := c.Repo.ListarGuichesAtivos(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(guiches))
	for i, g := range guiches {
		out[i] = views.GuicheParaJSON(g)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// PostOcuparGuiche: POST /grades/{gradeId}/guiches/{guicheId}/ocupar — atendente/gestor.
// Reivindica o guichê pro usuário logado (escolha no modal, ou heartbeat periódico
// reafirmando enquanto continua nele) — ver skill regras-negocio-fila. 409 se outro
// atendente já está lá de verdade (heartbeat recente).
func (c *Controllers) PostOcuparGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	guiche, err := c.Repo.OcuparGuiche(r.Context(), grade.ID, guicheID, usuario.ID)
	if err != nil {
		switch err {
		case repository.ErrNaoEncontrado:
			views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
		case repository.ErrGuicheOcupado:
			views.ResponderErro(w, http.StatusConflict, "guiche_ocupado", "outro atendente ja esta usando este guiche")
		default:
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		}
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GuicheParaJSON(*guiche))
}

// PostLiberarGuiche: POST /grades/{gradeId}/guiches/{guicheId}/liberar — atendente/gestor,
// chamado ao sair ou trocar de guichê, pra outros atendentes verem ele livre na hora, sem
// esperar o heartbeat expirar.
func (c *Controllers) PostLiberarGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	if err := c.Repo.LiberarGuiche(r.Context(), grade.ID, guicheID, usuario.ID); err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// minhasGrades resolve as grades do ATENDENTE logado (só esse papel — recepcionista não usa
// os endpoints "minhas grades", continua escopada por unidade) — base de todos os endpoints
// consolidados abaixo. Auditoria 23/09: "não quero uma interface em que o atendente precise
// selecionar uma grade pra conseguir visualizar ou trabalhar nela" — o gate de seleção foi
// removido do frontend, e esses endpoints são o que alimenta a visão consolidada de TODAS as
// grades alocadas numa resposta só (evita N requisições soltas, uma por grade).
func (c *Controllers) minhasGrades(w http.ResponseWriter, r *http.Request) (usuario *domain.Usuario, grades []repository.GradeComUnidade, ok bool) {
	usuario = auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAtendente {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente")
		return nil, nil, false
	}
	grades, err := c.Repo.ListarGradesDoUsuario(r.Context(), usuario.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return nil, nil, false
	}
	return usuario, grades, true
}

func idsDeGrades(grades []repository.GradeComUnidade) []string {
	ids := make([]string, len(grades))
	for i, g := range grades {
		ids[i] = g.ID
	}
	return ids
}

// GetMinhasGrades: GET /api/v1/atendente/grades — a lista de grades do atendente logado,
// com unidadeId/unidadeNome (pro frontend agrupar por local sem outra requisição). Espelha
// exatamente o que a sessão já devolve em `grades`, mas como endpoint próprio pra poder ser
// repollado sem precisar buscar a sessão inteira de novo.
func (c *Controllers) GetMinhasGrades(w http.ResponseWriter, r *http.Request) {
	_, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GradesResumoJSON(grades))
}

// GetMinhasGradesGuiches: GET /api/v1/atendente/grades/guiches — guichês de TODAS as grades
// do atendente, consolidados (cada guichê continua pertencendo a uma grade só, ver
// Guiche.GradeId no payload).
func (c *Controllers) GetMinhasGradesGuiches(w http.ResponseWriter, r *http.Request) {
	_, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	guiches, err := c.Repo.ListarGuichesAtivosPorGrades(r.Context(), idsDeGrades(grades))
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(guiches))
	for i, g := range guiches {
		out[i] = views.GuicheParaJSON(g)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// GetMinhasGradesSalaEspera: GET /api/v1/atendente/grades/sala-espera — consolidado.
func (c *Controllers) GetMinhasGradesSalaEspera(w http.ResponseWriter, r *http.Request) {
	_, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	lista, err := c.Repo.ListarSalaEsperaPorGrades(r.Context(), idsDeGrades(grades))
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// GetMinhasGradesChamados: GET /api/v1/atendente/grades/chamados — bloco "Chamando agora"
// consolidado de todas as grades do atendente. `podeRechamarEm`/`prazoAusenciaEm` de cada
// item usam o cooldown/SLA DA GRADE DAQUELE item especificamente (grades diferentes podem
// ter configurações diferentes) — não faria sentido usar uma grade só pra todos os itens.
func (c *Controllers) GetMinhasGradesChamados(w http.ResponseWriter, r *http.Request) {
	usuario, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	mapaGrades := make(map[string]repository.GradeComUnidade, len(grades))
	for _, g := range grades {
		mapaGrades[g.ID] = g
	}
	lista, err := c.Repo.ListarChamadosCompartilhadosPorGrades(r.Context(), idsDeGrades(grades))
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, item := range lista {
		g := mapaGrades[item.Agendamento.GradeID]
		out[i] = views.ChamadoCompartilhadoParaJSON(item, &g.Grade, usuario.ID)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// GetMinhasGradesAtendidosHoje: GET /api/v1/atendente/grades/atendidos-hoje — consolidado.
func (c *Controllers) GetMinhasGradesAtendidosHoje(w http.ResponseWriter, r *http.Request) {
	usuario, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	lista, err := c.Repo.ListarAtendidosHojePorGrades(r.Context(), idsDeGrades(grades), usuario.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// GetMinhasGradesAusentesHoje: GET /api/v1/atendente/grades/ausentes-hoje — consolidado.
func (c *Controllers) GetMinhasGradesAusentesHoje(w http.ResponseWriter, r *http.Request) {
	_, grades, ok := c.minhasGrades(w, r)
	if !ok {
		return
	}
	lista, err := c.Repo.ListarAusentesHojePorGrades(r.Context(), idsDeGrades(grades))
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// GetSalaEspera: GET /api/v1/grades/{gradeId}/sala-espera — compartilhada entre atendentes
// da grade, ordenada pela regra de prioridade.
func (c *Controllers) GetSalaEspera(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarSalaEspera(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// PostChamarProximo: POST /api/v1/grades/{gradeId}/guiches/{guicheId}/chamar-proximo
func (c *Controllers) PostChamarProximo(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	guiche, err := c.Repo.BuscarGuichePorID(r.Context(), grade.ID, guicheID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
		return
	}
	if !guiche.Ativo {
		views.ResponderErro(w, http.StatusConflict, "guiche_inativo", "guiche esta inativo")
		return
	}

	a, chamada, err := c.Repo.ChamarProximo(r.Context(), grade.ID, guicheID, usuario.ID, grade.CooldownRechamadaMinutos)
	if err != nil {
		switch err {
		case repository.ErrNaoEncontrado:
			views.ResponderErro(w, http.StatusNotFound, "fila_vazia", "nenhum agendamento aguardando")
		case repository.ErrChamadaPendente:
			views.ResponderErro(w, http.StatusConflict, "chamada_pendente", "voce tem uma chamada pendente — aguarde o cooldown e rechame antes de chamar outra pessoa")
		case repository.ErrExistemOrfaos:
			views.ResponderErro(w, http.StatusConflict, "existem_chamados_pendentes", "existem chamados pendentes de outros atendentes — assuma um deles antes de chamar alguem novo")
		default:
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		}
		return
	}

	c.empurrarParaPainel(grade.ID, *a, guiche.Nome)

	views.ResponderJSON(w, http.StatusCreated, map[string]any{
		"agendamento": views.AgendamentoParaJSON(*a),
		"chamada":     map[string]any{"id": chamada.ID, "chamadoEm": chamada.ChamadoEm},
	})
}

func (c *Controllers) empurrarParaPainel(gradeID string, a domain.Agendamento, guicheNome string) {
	c.Painel.Empurrar(gradeID, painel.ItemExibicao{
		ChamadaID:  a.ID,
		Protocolo:  a.Protocolo,
		Nome:       a.NomeCidadao,
		GuicheNome: guicheNome,
	})
}

// PostRechamar: POST /api/v1/agendamentos/{id}/rechamar — só o dono (quem fez a última
// chamada) pode rechamar, e só depois do cooldown configurado da grade (decisão 20/09).
func (c *Controllers) PostRechamar(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")
	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado nesta unidade")
		return
	}
	if a.Status != domain.StatusChamado {
		views.ResponderErro(w, http.StatusConflict, "estado_invalido", "so e possivel rechamar quem esta chamado")
		return
	}
	info, err := c.Repo.UltimaChamadaDe(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if info.UsuarioID != usuario.ID {
		views.ResponderErro(w, http.StatusForbidden, "nao_e_dono", "apenas quem chamou pode rechamar")
		return
	}
	grade, err := c.Repo.BuscarGradePorID(r.Context(), a.GradeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	liberaEm := info.ChamadoEm.Add(time.Duration(grade.CooldownRechamadaMinutos) * time.Minute)
	if time.Now().Before(liberaEm) {
		views.ResponderErro(w, http.StatusConflict, "cooldown_ativo", "aguarde o tempo minimo entre chamadas antes de rechamar")
		return
	}

	chamada, err := c.Repo.Rechamar(r.Context(), agendamentoID, usuario.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	guiche, err := c.Repo.BuscarGuichePorID(r.Context(), a.GradeID, chamada.GuicheID)
	if err == nil {
		c.empurrarParaPainel(a.GradeID, *a, guiche.Nome)
	}
	views.ResponderJSON(w, http.StatusCreated, map[string]any{"id": chamada.ID, "chamadoEm": chamada.ChamadoEm})
}

// PostAtendido: POST /api/v1/agendamentos/{id}/atendido — só o dono (quem chamou por
// último, ou quem assumiu) pode marcar presente (decisão 20/09).
func (c *Controllers) PostAtendido(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")
	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado nesta unidade")
		return
	}
	info, err := c.Repo.UltimaChamadaDe(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if info.UsuarioID != usuario.ID {
		views.ResponderErro(w, http.StatusForbidden, "nao_e_dono", "apenas quem chamou pode marcar presente")
		return
	}
	linhas, err := c.Repo.MarcarAtendido(r.Context(), agendamentoID, time.Now())
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if linhas == 0 {
		views.ResponderErro(w, http.StatusConflict, "estado_invalido", "agendamento nao esta chamado")
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PostAusenciaManual: POST /api/v1/agendamentos/{id}/ausencia — botão de ausência manual
// (23/09), pro atendente marcar ausente na hora (ex: alguém avisou que o cidadão já foi
// embora) em vez de esperar o sweeper automático rodar até o fim do SLA de atendimento.
// Mesma regra de posse de atendido/rechamar — só quem chamou por último pode agir.
func (c *Controllers) PostAusenciaManual(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")
	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado nesta unidade")
		return
	}
	info, err := c.Repo.UltimaChamadaDe(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if info.UsuarioID != usuario.ID {
		views.ResponderErro(w, http.StatusForbidden, "nao_e_dono", "apenas quem chamou pode marcar ausencia")
		return
	}
	linhas, err := c.Repo.MarcarAusenteManual(r.Context(), agendamentoID, time.Now())
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if linhas == 0 {
		views.ResponderErro(w, http.StatusConflict, "estado_invalido", "agendamento nao esta chamado")
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// chamadoCompartilhadoParaJSON monta a linha do bloco "Chamando agora" — visível a todos os
// atendentes da grade, com "souDono" indicando se É este usuário quem pode agir nela
// (marcar presente / rechamar) e "ehOrfao" indicando se qualquer atendente ocioso pode
// assumir. Ver skill regras-negocio-fila.
// GetChamados: GET /api/v1/grades/{gradeId}/chamados — bloco "Chamando agora",
// compartilhado entre TODOS os atendentes da grade (decisão 20/09): todo mundo vê quem
// está sendo chamado e por qual guichê, mas só o dono ganha os botões de ação no frontend
// (a API já manda "souDono" pronto pra facilitar).
func (c *Controllers) GetChamados(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarChamadosCompartilhados(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, item := range lista {
		out[i] = views.ChamadoCompartilhadoParaJSON(item, grade, usuario.ID)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// PostAssumirChamada: POST /api/v1/agendamentos/{id}/assumir — um atendente ocioso assume
// um chamado órfão (abandonado por quem chamou originalmente) pro seu próprio guichê.
func (c *Controllers) PostAssumirChamada(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	agendamentoID := chi.URLParam(r, "id")
	var corpo struct {
		GuicheID string `json:"guicheId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.GuicheID == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "guicheId e obrigatorio")
		return
	}
	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado nesta unidade")
		return
	}
	guiche, err := c.Repo.BuscarGuichePorID(r.Context(), a.GradeID, corpo.GuicheID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
		return
	}
	if !guiche.Ativo {
		views.ResponderErro(w, http.StatusConflict, "guiche_inativo", "guiche esta inativo")
		return
	}

	a, chamada, err := c.Repo.AssumirChamada(r.Context(), a.GradeID, agendamentoID, corpo.GuicheID, usuario.ID)
	if err != nil {
		if err == repository.ErrNaoEhOrfao {
			views.ResponderErro(w, http.StatusConflict, "nao_e_orfao", "esse chamado ainda nao pode ser assumido")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	c.empurrarParaPainel(a.GradeID, *a, guiche.Nome)
	views.ResponderJSON(w, http.StatusCreated, map[string]any{
		"agendamento": views.AgendamentoParaJSON(*a),
		"chamada":     map[string]any{"id": chamada.ID, "chamadoEm": chamada.ChamadoEm},
	})
}

// GetAtendidosHoje: GET /api/v1/grades/{gradeId}/atendidos-hoje — bloco de consulta (lado
// direito da tela do atendente), só quem ESTE usuário atendeu hoje, sem ação.
func (c *Controllers) GetAtendidosHoje(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarAtendidosHoje(r.Context(), grade.ID, usuario.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// GetAusentesHoje: GET /api/v1/grades/{gradeId}/ausentes-hoje — aba "Ausentes" do bloco
// compartilhado "Sala de espera" do atendente (22/09: grade inteira, não filtra por quem
// chamou — ver comentário em ListarAusentesHoje).
func (c *Controllers) GetAusentesHoje(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelAtendente && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas atendente, gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarAusentesHoje(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.AgendamentosParaJSON(lista))
}

// GetHistorico: GET /api/v1/agendamentos/{id}/historico — "janela do cidadão".
func (c *Controllers) GetHistorico(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	agendamentoID := chi.URLParam(r, "id")
	a, err := c.Repo.BuscarAgendamentoPorID(r.Context(), agendamentoID)
	if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuario, a.UnidadeID) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "agendamento nao encontrado nesta unidade")
		return
	}
	chamadas, err := c.Repo.HistoricoDoAgendamento(r.Context(), agendamentoID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{
		"agendamento": views.AgendamentoParaJSON(*a),
		"chamadas":    chamadas,
	})
}
