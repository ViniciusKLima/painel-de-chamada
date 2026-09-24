package controllers

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

// GetPaineisPublicoDaUnidade: GET /api/v1/unidades/{unidadeId}/paineis-publico — PÚBLICO,
// sem sessão (23/09, pedido explícito: "quero um acesso a essa aba de painéis sem ser pelo
// gestor... quando for acessar esse painel pela tv do local"). Só o essencial pra montar a
// central de painéis de UMA unidade física sem exigir login: nome da unidade + id/nome de
// cada grade ATIVA dela (grade inativa não tem fila rodando, não faz sentido oferecer).
// Nenhum dado de agendamento/cidadão aqui — isso só entra depois, no próprio GetPainel de
// cada grade, que já era público.
func (c *Controllers) GetPaineisPublicoDaUnidade(w http.ResponseWriter, r *http.Request) {
	unidadeID := chi.URLParam(r, "unidadeId")
	unidade, err := c.Repo.BuscarUnidadePorID(r.Context(), unidadeID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "unidade nao encontrada")
		return
	}
	grades, err := c.Repo.ListarGradesPorUnidade(r.Context(), unidadeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.PaineisPublicoDaUnidadeResponse(unidade.Nome, grades))
}

// GetPainel: GET /api/v1/painel/{gradeId} — PÚBLICO, sem sessão nenhuma. Escopo por grade
// (22/09) — cada grade (serviço dentro de uma unidade) tem seu próprio painel/link público,
// já que cada uma tem sua fila e guichês independentes. Decisão revertida em 20/09 a pedido
// do dono do produto: exibe nome COMPLETO (não mais parcial) — nunca retorna CPF/telefone,
// só protocolo, nome e guichê.
//
// O controller só orquestra: busca grade/unidade, consulta a fila de exibição em memória e o
// histórico de chamadas, resolve as cores da prefeitura — e entrega tudo já resolvido pra
// `views.PainelResponse` decidir o formato (24/09, refactor pra MVC — antes toda essa decisão
// de formato vivia misturada aqui dentro do handler).
func (c *Controllers) GetPainel(w http.ResponseWriter, r *http.Request) {
	gradeID := chi.URLParam(r, "gradeId")

	grade, err := c.Repo.BuscarGradePorID(r.Context(), gradeID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "grade nao encontrada")
		return
	}
	unidade, err := c.Repo.BuscarUnidadePorID(r.Context(), grade.UnidadeID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}

	duracaoNormal := time.Duration(grade.DuracaoChamadaPainelSegundos) * time.Second
	duracaoFila := time.Duration(grade.DuracaoChamadaPainelFilaSegundos) * time.Second
	atual, expiraEm, duracaoTotal := c.Painel.Atual(gradeID, duracaoNormal, duracaoFila)

	ultimas, err := c.Repo.UltimasChamadas(r.Context(), gradeID, 10)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}

	// Modo "aguardando" (23/09, pedido do dono do produto) só entra em jogo quando NÃO há
	// nenhuma chamada nova ativa em destaque — nunca compete com a exibição normal.
	var aguardando []repository.AguardandoExibicao
	if atual == nil {
		aguardando, err = c.Repo.ListarAguardandoExibicao(r.Context(), gradeID, 4)
		if err != nil {
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
			return
		}
	}

	// Cores de identidade da prefeitura (22/09, configuráveis pelo admin) — resolvidas via
	// unidade -> secretaria -> prefeitura, mesma cadeia usada em Controllers.usuarioParaJSON.
	// O painel é público (sem sessão), então precisa resolver isso aqui, não reaproveita a
	// sessão de ninguém.
	var corDestaque, corClara any
	if secretaria, err := c.Repo.BuscarSecretariaPorID(r.Context(), unidade.SecretariaID); err == nil {
		if prefeitura, err := c.Repo.BuscarPrefeituraPorID(r.Context(), secretaria.PrefeituraID); err == nil {
			corDestaque = prefeitura.CorDestaque
			corClara = prefeitura.CorClara
		}
	}

	views.ResponderJSON(w, http.StatusOK, views.PainelResponse(grade, unidade.Nome, atual, expiraEm, duracaoTotal, aguardando, ultimas, corDestaque, corClara))
}
