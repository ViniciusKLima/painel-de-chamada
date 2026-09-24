package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

// GetUsuarios: GET /api/v1/secretarias/{secretariaId}/usuarios — gestor/admin. Lista todo
// mundo que trabalha na secretaria (o próprio gestor, e todo recepcionista/atendente de
// qualquer unidade dela — 22/09, ver skill modelo-dados).
func (c *Controllers) GetUsuarios(w http.ResponseWriter, r *http.Request) {
	usuario, secretariaID, ok := c.exigirSecretaria(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	lista, err := c.Repo.ListarUsuariosPorSecretaria(r.Context(), secretariaID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	ids := make([]string, len(lista))
	for i, u := range lista {
		ids[i] = u.ID
	}
	gradesPorUsuario, err := c.Repo.GradesPorUsuario(r.Context(), ids)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, u := range lista {
		out[i] = views.UsuarioInternoParaJSON(u, gradesPorUsuario[u.ID])
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

type corpoAtualizacaoUsuario struct {
	Nome     string   `json:"nome"`
	Email    string   `json:"email"`
	Papel    string   `json:"papel"`
	Senha    string   `json:"senha"`
	GradeIDs []string `json:"gradeIds"`
}

func papelValido(papel string) (domain.Papel, bool) {
	switch domain.Papel(papel) {
	case domain.PapelGestor, domain.PapelAtendente, domain.PapelRecepcionista:
		return domain.Papel(papel), true
	default:
		return "", false
	}
}

// usuarioAlvoAcessivel decide se `usuarioLogado` pode editar `alvo` (ver
// internal/auth/permissoes.go): admin sempre pode; gestor só dentro da própria secretaria —
// direto se o alvo também tiver secretaria_id igual (outro gestor), ou via a unidade do
// alvo (recepcionista/atendente) pertencer à secretaria do gestor.
func (c *Controllers) usuarioAlvoAcessivel(ctx context.Context, usuarioLogado, alvo *domain.Usuario) bool {
	if auth.EhAdmin(usuarioLogado) {
		return true
	}
	if usuarioLogado.Papel != domain.PapelGestor || usuarioLogado.SecretariaID == nil {
		return false
	}
	if alvo.SecretariaID != nil {
		return *alvo.SecretariaID == *usuarioLogado.SecretariaID
	}
	if alvo.UnidadeID != nil {
		return c.usuarioPodeAcessarUnidade(ctx, usuarioLogado, *alvo.UnidadeID)
	}
	return false
}

// PatchUsuario: PATCH /api/v1/usuarios/{usuarioId} — gestor (só dentro da própria
// secretaria) ou admin (qualquer usuário). Nome/email sempre atualizados; senha só troca se
// vier preenchida; gradeId realoca o atendente pra outra fila (vazio = sem grade, válido
// pra gestor/recepcionista).
func (c *Controllers) PatchUsuario(w http.ResponseWriter, r *http.Request) {
	usuarioLogado := auth.UsuarioDoContexto(r.Context())
	if usuarioLogado.Papel != domain.PapelGestor && usuarioLogado.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	usuarioID := chi.URLParam(r, "usuarioId")
	alvo, err := c.Repo.BuscarUsuarioPorID(r.Context(), usuarioID)
	if err != nil || !c.usuarioAlvoAcessivel(r.Context(), usuarioLogado, alvo) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "usuario nao encontrado nesta secretaria")
		return
	}

	var corpo corpoAtualizacaoUsuario
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" || corpo.Email == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e email sao obrigatorios")
		return
	}
	papel, ok := papelValido(corpo.Papel)
	if !ok {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "papel deve ser 'gestor', 'atendente' ou 'recepcionista'")
		return
	}
	// Gestor não pode mudar o próprio cargo (pedido explícito do dono do produto) — evita
	// ele se rebaixar/travar o próprio acesso sem outro gestor por perto pra reverter. Editar
	// nome/email/senha de si mesmo continua permitido normalmente.
	if usuarioID == usuarioLogado.ID && usuarioLogado.Papel == domain.PapelGestor && papel != usuarioLogado.Papel {
		views.ResponderErro(w, http.StatusForbidden, "nao_pode_mudar_proprio_cargo", "voce nao pode mudar o proprio cargo")
		return
	}

	// Cada grade alocada precisa pertencer a uma unidade que o usuário logado acessa
	// (mesma checagem de sempre, agora em loop — 23/09, "atendente ou recepção podem ser
	// alocados em mais de uma grade"). Lista vazia é válida (gestor/recepcionista sem
	// restrição, ou atendente ainda não alocado).
	//
	// Atendente PODE ter grades de unidades DIFERENTES (exemplo real confirmado pelo dono do
	// produto: um atendente com grades na Unidade A e na Unidade B) — por isso a checagem
	// acima só valida contra a SECRETARIA do editor, não contra uma unidade fixa.
	//
	// Recepcionista é diferente: ela trabalha o balcão de UMA unidade física só (decisão de
	// 22/09, não revertida aqui), e a lista de grades dela é só um FILTRO de quais grades
	// pode cadastrar encaixe DENTRO dessa mesma unidade (ver GetGradesDaUnidade) — uma grade
	// de outra unidade nunca apareceria pra ela de qualquer jeito (ficaria "morta", nunca
	// usada), então bloqueamos aqui em vez de deixar um vínculo sem efeito nenhum acumular
	// silenciosamente (auditoria 23/09, "verifique... dados duplicados ou inconsistentes").
	for _, gradeID := range corpo.GradeIDs {
		grade, err := c.Repo.BuscarGradePorID(r.Context(), gradeID)
		if err != nil || !c.usuarioPodeAcessarUnidade(r.Context(), usuarioLogado, grade.UnidadeID) {
			views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "grade nao encontrada nesta secretaria")
			return
		}
		if papel == domain.PapelRecepcionista && alvo.UnidadeID != nil && grade.UnidadeID != *alvo.UnidadeID {
			views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "recepcionista so pode ser alocada a grades da propria unidade")
			return
		}
	}

	a := repository.AtualizacaoUsuario{Nome: corpo.Nome, Email: corpo.Email, Papel: papel, GradeIDs: corpo.GradeIDs}
	if corpo.Senha != "" {
		hash, err := auth.HashSenha(corpo.Senha)
		if err != nil {
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
			return
		}
		a.NovaSenhaHash = &hash
	}

	atualizado, err := c.Repo.AtualizarUsuario(r.Context(), usuarioID, a)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	grades, err := c.Repo.ListarGradesDoUsuario(r.Context(), atualizado.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.UsuarioInternoParaJSON(*atualizado, grades))
}

// DeleteUsuario: DELETE /api/v1/usuarios/{usuarioId} — gestor (só dentro da própria
// secretaria) ou admin (23/09, "quero que seja possível excluir um usuário"). Soft-delete
// (`ativo = false`, ver Repo.ExcluirUsuario) — histórico de chamadas continua intacto.
func (c *Controllers) DeleteUsuario(w http.ResponseWriter, r *http.Request) {
	usuarioLogado := auth.UsuarioDoContexto(r.Context())
	if usuarioLogado.Papel != domain.PapelGestor && usuarioLogado.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	usuarioID := chi.URLParam(r, "usuarioId")
	alvo, err := c.Repo.BuscarUsuarioPorID(r.Context(), usuarioID)
	if err != nil || !c.usuarioAlvoAcessivel(r.Context(), usuarioLogado, alvo) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "usuario nao encontrado nesta secretaria")
		return
	}
	if usuarioID == usuarioLogado.ID {
		views.ResponderErro(w, http.StatusForbidden, "nao_pode_excluir_a_si_mesmo", "voce nao pode excluir a propria conta")
		return
	}
	if err := c.Repo.ExcluirUsuario(r.Context(), usuarioID); err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// PostReativarUsuario: POST /api/v1/usuarios/{usuarioId}/reativar — gestor (só dentro da
// própria secretaria) ou admin (23/09, auditoria de reativação: "excluir" nunca tinha um
// caminho de volta na UI — só mexendo direto no banco). Mesma checagem de acesso que
// PatchUsuario/DeleteUsuario, reaproveitada.
func (c *Controllers) PostReativarUsuario(w http.ResponseWriter, r *http.Request) {
	usuarioLogado := auth.UsuarioDoContexto(r.Context())
	if usuarioLogado.Papel != domain.PapelGestor && usuarioLogado.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	usuarioID := chi.URLParam(r, "usuarioId")
	alvo, err := c.Repo.BuscarUsuarioPorID(r.Context(), usuarioID)
	if err != nil || !c.usuarioAlvoAcessivel(r.Context(), usuarioLogado, alvo) {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "usuario nao encontrado nesta secretaria")
		return
	}
	if err := c.Repo.ReativarUsuario(r.Context(), usuarioID); err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	atualizado, err := c.Repo.BuscarUsuarioPorID(r.Context(), usuarioID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	grades, err := c.Repo.ListarGradesDoUsuario(r.Context(), atualizado.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.UsuarioInternoParaJSON(*atualizado, grades))
}
