package auth

import (
	"slices"

	"painel-chamada-backend/internal/domain"
)

// Permissões centralizadas (22/09, pedido explícito do dono do produto: "não quero
// permissões espalhadas em vários componentes através de condições difíceis de manter").
// Todo handler que precisa decidir "esse usuário pode mexer nisso?" chama uma função daqui
// em vez de comparar IDs/papel diretamente — a regra de quem pode o quê vive nesse arquivo
// só, não duplicada em cada handler.
//
// Essas funções são puras (não acessam banco) — quem chama já resolveu os IDs relevantes
// (ex. a secretaria dona da unidade/grade em questão) via repository antes de perguntar.

// EhAdmin: admin é sempre a checagem que vem primeiro em qualquer regra abaixo — acesso
// master à plataforma inteira, nunca limitado a prefeitura/secretaria/unidade específica.
func EhAdmin(usuario *domain.Usuario) bool {
	return usuario != nil && usuario.Papel == domain.PapelAdmin
}

// PodeAcessarSecretaria: admin sempre pode; gestor só a própria secretaria (a que ele foi
// alocado na criação, ver skill modelo-dados). Recepcionista/atendente nunca — eles operam
// por unidade/grade, não têm uma visão de secretaria inteira.
func PodeAcessarSecretaria(usuario *domain.Usuario, secretariaID string) bool {
	if EhAdmin(usuario) {
		return true
	}
	return usuario.Papel == domain.PapelGestor && usuario.SecretariaID != nil && *usuario.SecretariaID == secretariaID
}

// PodeAcessarUnidade: admin sempre pode. Gestor pode qualquer unidade que pertença à SUA
// secretaria (por isso recebe unidadeSecretariaID, resolvido pelo chamador via
// repo.BuscarUnidadePorID). Recepcionista só a própria unidade (comparação direta).
// Atendente não usa endpoints de unidade hoje (ele é escopado por grade), mas cai no mesmo
// caso de recepcionista se algum dia precisar.
func PodeAcessarUnidade(usuario *domain.Usuario, unidadeSecretariaID, unidadeID string) bool {
	if EhAdmin(usuario) {
		return true
	}
	if usuario.Papel == domain.PapelGestor {
		return usuario.SecretariaID != nil && *usuario.SecretariaID == unidadeSecretariaID
	}
	return usuario.UnidadeID != nil && *usuario.UnidadeID == unidadeID
}

// PodeAcessarGrade: admin sempre pode. Gestor pode qualquer grade de qualquer unidade da
// sua secretaria. Atendente só uma grade em que está alocado (23/09: agora pode estar
// alocado em mais de uma — `gradesDoUsuario` é a lista resolvida pelo chamador via
// repo.ListarGradesDoUsuario, esta função continua pura/sem acesso a banco). Recepcionista
// não usa endpoints de grade hoje (trabalha a unidade inteira via PodeAcessarUnidade).
func PodeAcessarGrade(usuario *domain.Usuario, gradeUnidadeSecretariaID, gradeUnidadeID, gradeID string, gradesDoUsuario []string) bool {
	if EhAdmin(usuario) {
		return true
	}
	if usuario.Papel == domain.PapelGestor {
		return usuario.SecretariaID != nil && *usuario.SecretariaID == gradeUnidadeSecretariaID
	}
	if usuario.Papel == domain.PapelAtendente {
		return slices.Contains(gradesDoUsuario, gradeID)
	}
	return usuario.UnidadeID != nil && *usuario.UnidadeID == gradeUnidadeID
}

// TemPapel: helper pra checagem "só papel X ou Y pode" sem repetir `usuario.Papel != a &&
// usuario.Papel != b` em cada handler — admin NÃO é incluído automaticamente aqui, porque
// nem toda rota de papel específico deveria abrir pro admin sem pensar (ex. "chamar
// próximo" é uma ação operacional de atendente, o admin normalmente só configura, não
// atende fila). Passe domain.PapelAdmin explicitamente na lista quando fizer sentido.
func TemPapel(usuario *domain.Usuario, papeis ...domain.Papel) bool {
	if usuario == nil {
		return false
	}
	for _, p := range papeis {
		if usuario.Papel == p {
			return true
		}
	}
	return false
}
