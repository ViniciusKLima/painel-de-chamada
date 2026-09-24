package controllers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

// GetGrades: GET /api/v1/secretarias/{secretariaId}/grades — tela "Grade de horário" do
// gestor/admin, lista de todas as filas/serviços de TODAS as unidades da secretaria (22/09
// — gestor deixou de ser vinculado a uma unidade só, ver skill modelo-dados).
func (c *Controllers) GetGrades(w http.ResponseWriter, r *http.Request) {
	usuario, secretariaID, ok := c.exigirSecretaria(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	grades, err := c.Repo.ListarGradesPorSecretaria(r.Context(), secretariaID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GradesParaJSON(grades))
}

// GetGradesDaUnidade: GET /api/v1/unidades/{unidadeId}/grades — recepcionista/gestor/admin.
// Só as grades DAQUELA unidade (não a secretaria inteira) — usado pelo formulário de
// encaixe da recepção, que precisa saber pra qual serviço/fila é o walk-in dentro do local
// físico onde a pessoa chegou. Distinto de GetGrades (secretaria inteira, tela de gestão).
func (c *Controllers) GetGradesDaUnidade(w http.ResponseWriter, r *http.Request) {
	usuario, unidadeID, ok := c.exigirUnidade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelRecepcionista && usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas recepcao, gestor ou admin")
		return
	}
	grades, err := c.Repo.ListarGradesPorUnidade(r.Context(), unidadeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	// Recepcionista só vê as grades onde pode de fato cadastrar encaixe (23/09) — evita
	// oferecer no formulário uma opção que o POST /encaixe vai recusar de qualquer forma.
	// Gestor/admin continuam vendo todas (podem cadastrar encaixe em qualquer uma).
	if usuario.Papel == domain.PapelRecepcionista {
		permitidas := make([]domain.Grade, 0, len(grades))
		for _, g := range grades {
			if g.PermiteEncaixeRecepcao {
				permitidas = append(permitidas, g)
			}
		}
		grades = permitidas
		// Se a recepcionista foi alocada em grades específicas (23/09, "recepção pode ser
		// alocada em mais de uma grade"), filtra mais ainda pra só essas — sem nenhuma
		// alocação, continua vendo/cadastrando em qualquer grade da unidade (comportamento
		// original, unidade inteira).
		alocadas, err := c.Repo.ListarGradesDoUsuario(r.Context(), usuario.ID)
		if err != nil {
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
			return
		}
		if len(alocadas) > 0 {
			alocadasIDs := make(map[string]bool, len(alocadas))
			for _, g := range alocadas {
				alocadasIDs[g.ID] = true
			}
			restritas := make([]domain.Grade, 0, len(grades))
			for _, g := range grades {
				if alocadasIDs[g.ID] {
					restritas = append(restritas, g)
				}
			}
			grades = restritas
		}
	}
	views.ResponderJSON(w, http.StatusOK, views.GradesParaJSON(grades))
}

// DeleteGrade: DELETE /api/v1/grades/{gradeId} — só gestor/admin (23/09, "quero que seja
// possível excluir uma grade"). Soft-delete (ver ExcluirGrade) — histórico intacto.
func (c *Controllers) DeleteGrade(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	if err := c.Repo.ExcluirGrade(r.Context(), grade.ID); err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PostReativarGrade: POST /api/v1/grades/{gradeId}/reativar — só gestor/admin (23/09,
// auditoria de reativação). Volta a grade pra "Grade de horário" e pra qualquer roteamento de
// importação futuro — mas sem os atendentes/recepcionistas que estavam alocados antes de
// excluir (ver nota em Repo.ReativarGrade), precisa realocar de novo se for o caso.
func (c *Controllers) PostReativarGrade(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	atualizada, err := c.Repo.ReativarGrade(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GradeParaJSON(atualizada))
}

// DeleteGuiche: DELETE /api/v1/grades/{gradeId}/guiches/{guicheId} — só gestor/admin (23/09,
// "quero que seja possível apagar um guichê ou sala"). Bloqueado se for o último guichê/sala
// não-excluído da grade (ver ExcluirGuiche/ErrUltimoGuiche — toda grade precisa de ao menos
// um pra o atendente ter onde escolher).
func (c *Controllers) DeleteGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	err := c.Repo.ExcluirGuiche(r.Context(), grade.ID, guicheID)
	if err != nil {
		if err == repository.ErrUltimoGuiche {
			views.ResponderErro(w, http.StatusConflict, "ultimo_guiche", "a grade precisa de pelo menos um guiche ou sala")
			return
		}
		if err == repository.ErrNaoEncontrado {
			views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PostReativarGuiche: POST /api/v1/grades/{gradeId}/guiches/{guicheId}/reativar — só gestor/
// admin (23/09, auditoria de reativação). Volta com o `ativo` que tinha antes de ser
// excluído — ver nota em Repo.ReativarGuiche.
func (c *Controllers) PostReativarGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	guiche, err := c.Repo.ReativarGuiche(r.Context(), grade.ID, guicheID)
	if err != nil {
		if err == repository.ErrNaoEncontrado {
			views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GuicheParaJSON(*guiche))
}

// GetConfiguracaoGrade: GET /api/v1/grades/{gradeId}/configuracao — só gestor. A grade em
// si + todos os guichês/salas (inclusive inativos, pra poder reativar).
func (c *Controllers) GetConfiguracaoGrade(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	guiches, err := c.Repo.ListarTodosGuiches(r.Context(), grade.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	guichesJSON := make([]map[string]any, len(guiches))
	for i, g := range guiches {
		guichesJSON[i] = views.GuicheParaJSON(g)
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{
		"grade":   views.GradeParaJSON(grade),
		"guiches": guichesJSON,
	})
}

type corpoConfiguracaoGrade struct {
	Nome                             string `json:"nome"`
	SLAChegadaMinutos                int    `json:"slaChegadaMinutos"`
	SLAAtendimentoMinutos            int    `json:"slaAtendimentoMinutos"`
	DuracaoChamadaPainelSegundos     int    `json:"duracaoChamadaPainelSegundos"`
	DuracaoChamadaPainelFilaSegundos int    `json:"duracaoChamadaPainelFilaSegundos"`
	RepeticoesChamada                int    `json:"repeticoesChamada"`
	IntervaloRepeticaoSegundos       int    `json:"intervaloRepeticaoSegundos"`
	CooldownRechamadaMinutos         int    `json:"cooldownRechamadaMinutos"`
	PermiteEncaixeRecepcao           bool   `json:"permiteEncaixeRecepcao"`
	Ativo                            bool   `json:"ativo"`
}

// PatchConfiguracaoGrade: PATCH /api/v1/grades/{gradeId}/configuracao — só gestor.
func (c *Controllers) PatchConfiguracaoGrade(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	var corpo corpoConfiguracaoGrade
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "corpo da requisicao invalido")
		return
	}
	if corpo.Nome == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e obrigatorio")
		return
	}
	if corpo.SLAChegadaMinutos <= 0 || corpo.SLAAtendimentoMinutos <= 0 ||
		corpo.DuracaoChamadaPainelSegundos <= 0 || corpo.DuracaoChamadaPainelFilaSegundos <= 0 {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "todos os valores devem ser maiores que zero")
		return
	}
	if corpo.RepeticoesChamada <= 0 || corpo.RepeticoesChamada > 10 {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "repeticoes da chamada deve ser entre 1 e 10")
		return
	}
	if corpo.IntervaloRepeticaoSegundos <= 0 || corpo.IntervaloRepeticaoSegundos > 120 {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "intervalo de repeticao deve ser entre 1 e 120 segundos")
		return
	}
	if corpo.CooldownRechamadaMinutos <= 0 || corpo.CooldownRechamadaMinutos > 30 {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "tempo minimo entre chamadas deve ser entre 1 e 30 minutos")
		return
	}
	if _, err := c.Repo.AtualizarNomeEAtivoGrade(r.Context(), grade.ID, corpo.Nome, corpo.Ativo); err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	atualizada, err := c.Repo.AtualizarConfiguracaoGrade(r.Context(), grade.ID,
		corpo.SLAChegadaMinutos, corpo.SLAAtendimentoMinutos,
		corpo.DuracaoChamadaPainelSegundos, corpo.DuracaoChamadaPainelFilaSegundos,
		corpo.RepeticoesChamada, corpo.IntervaloRepeticaoSegundos, corpo.CooldownRechamadaMinutos,
		corpo.PermiteEncaixeRecepcao)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GradeParaJSON(atualizada))
}

func tipoGuicheValido(tipo string) (domain.TipoGuiche, bool) {
	switch domain.TipoGuiche(tipo) {
	case domain.TipoGuicheGuiche, domain.TipoGuicheSala:
		return domain.TipoGuiche(tipo), true
	default:
		return "", false
	}
}

type corpoNovoGuiche struct {
	Nome       string  `json:"nome"`
	Tipo       string  `json:"tipo"`
	Andar      *string `json:"andar"`
	Capacidade int     `json:"capacidade"`
}

// PostGuiche: POST /api/v1/grades/{gradeId}/guiches — só gestor. "Tipo" e "capacidade"
// são metadados descritivos pra organizar a tela (se é um guichê de atendimento individual
// ou uma sala, quantas pessoas cabem) — não mudam a regra de "um chamado por vez" do
// chamar-próximo, que continua igual pros dois tipos (ver skill regras-negocio-fila).
func (c *Controllers) PostGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	var corpo corpoNovoGuiche
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e obrigatorio")
		return
	}
	tipo := domain.TipoGuicheGuiche
	if corpo.Tipo != "" {
		t, ok := tipoGuicheValido(corpo.Tipo)
		if !ok {
			views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "tipo deve ser 'guiche' ou 'sala'")
			return
		}
		tipo = t
	}
	capacidade := corpo.Capacidade
	if capacidade <= 0 {
		capacidade = 1
	}
	guiche, err := c.Repo.CriarGuiche(r.Context(), grade.ID, repository.NovoGuicheOuSala{
		Nome: corpo.Nome, Tipo: tipo, Andar: corpo.Andar, Capacidade: capacidade,
	})
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, views.GuicheParaJSON(*guiche))
}

type corpoAtivoGuiche struct {
	Ativo bool `json:"ativo"`
}

// PatchGuiche: PATCH /api/v1/grades/{gradeId}/guiches/{guicheId} — liga/desliga rápido
// (usado pelo botão "Desativar/Reativar" fora do bloco expandido), só gestor.
func (c *Controllers) PatchGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	var corpo corpoAtivoGuiche
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "corpo da requisicao invalido")
		return
	}
	linhas, err := c.Repo.AtualizarAtivoGuiche(r.Context(), grade.ID, guicheID, corpo.Ativo)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if linhas == 0 {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

type corpoEdicaoGuiche struct {
	Nome       string  `json:"nome"`
	Tipo       string  `json:"tipo"`
	Andar      *string `json:"andar"`
	Capacidade int     `json:"capacidade"`
	Ativo      bool    `json:"ativo"`
}

// PutGuiche: PUT /api/v1/grades/{gradeId}/guiches/{guicheId} — edição completa do bloco
// expandido na tela de configurações da grade, só gestor.
func (c *Controllers) PutGuiche(w http.ResponseWriter, r *http.Request) {
	usuario, grade, ok := c.exigirGrade(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor")
		return
	}
	guicheID := chi.URLParam(r, "guicheId")
	var corpo corpoEdicaoGuiche
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e obrigatorio")
		return
	}
	tipo, ok := tipoGuicheValido(corpo.Tipo)
	if !ok {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "tipo deve ser 'guiche' ou 'sala'")
		return
	}
	capacidade := corpo.Capacidade
	if capacidade <= 0 {
		capacidade = 1
	}
	guiche, err := c.Repo.AtualizarGuiche(r.Context(), grade.ID, guicheID, repository.AtualizacaoGuiche{
		Nome: corpo.Nome, Tipo: tipo, Andar: corpo.Andar, Capacidade: capacidade, Ativo: corpo.Ativo,
	})
	if err != nil {
		if err == repository.ErrNaoEncontrado {
			views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "guiche nao encontrado nesta grade")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.GuicheParaJSON(*guiche))
}
