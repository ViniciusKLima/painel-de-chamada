package controllers

import (
	"context"
	"encoding/json"
	"net/http"

	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

type corpoLogin struct {
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// PostLogin: POST /api/v1/login — login do MVP (email+senha), ver skill auth-login-mvp.
// Não detalha se foi o email ou a senha que errou, de propósito.
func (c *Controllers) PostLogin(w http.ResponseWriter, r *http.Request) {
	var corpo corpoLogin
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "corpo da requisicao invalido")
		return
	}

	usuario, err := c.Repo.BuscarUsuarioPorEmail(r.Context(), corpo.Email)
	if err != nil || usuario.SenhaHash == nil || !auth.VerificarSenha(*usuario.SenhaHash, corpo.Senha) {
		views.ResponderErro(w, http.StatusUnauthorized, "identidade_invalida", "email ou senha invalidos")
		return
	}
	if !usuario.Ativo {
		views.ResponderErro(w, http.StatusForbidden, "usuario_desativado", "acesso revogado")
		return
	}

	token, err := auth.CriarSessao(c.SessaoSegredo, usuario.ID, c.SessaoDuracao)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao criar sessao")
		return
	}
	// Best-effort: se falhar, não impede o login — é só o registro de "último acesso" da
	// tela "Usuários internos" do gestor (22/09), não uma etapa crítica da autenticação.
	_ = c.Repo.AtualizarUltimoAcesso(r.Context(), usuario.ID)
	auth.DefinirCookieSessao(w, token, c.SessaoDuracao, c.CookieSeguro)
	views.ResponderJSON(w, http.StatusOK, c.usuarioParaJSON(r.Context(), usuario))
}

func (c *Controllers) GetSessao(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	views.ResponderJSON(w, http.StatusOK, c.usuarioParaJSON(r.Context(), usuario))
}

func (c *Controllers) DeleteSessao(w http.ResponseWriter, r *http.Request) {
	auth.RemoverCookieSessao(w, c.CookieSeguro)
	w.WriteHeader(http.StatusNoContent)
}

// usuarioParaJSON resolve o que cada papel precisa pra navegar (22/09, ver skill
// modelo-dados sobre os vínculos por papel): admin não tem nem unidade nem secretaria;
// gestor tem secretariaId direto (SecretariaID no domínio); recepcionista/atendente têm
// unidadeId (e resolvem secretariaId a partir dela); atendente também tem gradeId. Cores de
// identidade da prefeitura (destaque/clara, configuráveis pelo admin) vêm junto pra o
// frontend aplicar o tema em runtime — nulas pra admin, que não pertence a uma prefeitura
// específica.
func (c *Controllers) usuarioParaJSON(ctx context.Context, u *domain.Usuario) map[string]any {
	var unidadeID, unidadeNome, secretariaID any
	if u.UnidadeID != nil {
		unidadeID = *u.UnidadeID
		if unidade, err := c.Repo.BuscarUnidadePorID(ctx, *u.UnidadeID); err == nil {
			secretariaID = unidade.SecretariaID
			unidadeNome = unidade.Nome
		}
	}
	if u.SecretariaID != nil {
		secretariaID = *u.SecretariaID
	}
	// Grades alocadas (23/09, "atendente ou recepção podem ser alocados em mais de uma
	// grade") — não é mais um gradeId/gradeNome fixo na sessão. Correção da mesma data
	// (auditoria pedida pelo dono do produto): o atendente NÃO escolhe uma grade pra
	// trabalhar — ele vê e trabalha em TODAS as que estiver alocado, consolidadas (o gate de
	// seleção foi removido do frontend). `unidadeId`/`unidadeNome` vêm por grade (não só o
	// `unidadeId` singular do usuário) porque um atendente pode legitimamente ter grades de
	// unidades DIFERENTES — o painel/operacional agrupa por essa unidade de cada grade, não
	// pela unidade fixa da sessão. Lista vazia pra quem não tem papel escopado por grade.
	var grades []repository.GradeComUnidade
	if u.Papel == domain.PapelAtendente || u.Papel == domain.PapelRecepcionista {
		if alocadas, err := c.Repo.ListarGradesDoUsuario(ctx, u.ID); err == nil {
			grades = alocadas
		}
	}
	var corDestaque, corClara, prefeituraNome any
	if sid, ok := secretariaID.(string); ok {
		if secretaria, err := c.Repo.BuscarSecretariaPorID(ctx, sid); err == nil {
			if prefeitura, err := c.Repo.BuscarPrefeituraPorID(ctx, secretaria.PrefeituraID); err == nil {
				corDestaque = prefeitura.CorDestaque
				corClara = prefeitura.CorClara
				prefeituraNome = prefeitura.Nome
			}
		}
	}
	// A partir daqui é pura decisão de formato — nenhuma consulta a mais, por isso vira uma
	// chamada à camada de View (internal/views/usuario_view.go), não fica aqui no controller.
	return views.UsuarioParaJSON(u, unidadeID, unidadeNome, secretariaID, grades, corDestaque, corClara, prefeituraNome)
}
