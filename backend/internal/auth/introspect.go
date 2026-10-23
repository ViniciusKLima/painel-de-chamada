package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

// ErrIntrospectInativo: a introspecção respondeu, mas `active: false` — token expirado,
// inválido ou (o caso que mais importa aqui) já usado antes. Distinto de ErrJWKSIndisponivel
// (que é "não consegui nem perguntar") — este é "perguntei, e a resposta foi não".
var ErrIntrospectInativo = errors.New("token inativo na introspeccao")

type respostaIntrospect struct {
	Active bool `json:"active"`
}

// Introspect chama `POST {issuer}/oauth/introspect` (uso OBRIGATÓRIO segundo a doc oficial —
// ver skill auth-conecta-cidades, "Confirmação por documentação OFICIAL real" e "Confirmação
// final: doc oficial do PRÓPRIO Jaboatão") com o token de API `apiToken` (permissão
// `introspect_api`, específica do município — guardado em CONECTA_INTROSPECT_API_TOKEN,
// nunca versionado). É o único jeito de saber se ESTE token específico já foi consumido
// antes (uso único) — verificação de assinatura local não basta pra isso.
func Introspect(ctx context.Context, issuer, apiToken, credencial string) error {
	corpo, err := json.Marshal(map[string]string{"token": credencial})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, issuer+"/oauth/introspect", bytes.NewReader(corpo))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiToken)

	cliente := &http.Client{Timeout: 5 * time.Second}
	resp, err := cliente.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrJWKSIndisponivel, err)
	}
	defer resp.Body.Close()

	// 429 (rate limit) e qualquer erro de infraestrutura do lado da Conecta Cidades contam
	// como "identidade indisponível" (503 pro nosso cliente) — não é "token inválido", é "não
	// consegui confirmar agora". 401 (token de API errado/sem permissão introspect_api) é erro
	// de CONFIGURAÇÃO nossa, não do usuário — também vira identidade_indisponivel, pra não
	// confundir com uma credencial de usuário ruim.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("%w: introspect respondeu status %d", ErrJWKSIndisponivel, resp.StatusCode)
	}

	var r respostaIntrospect
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		return fmt.Errorf("%w: resposta ilegivel: %v", ErrJWKSIndisponivel, err)
	}
	if !r.Active {
		return ErrIntrospectInativo
	}
	return nil
}
