package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var ErrSessaoInvalida = errors.New("sessao invalida")

func HashSenha(senha string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(senha), bcrypt.DefaultCost)
	return string(b), err
}

func VerificarSenha(hash, senha string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(senha)) == nil
}

type SessaoClaims struct {
	UsuarioID string `json:"sub"`
	jwt.RegisteredClaims
}

// Sessão própria do painel — não confundir com o SSO real da plataforma (adiado pro
// pós-MVP, ver skill auth-conecta-cidades). Papel/situação do usuário NÃO ficam aqui:
// são relidos do banco a cada requisição (ver middleware), pra uma promoção ou
// desativação valer na hora, sem esperar o token expirar.
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
