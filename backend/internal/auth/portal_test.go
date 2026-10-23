package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Testes de verdade contra uma chave RSA gerada na hora + um servidor JWKS local — não dá
// pra testar contra a chave privada real da Conecta Cidades (não temos acesso a ela, só a
// pública), mas o formato (JWKS em /.well-known/jwks.json, RS256, claims do token) é
// exatamente o documentado oficialmente (ver skill auth-conecta-cidades). Isso exercita a
// lógica de verificação de ponta a ponta, não é mock do nosso próprio sistema.

func servidorJWKSDeTeste(t *testing.T, chave *rsa.PublicKey, kid string) *httptest.Server {
	t.Helper()
	jwks := conjuntoChaves{Keys: []jwk{{
		Kty: "RSA",
		Use: "sig",
		Kid: kid,
		N:   base64.RawURLEncoding.EncodeToString(chave.N.Bytes()),
		E:   base64.RawURLEncoding.EncodeToString(big2Bytes(chave.E)),
	}}}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(jwks)
	}))
}

func big2Bytes(e int) []byte {
	// Exponente público padrão (65537 = 0x010001) cabe em 3 bytes — suficiente pros testes,
	// mesma faixa que toda chave RSA real de assinatura usa.
	return []byte{byte(e >> 16), byte(e >> 8), byte(e)}
}

func assinarTokenDeTeste(t *testing.T, chavePrivada *rsa.PrivateKey, kid string, claims ClaimsPortal) string {
	t.Helper()
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = kid
	assinado, err := token.SignedString(chavePrivada)
	if err != nil {
		t.Fatalf("falha ao assinar token de teste: %v", err)
	}
	return assinado
}

func TestVerificarCredencialPortal_TokenValidoPassa(t *testing.T) {
	chavePrivada, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("gerar chave: %v", err)
	}
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "chave-teste-1")
	defer servidor.Close()

	claims := ClaimsPortal{
		PreferredUsername: "servidor@prefeitura.teste",
		Email:             "servidor@prefeitura.teste",
		Name:              "Servidor Teste",
		Roles:             []string{"attendant"},
		DepartmentIDs:     []int{3},
		UnitIDs:           []int{12},
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    servidor.URL,
			Subject:   "42",
			Audience:  jwt.ClaimStrings{"painel.teste.com"},
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	credencial := assinarTokenDeTeste(t, chavePrivada, "chave-teste-1", claims)

	chaves := NovoChavesPublicas()
	resultado, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if err != nil {
		t.Fatalf("esperava token valido, deu erro: %v", err)
	}
	if resultado.Subject != "42" || resultado.Email != "servidor@prefeitura.teste" {
		t.Errorf("claims nao bateram: %+v", resultado)
	}
}

func TestVerificarCredencialPortal_AudienciaVaziaFalhaFechado(t *testing.T) {
	// Achado por revisão de segurança (22/10): sem audiência configurada, a verificação
	// ANTES simplesmente pulava a checagem de `aud` (falha ABERTA) — um token válido emitido
	// pra outra aplicação integrada ao mesmo emissor seria aceito aqui também. Agora precisa
	// falhar FECHADO: nenhum token passa sem uma audiência esperada configurada.
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    servidor.URL,
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"qualquer-coisa.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	credencial := assinarTokenDeTeste(t, chavePrivada, "k1", claims)

	chaves := NovoChavesPublicas()
	_, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "", credencial)
	if !errors.Is(err, ErrAudienciaNaoConfigurada) {
		t.Errorf("esperava ErrAudienciaNaoConfigurada com audiencia vazia, veio: %v", err)
	}
}

func TestVerificarCredencialPortal_AudienciaErradaRecusa(t *testing.T) {
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    servidor.URL,
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"outro-dominio.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	credencial := assinarTokenDeTeste(t, chavePrivada, "k1", claims)

	chaves := NovoChavesPublicas()
	_, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Errorf("esperava ErrCredencialInvalida por aud errado, veio: %v", err)
	}
}

func TestVerificarCredencialPortal_TokenExpiradoRecusa(t *testing.T) {
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:   servidor.URL,
			Subject:  "1",
			Audience: jwt.ClaimStrings{"painel.teste.com"},
			// Expirou há 5 minutos — bem fora da tolerância de 60s.
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-5 * time.Minute)),
		},
	}
	credencial := assinarTokenDeTeste(t, chavePrivada, "k1", claims)

	chaves := NovoChavesPublicas()
	_, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Errorf("esperava ErrCredencialInvalida por token expirado, veio: %v", err)
	}
}

func TestVerificarCredencialPortal_IssuerErradoRecusa(t *testing.T) {
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://emissor-impostor.exemplo.com", // não bate com `issuer` passado abaixo
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"painel.teste.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	credencial := assinarTokenDeTeste(t, chavePrivada, "k1", claims)

	chaves := NovoChavesPublicas()
	// Note: issuer passado é o do servidor de teste (de onde vem o JWKS de verdade), mas o
	// `iss` DENTRO do token é outro — simula alguém tentando usar uma chave de um emissor
	// pra validar um token que alega ser de outro.
	_, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Errorf("esperava ErrCredencialInvalida por iss nao bater, veio: %v", err)
	}
}

func TestVerificarCredencialPortal_AssinaturaAdulteradaRecusa(t *testing.T) {
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	outraChave, _ := rsa.GenerateKey(rand.Reader, 2048) // chave DIFERENTE da publicada no JWKS
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    servidor.URL,
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"painel.teste.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	// Assinado com uma chave que NÃO é a publicada no JWKS — simula um token forjado.
	credencial := assinarTokenDeTeste(t, outraChave, "k1", claims)

	chaves := NovoChavesPublicas()
	_, err := VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if !errors.Is(err, ErrCredencialInvalida) {
		t.Errorf("esperava ErrCredencialInvalida por assinatura forjada, veio: %v", err)
	}
}

func TestVerificarCredencialPortal_AlgoritmoNaoAssimetricoRecusa(t *testing.T) {
	// "none" e HS* nunca são aceitos, mesmo que o header do token peça — lista fechada de
	// algoritmos (skill auth-conecta-cidades, seção 3 e 14).
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	servidor := servidorJWKSDeTeste(t, &chavePrivada.PublicKey, "k1")
	defer servidor.Close()

	claims := ClaimsPortal{
		Email: "x@y.com",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    servidor.URL,
			Subject:   "1",
			Audience:  jwt.ClaimStrings{"painel.teste.com"},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	credencial, err := token.SignedString([]byte("segredo-qualquer"))
	if err != nil {
		t.Fatalf("assinar HS256: %v", err)
	}

	chaves := NovoChavesPublicas()
	_, err = VerificarCredencialPortal(context.Background(), chaves, servidor.URL, "painel.teste.com", credencial)
	if err == nil {
		t.Fatal("esperava recusa de token HS256, mas passou")
	}
}

func TestLerEmissorSemVerificar(t *testing.T) {
	chavePrivada, _ := rsa.GenerateKey(rand.Reader, 2048)
	claims := ClaimsPortal{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "https://Jaboataomaisfacil.Jaboatao.PE.gov.br/Realms/Teste/",
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(30 * time.Second)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	credencial, _ := token.SignedString(chavePrivada)

	emissor, err := LerEmissorSemVerificar(credencial)
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	esperado := "https://jaboataomaisfacil.jaboatao.pe.gov.br/Realms/Teste"
	if emissor != esperado {
		t.Errorf("emissor normalizado = %q, esperado %q", emissor, esperado)
	}
}

func TestNormalizarEmissor(t *testing.T) {
	casos := []struct {
		entrada  string
		esperado string
	}{
		{"https://exemplo.com", "https://exemplo.com"},
		{"https://Exemplo.COM/", "https://exemplo.com"},
		{"https://exemplo.com/realms/x/", "https://exemplo.com/realms/x"},
		{"https://exemplo.com?token=abc", "https://exemplo.com"},
		{"ftp://exemplo.com", ""},
		{"nao-e-url-nenhuma", ""},
		{"", ""},
	}
	for _, c := range casos {
		got := NormalizarEmissor(c.entrada)
		if got != c.esperado {
			t.Errorf("NormalizarEmissor(%q) = %q, esperado %q", c.entrada, got, c.esperado)
		}
	}
}

func TestMapearPapel(t *testing.T) {
	casos := []struct {
		roles       []string
		papel       string
		reconhecido bool
	}{
		{[]string{"attendant"}, "atendente", true},
		{[]string{"receptionist"}, "recepcionista", true},
		{[]string{"manager"}, "gestor", true},
		{[]string{"contract_admin"}, "gestor", true},
		{[]string{"admin"}, "admin", true},
		{[]string{"super_admin"}, "admin", true},
		{[]string{"ADMIN"}, "admin", true}, // case-insensitive
		{[]string{"algum_papel_desconhecido"}, "", false},
		{[]string{}, "", false},
		// Mais de um reconhecido: admin vence (prioridade mais alta na lista).
		{[]string{"attendant", "admin"}, "admin", true},
	}
	for _, c := range casos {
		papel, ok := MapearPapel(c.roles)
		if ok != c.reconhecido || string(papel) != c.papel {
			t.Errorf("MapearPapel(%v) = (%q, %v), esperado (%q, %v)", c.roles, papel, ok, c.papel, c.reconhecido)
		}
	}
}

func TestIntrospect(t *testing.T) {
	servidor := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer token-valido" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		var corpo map[string]string
		json.NewDecoder(r.Body).Decode(&corpo)
		w.Header().Set("Content-Type", "application/json")
		if corpo["token"] == "token-ja-usado" {
			json.NewEncoder(w).Encode(map[string]any{"active": false})
			return
		}
		json.NewEncoder(w).Encode(map[string]any{"active": true})
	}))
	defer servidor.Close()

	if err := Introspect(context.Background(), servidor.URL, "token-valido", "qualquer-jwt"); err != nil {
		t.Errorf("esperava sucesso (active:true), deu: %v", err)
	}

	err := Introspect(context.Background(), servidor.URL, "token-valido", "token-ja-usado")
	if !errors.Is(err, ErrIntrospectInativo) {
		t.Errorf("esperava ErrIntrospectInativo, veio: %v", err)
	}

	err = Introspect(context.Background(), servidor.URL, "token-de-api-errado", "qualquer-jwt")
	if !errors.Is(err, ErrJWKSIndisponivel) {
		t.Errorf("esperava ErrJWKSIndisponivel (401 da API), veio: %v", err)
	}
}
