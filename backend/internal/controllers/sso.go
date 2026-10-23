package controllers

import (
	"encoding/json"
	"errors"
	"log"
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

// ssoErro loga no stdout do servidor (achado real, 23/10: diagnosticar um login via portal
// que falha era só "tenta de novo e vê qual tela aparece" — sem log nenhum do lado do
// servidor, cada etapa de validação era uma caixa preta) e devolve ao CLIENTE só a mensagem
// pública genérica — o `detalheInterno` (texto de erro de biblioteca, nome de role não
// reconhecida etc.) fica só no log do servidor. ACHADO POR REVISÃO DE SEGURANÇA (23/10,
// mesmo dia): a 1ª versão desta função concatenava `err.Error()` direto na mensagem pública
// em vários pontos — informação interna (detalhe de validação de JWT, de uma chamada HTTP
// pra Conecta Cidades etc.) foi pra fora, no corpo da resposta HTTP, não só no log.
func ssoErro(w http.ResponseWriter, status int, codigo, mensagemPublica, detalheInterno, emissor string) {
	log.Printf("sso: falhou em %q (emissor=%q, status=%d): %s", codigo, emissor, status, detalheInterno)
	views.ResponderErro(w, status, codigo, mensagemPublica)
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
		ssoErro(w, http.StatusUnauthorized, "identidade_ausente", "credencial do portal ausente", "corpo da requisicao sem credencial valida", "")
		return
	}
	// Sem a credencial de introspecção OU sem a audiência configurada, o login via portal
	// fica DESLIGADO por completo — nunca aceitamos um login SSO sem conseguir confirmar o
	// uso único depois (ver skill: "não usar introspection" está na lista de "implementação
	// incorreta, nunca faça" da própria doc oficial), nem sem checar `aud` (sem isso, um
	// token válido emitido pra OUTRA aplicação integrada ao mesmo emissor seria aceito aqui
	// também — achado por revisão de segurança, 22/10).
	if c.ConectaIntrospectToken == "" || c.ConectaAudiencia == "" {
		ssoErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "login via portal nao configurado neste ambiente", "CONECTA_INTROSPECT_API_TOKEN ou CONECTA_AUDIENCIA vazios", "")
		return
	}

	// 1-2. Ler o iss SEM confiar, só pra saber contra qual prefeitura/JWKS checar depois.
	emissor, err := auth.LerEmissorSemVerificar(corpo.Credencial)
	if err != nil {
		ssoErro(w, http.StatusUnauthorized, "identidade_invalida", "credencial invalida", err.Error(), "")
		return
	}

	// 3. Resolver iss -> prefeitura (comparação exata). Origem desconhecida = recusa seca.
	// O ID da prefeitura resolvida (não só a existência) é usado mais abaixo pra escopar a
	// resolução de secretaria/unidade por tenant (ver mapeamento_externo.go).
	prefeitura, err := c.Repo.BuscarPrefeituraPorIssuer(r.Context(), emissor)
	if err != nil {
		ssoErro(w, http.StatusForbidden, "prefeitura_nao_atendida", "prefeitura nao cadastrada pra login via portal", err.Error(), emissor)
		return
	}

	// 4-5. Verificar assinatura (JWKS do emissor resolvido), iss, exp, aud.
	claims, err := auth.VerificarCredencialPortal(r.Context(), c.ChavesPublicas, emissor, c.ConectaAudiencia, corpo.Credencial)
	if err != nil {
		if errors.Is(err, auth.ErrJWKSIndisponivel) {
			ssoErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "nao foi possivel validar a credencial agora", err.Error(), emissor)
		} else {
			ssoErro(w, http.StatusUnauthorized, "identidade_invalida", "credencial invalida ou expirada", err.Error(), emissor)
		}
		return
	}

	// 6. Papel — só depois da assinatura conferir (recusar por "papel não reconhecido" a um
	// token forjado entregaria de graça o formato esperado pra quem não devia saber).
	papel, reconhecido := auth.MapearPapel(claims.Roles)
	if !reconhecido {
		ssoErro(w, http.StatusForbidden, "papel_sem_acesso", "papel do servidor nao reconhecido", "roles="+strings.Join(claims.Roles, ","), emissor)
		return
	}
	// Admin NUNCA é concedido via SSO (achado por revisão de segurança, 22/10): o nosso
	// `admin` é GLOBAL (gerencia TODAS as prefeituras cadastradas, ver domain.Usuario) — mas
	// "admin" no token só significa "administrador DAQUELE município" na Conecta Cidades. Sem
	// essa trava, ser admin só da Smartândia (ambiente de teste) daria acesso de admin
	// GLOBAL, inclusive a dado real de outras prefeituras. Admin continua só manual (login
	// por senha, criado direto no banco/por outro admin) — nunca auto-provisionado aqui.
	if papel == domain.PapelAdmin {
		ssoErro(w, http.StatusForbidden, "papel_sem_acesso", "admin nao pode ser concedido via login integrado", "papel=admin", emissor)
		return
	}

	if strings.TrimSpace(claims.Subject) == "" {
		ssoErro(w, http.StatusUnauthorized, "credencial_sem_identificador", "token sem identificador utilizavel", "token sem sub", emissor)
		return
	}
	email := strings.TrimSpace(claims.Email)
	if email == "" {
		email = strings.TrimSpace(claims.PreferredUsername)
	}
	if email == "" {
		ssoErro(w, http.StatusUnauthorized, "credencial_sem_identificador", "token sem identificador utilizavel", "token sem email nem preferred_username", emissor)
		return
	}

	// 7. Introspecção — OBRIGATÓRIA, só entra em jogo depois de tudo acima já ter passado.
	// Confirma uso único (o token torna-se inválido pra qualquer chamada seguinte).
	if err := auth.Introspect(r.Context(), emissor, c.ConectaIntrospectToken, corpo.Credencial); err != nil {
		if errors.Is(err, auth.ErrIntrospectInativo) {
			ssoErro(w, http.StatusUnauthorized, "identidade_invalida", "token ja utilizado ou expirado", err.Error(), emissor)
		} else {
			ssoErro(w, http.StatusServiceUnavailable, "identidade_indisponivel", "nao foi possivel confirmar a credencial agora", err.Error(), emissor)
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
		ssoErro(w, http.StatusInternalServerError, "erro_interno", "erro interno", err.Error(), emissor)
		return
	}
	if papel != domain.PapelAdmin && secretariaID == nil && unidadeID == nil {
		ssoErro(w, http.StatusForbidden, "vinculo_nao_encontrado", "secretaria/unidade do servidor nao mapeada", "department_ids/unit_ids nao resolveram nenhum vinculo", emissor)
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
			ssoErro(w, http.StatusForbidden, "credencial_sem_identificador", "e-mail ja vinculado a outra identidade", "vinculo externo conflitante", emissor)
			return
		}
		ssoErro(w, http.StatusInternalServerError, "erro_interno", "erro interno", err.Error(), emissor)
		return
	}
	if !usuario.Ativo {
		ssoErro(w, http.StatusForbidden, "usuario_desativado", "acesso revogado", "usuario inativo", emissor)
		return
	}

	token, err := auth.CriarSessao(c.SessaoSegredo, usuario.ID, c.SessaoDuracao)
	if err != nil {
		ssoErro(w, http.StatusInternalServerError, "erro_interno", "erro interno", "falha ao criar sessao: "+err.Error(), emissor)
		return
	}
	_ = c.Repo.AtualizarUltimoAcesso(r.Context(), usuario.ID)
	auth.DefinirCookieSessao(w, token, c.SessaoDuracao, c.CookieSeguro || auth.RequisicaoViaHTTPS(r))
	log.Printf("sso: login ok (emissor=%q, papel=%q, usuario=%q)", emissor, papel, usuario.ID)
	views.ResponderJSON(w, http.StatusOK, c.usuarioParaJSON(r.Context(), usuario))
}
