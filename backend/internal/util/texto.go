package util

import "strings"

var substituicoesAcento = strings.NewReplacer(
	"á", "a", "à", "a", "ã", "a", "â", "a", "ä", "a",
	"é", "e", "è", "e", "ê", "e", "ë", "e",
	"í", "i", "ì", "i", "î", "i", "ï", "i",
	"ó", "o", "ò", "o", "õ", "o", "ô", "o", "ö", "o",
	"ú", "u", "ù", "u", "û", "u", "ü", "u",
	"ç", "c", "ñ", "n",
)

// NormalizarNome remove acento/caixa/espaço extra pra comparar nomes (unidades, etc) sem
// sensibilidade a diferença cosmética.
func NormalizarNome(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	s = substituicoesAcento.Replace(s)
	return strings.Join(strings.Fields(s), " ")
}

// NormalizarCabecalho faz o mesmo que NormalizarNome, mas também troca qualquer
// caractere que não seja letra/número/espaço por espaço — usado pra casar cabeçalho de
// planilha (que vem com parênteses, acentos etc) contra uma lista fixa de aliases.
func NormalizarCabecalho(s string) string {
	s = NormalizarNome(s)
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == ' ' {
			b.WriteRune(r)
		} else {
			b.WriteRune(' ')
		}
	}
	return strings.Join(strings.Fields(b.String()), " ")
}
