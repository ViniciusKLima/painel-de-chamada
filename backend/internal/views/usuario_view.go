package views

import (
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
)

// GradeResumoJSON — forma padrão (id/nome/unidadeId/unidadeNome) de expor uma grade alocada
// em qualquer lugar que devolve `grades: [...]` (sessão, "Usuários internos", "Usuários" da
// prefeitura). `unidadeId`/`unidadeNome` (23/09, auditoria de multi-grade) são o que permite
// o frontend agrupar por LOCAL sem precisar de outra requisição — importante porque um
// atendente pode ter grades de unidades DIFERENTES (o `usuario.unidadeId` sozinho não conta
// essa história).
func GradeResumoJSON(g repository.GradeComUnidade) map[string]any {
	return map[string]any{"id": g.ID, "nome": g.Nome, "unidadeId": g.UnidadeID, "unidadeNome": g.UnidadeNome}
}

func GradesResumoJSON(grades []repository.GradeComUnidade) []map[string]any {
	out := make([]map[string]any, len(grades))
	for i, g := range grades {
		out[i] = GradeResumoJSON(g)
	}
	return out
}

// UsuarioInternoParaJSON monta a linha da tela "Usuários internos" — nome, cargo (papel),
// grades em que está alocado (a "etiqueta" de cada uma, 23/09 — antes era uma grade só) e o
// status "pendente" (nunca logou) vs. "último acesso em X" (22/09).
func UsuarioInternoParaJSON(u domain.Usuario, grades []repository.GradeComUnidade) map[string]any {
	gradesJSON := GradesResumoJSON(grades)
	return map[string]any{
		"id":             u.ID,
		"nome":           u.Nome,
		"email":          u.Email,
		"papel":          u.Papel,
		"ativo":          u.Ativo,
		"grades":         gradesJSON,
		"ultimoAcessoEm": u.UltimoAcessoEm,
		"pendente":       u.UltimoAcessoEm == nil,
	}
}

// UsuarioComContextoParaJSON é a linha da tela "Usuários internos" consolidada do admin —
// mesma ideia de UsuarioInternoParaJSON, com prefeitura/secretaria/unidade já resolvidos via
// join (sem N+1 no frontend).
func UsuarioComContextoParaJSON(u repository.UsuarioComContexto) map[string]any {
	grades := GradesResumoJSON(u.Grades)
	return map[string]any{
		"id":             u.ID,
		"nome":           u.Nome,
		"email":          u.Email,
		"papel":          u.Papel,
		"ativo":          u.Ativo,
		"ultimoAcessoEm": u.UltimoAcessoEm,
		"pendente":       u.UltimoAcessoEm == nil,
		"prefeituraNome": u.PrefeituraNome,
		"secretariaId":   u.SecretariaID,
		"secretariaNome": u.SecretariaNome,
		"unidadeId":      u.UnidadeID,
		"unidadeNome":    u.UnidadeNome,
		"grades":         grades,
	}
}

// UsuarioParaJSON monta a resposta de identidade (`POST /login`, `GET /sessao`) a partir de
// dado JÁ RESOLVIDO pelo controller (que é quem faz as consultas em cascata — unidade →
// secretaria → prefeitura, mais as grades alocadas). Esta função em si é pura: só decide o
// formato, nunca toca o banco — ver `Controllers.usuarioParaJSON` em
// `internal/controllers/auth.go` pra a parte que resolve os dados.
func UsuarioParaJSON(u *domain.Usuario, unidadeID, unidadeNome, secretariaID any, grades []repository.GradeComUnidade, corDestaque, corClara, prefeituraNome any) map[string]any {
	return map[string]any{
		"id":             u.ID,
		"nome":           u.Nome,
		"email":          u.Email,
		"papel":          u.Papel,
		"unidadeId":      unidadeID,
		"unidadeNome":    unidadeNome,
		"secretariaId":   secretariaID,
		"grades":         GradesResumoJSON(grades),
		"corDestaque":    corDestaque,
		"corClara":       corClara,
		"prefeituraNome": prefeituraNome,
	}
}
