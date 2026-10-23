package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

type corpoSessaoPortal struct {
	Credencial string `json:"credencial"`
}

// PostSessaoPortal: POST /api/v1/sessao/portal — login integrado via SSO real da Conecta
// Cidades (22/10, ver skill auth-conecta-cidades pro desenho completo e as fontes oficiais).
// O frontend lê o `?token=` da URL (vindo do portal/painel administrativo da prefeitura) e
// manda aqui; a resposta é a mesma forma de PostLogin (sessão criada, usuário devolvido) —
// o frontend redireciona pelo papel igual já faz no login por senha.
//
// Ordem de validação (skill, seção 4) — a ordem IMPORTA: decide qual erro a pessoa vê e
// evita gastar uma busca de JWKS pra emissor desconhecido.
func (c *Controllers) PostSessaoPortal(w http.ResponseWriter, r *http.Request) {
	var corpo corpoSessaoPortal
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || strings.TrimSpace(corpo.Credencial) == "" {
		views.ResponderErro(w, http.StatusUnauthorized, "identidade_ausente", "credencial do portal ausente")
		return
	}
	// Sem a credencial de introspecção OU sem a audiência configurada, o login via portal
	// fica DESLIGADO por completo — nunca aceitamos um login SSO sem conseguir confirmar o
	// uso único depois (ver skill: "não usar introspection" está na lista de "implementação
	// incorreta, nunca faça" da própria doc oficial), nem sem checar `aud` (sem isso, um
	// token válido emitido pra OUTRA aplicação integrada ao mesmo emissor seria aceito aqui
	// também — achado por revisão de segurança, 22/10).
	if c.ConectaIntrospectToken == "" || c.ConectaAudiencia == "" {
		views.ResponderErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "login via portal nao configurado neste ambiente")
		return
	}

	// 1-2. Ler o iss SEM confiar, só pra saber contra qual prefeitura/JWKS checar depois.
	emissor, err := auth.LerEmissorSemVerificar(corpo.Credencial)
	if err != nil {
		views.ResponderErro(w, http.StatusUnauthorized, "identidade_invalida", "credencial invalida")
		return
	}

	// 3. Resolver iss -> prefeitura (comparação exata). Origem desconhecida = recusa seca.
	// O ID da prefeitura resolvida (não só a existência) é usado mais abaixo pra escopar a
	// resolução de secretaria/unidade por tenant (ver mapeamento_externo.go).
	prefeitura, err := c.Repo.BuscarPrefeituraPorIssuer(r.Context(), emissor)
	if err != nil {
		views.ResponderErro(w, http.StatusForbidden, "prefeitura_nao_atendida", "prefeitura nao cadastrada pra login via portal")
		return
	}

	// 4-5. Verificar assinatura (JWKS do emissor resolvido), iss, exp, aud.
	claims, err := auth.VerificarCredencialPortal(r.Context(), c.ChavesPublicas, emissor, c.ConectaAudiencia, corpo.Credencial)
	if err != nil {
		if errors.Is(err, auth.ErrJWKSIndisponivel) {
			views.ResponderErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "nao foi possivel validar a credencial agora")
		} else {
			views.ResponderErro(w, http.StatusUnauthorized, "identidade_invalida", "credencial invalida ou expirada")
		}
		return
	}

	// 6. Papel — só depois da assinatura conferir (recusar por "papel não reconhecido" a um
	// token forjado entregaria de graça o formato esperado pra quem não devia saber).
	papel, reconhecido := auth.MapearPapel(claims.Roles)
	if !reconhecido {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "papel do servidor nao reconhecido")
		return
	}
	// Admin NUNCA é concedido via SSO (achado por revisão de segurança, 22/10): o nosso
	// `admin` é GLOBAL (gerencia TODAS as prefeituras cadastradas, ver domain.Usuario) — mas
	// "admin" no token só significa "administrador DAQUELE município" na Conecta Cidades. Sem
	// essa trava, ser admin só da Smartândia (ambiente de teste) daria acesso de admin
	// GLOBAL, inclusive a dado real de outras prefeituras. Admin continua só manual (login
	// por senha, criado direto no banco/por outro admin) — nunca auto-provisionado aqui.
	if papel == domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "admin nao pode ser concedido via login integrado")
		return
	}

	if strings.TrimSpace(claims.Subject) == "" {
		views.ResponderErro(w, http.StatusUnauthorized, "credencial_sem_identificador", "token sem sub")
		return
	}
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		email = strings.TrimSpace(claims.PreferredUsername)
	}
	if email == "" {
		views.ResponderErro(w, http.StatusUnauthorized, "credencial_sem_identificador", "token sem email utilizavel")
		return
	}

	// 7. Introspecção — OBRIGATÓRIA, só entra em jogo depois de tudo acima já ter passado.
	// Confirma uso único (o token torna-se inválido pra qualquer chamada seguinte).
	if err := auth.Introspect(r.Context(), emissor, c.ConectaIntrospectToken, corpo.Credencial); err != nil {
		if errors.Is(err, auth.ErrIntrospectInativo) {
			views.ResponderErro(w, http.StatusUnauthorized, "identidade_invalida", "token ja utilizado ou expirado")
		} else {
			views.ResponderErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "nao foi possivel confirmar a credencial agora")
		}
		return
	}

	// Vínculo: gestor precisa de secretaria, atendente/recepcionista de unidade (via
	// department_ids/unit_ids -> mapeamento_externo, escopado pela prefeitura resolvida
	// acima). Admin não tem vínculo nenhum.
	var secretariaID, unidadeID *string
	switch papel {
	case domain.PapelGestor:
		secretariaID, err = c.Repo.ResolverSecretariaPorExternalIDs(r.Context(), prefeitura.ID, claims.DepartmentIDs)
	case domain.PapelAtendente, domain.PapelRecepcionista:
		unidadeID, err = c.Repo.ResolverUnidadePorExternalIDs(r.Context(), prefeitura.ID, claims.UnitIDs)
	}
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	if papel != domain.PapelAdmin && secretariaID == nil && unidadeID == nil {
		views.ResponderErro(w, http.StatusForbidden, "vinculo_nao_encontrado", "secretaria/unidade do servidor nao mapeada")
		return
	}

	// 8. Upsert — papel/vínculo são REAPLICADOS a cada entrada. ExternalIssuer (= emissor já
	// resolvido) e EmailVerificado viajam junto — ver os dois travamentos de segurança
	// documentados em UpsertUsuarioPortal (escopo de tenant + nunca rebindar vínculo externo
	// já estabelecido).
	usuario, err := c.Repo.UpsertUsuarioPortal(r.Context(), repository.DadosPortal{
		ExternalIssuer:  emissor,
		ExternalSub:     claims.Subject,
		Nome:            strings.TrimSpace(claims.Name),
		Email:           email,
		EmailVerificado: claims.EmailVerified,
		Papel:           papel,
		SecretariaID:    secretariaID,
		UnidadeID:       unidadeID,
		PrefeituraID:    prefeitura.ID,
	})
	if err != nil {
		if errors.Is(err, repository.ErrVinculoExternoConflitante) {
			views.ResponderErro(w, http.StatusForbidden, "credencial_sem_identificador", "e-mail ja vinculado a outra identidade")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
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
	_ = c.Repo.AtualizarUltimoAcesso(r.Context(), usuario.ID)
	auth.DefinirCookieSessao(w, token, c.SessaoDuracao, c.CookieSeguro || auth.RequisicaoViaHTTPS(r))
	views.ResponderJSON(w, http.StatusOK, c.usuarioParaJSON(r.Context(), usuario))
}
