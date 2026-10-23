package auth

import (
	"context"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"sync"
	"time"
)

// ErrJWKSIndisponivel cobre qualquer falha pra obter as chaves do emissor (rede, timeout,
// resposta ilegível, kid desconhecido mesmo após rebusca) — o chamador trata isso como 503
// `identidade_indisponivel` (ver skill auth-conecta-cidades, seção 7): "a prefeitura está
// fora do ar", nunca um modo degradado que aceita sem verificar.
var ErrJWKSIndisponivel = errors.New("jwks indisponivel")

// ErrChaveDesconhecida: o JWKS foi buscado com sucesso (fresco, não veio do cache fora da
// janela de rebusca) e o `kid` do token simplesmente não está lá — diferente de
// ErrJWKSIndisponivel ("não consegui nem checar"), aqui a checagem ACONTECEU e o token
// referencia uma chave que genuinamente não existe (token forjado, ou chave já rotacionada
// há muito tempo). O chamador trata isso como credencial inválida (401), não indisponível
// (503) — achado real (22/10): testar com um `kid` forjado contra o JWKS de verdade da
// Smartândia originalmente caía como 503, o que classificava errado um token claramente
// inválido como "provedor fora do ar".
var ErrChaveDesconhecida = errors.New("chave do token nao encontrada no jwks")

type jwk struct {
	Kty string `json:"kty"`
	Use string `json:"use"`
	Kid string `json:"kid"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type conjuntoChaves struct {
	Keys []jwk `json:"keys"`
}

type entradaCache struct {
	chaves    map[string]*rsa.PublicKey
	buscadoEm time.Time
}

// ChavesPublicas é o cache de JWKS por URL (um emissor por prefeitura, ver skill
// auth-conecta-cidades seção 3) — processo inteiro, protegido por mutex. Nunca busca o JWKS
// de uma URL arbitrária vinda do token: a URL é sempre derivada do `iss` já resolvido contra
// `prefeituras.issuer_jwt` no banco (CacheJWKS.Chave), então não existe o risco (que o
// desenho genérico de referência se preocupa) de alguém injetar uma URL de chaves de
// terceiros — aqui isso nunca é configurável por fora.
type ChavesPublicas struct {
	mu      sync.Mutex
	cache   map[string]entradaCache
	cliente *http.Client
}

func NovoChavesPublicas() *ChavesPublicas {
	return &ChavesPublicas{
		cache:   map[string]entradaCache{},
		cliente: &http.Client{Timeout: 3 * time.Second},
	}
}

// intervaloMinimoRebusca: um `kid` desconhecido dispara no máximo uma rebusca a cada 5min
// por URL (skill auth-conecta-cidades, seção 4, passo 4) — protege contra um emissor fora do
// ar virar fonte de requisições repetidas a cada login tentado.
const intervaloMinimoRebusca = 5 * time.Minute

// limiteBytesResposta: nunca ler mais que 1MB da resposta do JWKS (mesma seção) — um JWKS de
// verdade tem poucos KB; isso é só proteção contra uma resposta anormalmente grande.
const limiteBytesResposta = 1 << 20

// Chave devolve a chave pública RSA pro `kid` pedido, buscando (ou reaproveitando do cache)
// o JWKS em `jwksURL`. `kid` desconhecido no cache dispara uma rebusca (no máximo uma a cada
// intervaloMinimoRebusca) antes de desistir.
// resolverChave acha a chave certa dado o `kid` do token — ou, quando o token NÃO tem `kid`
// no cabeçalho (achado real, 22/10: os tokens de "Login Integrado (SSO) — Servidor" da
// Conecta Cidades vêm só com `{"alg":"RS256"}`, sem `kid` nenhum), usa a ÚNICA chave do JWKS
// se houver exatamente uma — ambíguo (e recusado) só quando há mais de uma chave disponível
// e nenhum `kid` pra escolher entre elas.
func resolverChave(chaves map[string]*rsa.PublicKey, kid string) (*rsa.PublicKey, bool) {
	if chave, ok := chaves[kid]; ok {
		return chave, true
	}
	if kid == "" && len(chaves) == 1 {
		for _, unica := range chaves {
			return unica, true
		}
	}
	return nil, false
}

func (c *ChavesPublicas) Chave(ctx context.Context, jwksURL, kid string) (*rsa.PublicKey, error) {
	c.mu.Lock()
	entrada, existe := c.cache[jwksURL]
	if existe {
		if chave, ok := resolverChave(entrada.chaves, kid); ok {
			c.mu.Unlock()
			return chave, nil
		}
		if time.Since(entrada.buscadoEm) < intervaloMinimoRebusca {
			c.mu.Unlock()
			return nil, fmt.Errorf("%w: kid %q nao encontrado (cache recente)", ErrJWKSIndisponivel, kid)
		}
	}
	c.mu.Unlock()

	chaves, err := buscarJWKS(ctx, c.cliente, jwksURL)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	c.cache[jwksURL] = entradaCache{chaves: chaves, buscadoEm: time.Now()}
	c.mu.Unlock()

	chave, ok := resolverChave(chaves, kid)
	if !ok {
		return nil, fmt.Errorf("%w: kid %q nao encontrado apos busca fresca", ErrChaveDesconhecida, kid)
	}
	return chave, nil
}

func buscarJWKS(ctx context.Context, cliente *http.Client, jwksURL string) (map[string]*rsa.PublicKey, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, jwksURL, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJWKSIndisponivel, err)
	}
	resp, err := cliente.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJWKSIndisponivel, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: status %d", ErrJWKSIndisponivel, resp.StatusCode)
	}

	corpo, err := io.ReadAll(io.LimitReader(resp.Body, limiteBytesResposta))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrJWKSIndisponivel, err)
	}

	var conjunto conjuntoChaves
	if err := json.Unmarshal(corpo, &conjunto); err != nil {
		return nil, fmt.Errorf("%w: resposta ilegivel: %v", ErrJWKSIndisponivel, err)
	}

	chaves := map[string]*rsa.PublicKey{}
	for _, k := range conjunto.Keys {
		// Só chaves RSA de assinatura (kty=RSA, use vazio ou "sig") — skill seção 4, passo 4.
		if k.Kty != "RSA" || (k.Use != "" && k.Use != "sig") {
			continue
		}
		chave, err := montarChaveRSA(k.N, k.E)
		if err != nil {
			continue
		}
		chaves[k.Kid] = chave
	}
	if len(chaves) == 0 {
		return nil, fmt.Errorf("%w: nenhuma chave RSA de assinatura no JWKS", ErrJWKSIndisponivel)
	}
	return chaves, nil
}

func montarChaveRSA(nBase64, eBase64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nBase64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eBase64)
	if err != nil {
		return nil, err
	}
	n := new(big.Int).SetBytes(nBytes)
	e := new(big.Int).SetBytes(eBytes)
	return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
}
