package views

import "painel-chamada-backend/internal/domain"

func PrefeituraParaJSON(p *domain.Prefeitura) map[string]any {
	return map[string]any{
		"id":          p.ID,
		"nome":        p.Nome,
		"slug":        p.Slug,
		"corDestaque": p.CorDestaque,
		"corClara":    p.CorClara,
	}
}

func SecretariaParaJSON(s *domain.Secretaria) map[string]any {
	return map[string]any{
		"id":           s.ID,
		"prefeituraId": s.PrefeituraID,
		"nome":         s.Nome,
		"sigla":        s.Sigla,
	}
}
