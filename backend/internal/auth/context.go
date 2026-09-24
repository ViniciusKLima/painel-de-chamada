package auth

import (
	"context"
	"net/http"
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

func DefinirCookieSessao(w http.ResponseWriter, token string, duracao time.Duration, seguro bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     NomeCookieSessao,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   seguro,
		SameSite: http.SameSiteLaxMode,
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
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}
