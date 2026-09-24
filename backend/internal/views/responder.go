// Package views é a camada de apresentação (o "V" de MVC) — decide como uma entidade do
// Model (internal/domain + internal/repository) vira a resposta JSON que o frontend recebe.
// Nunca acessa o banco nem contém regra de negócio: toda função aqui é pura, recebendo dado
// já resolvido pelo controller e devolvendo `map[string]any`/`[]map[string]any` prontos pra
// `ResponderJSON`. Extraído em 24/09 (refactor pra MVC) a partir de funções que viviam
// espalhadas dentro de `internal/handlers` (agora `internal/controllers`).
package views

import (
	"encoding/json"
	"net/http"
)

// ResponderJSON escreve `v` como JSON no corpo da resposta — o único ponto do sistema que
// efetivamente "renderiza" uma resposta HTTP.
func ResponderJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

// ResponderErro é o formato padrão de erro da API: `{"error_code": "...", "message": "..."}`.
func ResponderErro(w http.ResponseWriter, status int, codigo, mensagem string) {
	ResponderJSON(w, status, map[string]string{"error_code": codigo, "message": mensagem})
}
