package controllers

import (
	"time"

	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/painel"
	"painel-chamada-backend/internal/repository"
)

type Controllers struct {
	Repo          *repository.Repo
	SessaoSegredo []byte
	SessaoDuracao time.Duration
	CookieSeguro  bool
	Painel        *painel.GerenciadorDeFilas

	// Login integrado via portal (SSO real da Conecta Cidades, 22/10) — ver skill
	// auth-conecta-cidades. ChavesPublicas é o cache de JWKS (processo inteiro, um por
	// backend); ConectaAudiencia é o `aud` esperado nos tokens (CONECTA_AUDIENCIA);
	// ConectaIntrospectToken é a credencial de API com permissão `introspect_api` — vazio
	// desliga o endpoint de portal por completo (ver PostSessaoPortal), nunca aceita login
	// sem conseguir confirmar o uso único.
	ChavesPublicas         *auth.ChavesPublicas
	ConectaAudiencia       string
	ConectaIntrospectToken string
}
