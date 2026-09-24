package controllers

import (
	"net/http"

	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/views"
)

// GetDashboard: GET /api/v1/secretarias/{secretariaId}/dashboard — gestor/admin. Escopado
// pela secretaria inteira (22/09) — agrega todas as unidades dela.
func (c *Controllers) GetDashboard(w http.ResponseWriter, r *http.Request) {
	usuario, secretariaID, ok := c.exigirSecretaria(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	resumo, err := c.Repo.ResumoDoDia(r.Context(), secretariaID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, resumo)
}
