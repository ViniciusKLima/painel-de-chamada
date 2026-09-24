package util

import "strings"

// NomeParcial: primeiro nome + inicial do último sobrenome — usado no painel de TV
// público (decisão mantida em 19/09: nome parcial, não completo, mesmo com o protocolo
// já aparecendo junto).
func NomeParcial(nomeCompleto string) string {
	partes := strings.Fields(strings.TrimSpace(nomeCompleto))
	if len(partes) == 0 {
		return ""
	}
	if len(partes) == 1 {
		return partes[0]
	}
	primeiro := partes[0]
	ultimo := partes[len(partes)-1]
	inicial := strings.ToUpper(string([]rune(ultimo)[:1]))
	return primeiro + " " + inicial + "."
}
