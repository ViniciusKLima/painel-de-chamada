package controllers

import (
	"time"

	"painel-chamada-backend/internal/painel"
	"painel-chamada-backend/internal/repository"
)

type Controllers struct {
	Repo          *repository.Repo
	SessaoSegredo []byte
	SessaoDuracao time.Duration
	CookieSeguro  bool
	Painel        *painel.GerenciadorDeFilas
}
