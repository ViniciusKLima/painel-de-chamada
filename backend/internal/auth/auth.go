package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrSessaoInvalida = errors.New("sessao invalida")

// HashSenha/VerificarSenha REMOVIDAS (22/10, junto com o login por senha — ver
// controllers/auth.go) — a única forma de autenticar agora é o SSO via portal. A dependência
// `golang.org/x/crypto/bcrypt`, que só existia pra isso, também saiu do go.mod.

type SessaoClaims struct {
	UsuarioID string `json:"sub"`
	jwt.RegisteredClaims
}

// Sessão própria do painel — criada tanto pelo login via portal (SSO, skill
// auth-conecta-cidades) quanto, no histórico deste arquivo, pelo antigo login por senha
// (removido 22/10). Papel/situação do usuário NÃO ficam aqui: são relidos do banco a cada
// requisição (ver middleware), pra uma promoção ou desativação valer na hora, sem esperar o
// token expirar.
func CriarSessao(segredo []byte, usuarioID string, duracao time.Duration) (string, error) {
	claims := SessaoClaims{
		UsuarioID: usuarioID,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "painel-chamada",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(duracao)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(segredo)
}

func VerificarSessao(segredo []byte, tokenStr string) (*SessaoClaims, error) {
	claims := &SessaoClaims{}
	token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrSessaoInvalida
		}
		return segredo, nil
	}, jwt.WithValidMethods([]string{"HS256"}), jwt.WithIssuer("painel-chamada"))
	if err != nil || !token.Valid {
		return nil, ErrSessaoInvalida
	}
	return claims, nil
}
