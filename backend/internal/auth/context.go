package auth

import (
	"context"
	"net/http"
	"strings"
	"time"

	"painel-chamada-backend/internal/domain"
)

type ctxKey string

const chaveUsuario ctxKey = "usuarioAtual"
const NomeCookieSessao = "sessao"

func UsuarioDoContexto(ctx context.Context) *domain.Usuario {
	u, _ := ctx.Value(chaveUsuario).(*domain.Usuario)
	return u
}

type BuscadorDeUsuario interface {
	BuscarUsuarioPorID(ctx context.Context, id string) (*domain.Usuario, error)
}

// Middleware lê o cookie de sessão e repõe o usuário (papel/situação) do banco A CADA
// requisição — nunca confia em dado gravado no token além do id. Isso é o que faz uma
// promoção ou desativação valer na próxima requisição, sem esperar o token expirar. Ver
// skill auth-login-mvp / auth-conecta-cidades (mesmo desenho de sessão dos dois).
func Middleware(segredo []byte, repo BuscadorDeUsuario) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(NomeCookieSessao)
			if err != nil {
				responderNaoAutenticado(w, "identidade_ausente", "sessao ausente")
				return
			}
			claims, err := VerificarSessao(segredo, cookie.Value)
			if err != nil {
				responderNaoAutenticado(w, "sessao_invalida", "sessao invalida ou expirada")
				return
			}
			usuario, err := repo.BuscarUsuarioPorID(r.Context(), claims.UsuarioID)
			if err != nil {
				responderNaoAutenticado(w, "sessao_invalida", "sessao invalida")
				return
			}
			if !usuario.Ativo {
				responderNaoAutenticado(w, "usuario_desativado", "acesso revogado")
				return
			}
			ctx := context.WithValue(r.Context(), chaveUsuario, usuario)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func responderNaoAutenticado(w http.ResponseWriter, codigo, mensagem string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	w.Write([]byte(`{"error_code":"` + codigo + `","message":"` + mensagem + `"}`))
}

// sameSitePara (16/10): cookie "seguro" (HTTPS) usa SameSite=None, porque esse cenário só
// acontece quando front e back estão em domínios DIFERENTES (ex.: frontend no Vercel falando
// com um backend atrás de um túnel, fase de teste antes da hospedagem definitiva) — nesse
// caso o navegador trata como requisição CROSS-SITE de verdade, e SameSite=Lax não é enviado
// em chamadas fetch/XHR entre domínios (só em navegação de topo). Fora desse cenário (dev
// local, front e back no mesmo host), continua Lax — não precisa de None e Lax é mais
// restritivo/seguro por padrão.
func sameSitePara(seguro bool) http.SameSite {
	if seguro {
		return http.SameSiteNoneMode
	}
	return http.SameSiteLaxMode
}

func DefinirCookieSessao(w http.ResponseWriter, token string, duracao time.Duration, seguro bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     NomeCookieSessao,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   seguro,
		SameSite: sameSitePara(seguro),
		Expires:  time.Now().Add(duracao),
	})
}

func RemoverCookieSessao(w http.ResponseWriter, seguro bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     NomeCookieSessao,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   seguro,
		SameSite: sameSitePara(seguro),
		MaxAge:   -1,
	})
}

// RequisicaoViaHTTPS detecta se ESTA requisição específica chegou por HTTPS — direto (r.TLS)
// ou atrás de um proxy/túnel que termina TLS e repassa em texto puro pro nosso processo
// (cabeçalho `X-Forwarded-Proto`, convenção padrão desses proxies — inclui o cloudflared
// usado na fase de teste com túnel, 16/10). Combinado com a flag estática de ambiente
// (CookieSeguro) em vez de substituí-la: permite o MESMO processo atender ao mesmo tempo (a)
// acesso local direto, HTTP puro, sem túnel, e (b) acesso via túnel HTTPS — sem precisar
// escolher um modo fixo no boot nem rodar dois processos.
func RequisicaoViaHTTPS(r *http.Request) bool {
	if r.TLS != nil {
		return true
	}
	return strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
}
