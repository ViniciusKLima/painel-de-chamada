package auth

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"painel-chamada-backend/internal/domain"
)

// Login integrado via portal (SSO real da Conecta Cidades) — ver skill auth-conecta-cidades
// pro desenho completo e as fontes oficiais (doc do próprio Jaboatão, 22/10). Este arquivo
// cobre só a parte criptográfica/de claims (sem tocar banco) — a orquestração completa
// (resolver prefeitura, resolver secretaria/unidade, upsert de usuário) fica no controller
// (internal/controllers/sso.go), que é quem tem acesso ao repository.

var (
	ErrCredencialInvalida = errors.New("credencial invalida")
	ErrEmissorInvalido    = errors.New("emissor invalido")
)

// ClaimsPortal — formato confirmado pela doc oficial (seção "Login Integrado (SSO) —
// Servidor"): sub, iss, aud, iat, exp, jti, preferred_username, email, email_verified, name,
// roles[], department_ids[], unit_ids[], cpf (opcional, "normalmente ausente" segundo a
// própria doc — nunca assumir presença).
type ClaimsPortal struct {
	PreferredUsername string   `json:"preferred_username"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	Name              string   `json:"name"`
	Roles             []string `json:"roles"`
	DepartmentIDs     []int    `json:"department_ids"`
	UnitIDs           []int    `json:"unit_ids"`
	CPF               string   `json:"cpf"`
	jwt.RegisteredClaims
}

// NormalizarEmissor devolve o `iss` na forma usada nas comparações (esquema e host em
// minúsculas, sem barra final, sem query/fragmento; caminho preservado como veio — um realm
// pode morar no caminho), ou "" se inválido. Usada tanto no cadastro de uma prefeitura
// (campo issuer_jwt) quanto na entrada — os dois precisam concordar byte a byte.
func NormalizarEmissor(valor string) string {
	u, err := url.Parse(strings.TrimSpace(valor))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host) + strings.TrimRight(u.EscapedPath(), "/")
}

// LerEmissorSemVerificar decodifica o payload SEM checar assinatura, só pra extrair o `iss`
// normalizado — usado unicamente pra decidir contra qual prefeitura (e, por extensão, qual
// JWKS) checar a assinatura de verdade em seguida. Nada lido aqui é confiável ainda: um
// emissor mentiroso só erra a chave na verificação de verdade, nunca engana esta função (skill
// auth-conecta-cidades, seção 4, passo 1).
func LerEmissorSemVerificar(credencial string) (string, error) {
	var claims ClaimsPortal
	if _, _, err := jwt.NewParser().ParseUnverified(credencial, &claims); err != nil {
		return "", fmt.Errorf("%w: %v", ErrCredencialInvalida, err)
	}
	emissor := NormalizarEmissor(claims.Issuer)
	if emissor == "" {
		return "", fmt.Errorf("%w: iss ausente ou invalido", ErrEmissorInvalido)
	}
	return emissor, nil
}

// ErrAudienciaNaoConfigurada: chamar VerificarCredencialPortal sem uma audiência esperada é
// um erro de QUEM CHAMA, não do token — a verificação de `aud` nunca é pulada (achado por
// revisão de segurança, 22/10: antes, `audienciaEsperada` vazio simplesmente deixava de
// aplicar `jwt.WithAudience`, ou seja, falhava ABERTO — qualquer token válido pra QUALQUER
// outra aplicação integrada ao mesmo emissor seria aceito aqui também, um ataque clássico de
// "confusão de audiência"). Falha FECHADA: sem audiência configurada, nenhum token passa.
var ErrAudienciaNaoConfigurada = errors.New("audiencia esperada nao configurada")

// VerificarCredencialPortal confere a assinatura do token contra o JWKS do emissor JÁ
// RESOLVIDO no banco (`issuer` vem de uma `prefeituras.issuer_jwt` cadastrada — nunca direto
// do token, então a URL do JWKS, sempre `{issuer}/.well-known/jwks.json`, nunca é
// influenciável por fora). Lista fechada de algoritmos (RS256/384/512 — nunca "none" nem
// HS*, nunca o que o token pedir), `iss` exato, `exp` obrigatório com 60s de tolerância, e
// `aud` contra a audiência configurada desta instalação (CONECTA_AUDIENCIA — a doc oficial é
// explícita: "deve ser exatamente o seu domínio, sem http/https, páginas ou parâmetros"),
// SEMPRE aplicada, nunca pulada. Só depois da assinatura conferir as claims ficam confiáveis
// (skill seção 4, passos 3-6).
func VerificarCredencialPortal(ctx context.Context, chaves *ChavesPublicas, issuer, audienciaEsperada, credencial string) (*ClaimsPortal, error) {
	if audienciaEsperada == "" {
		return nil, ErrAudienciaNaoConfigurada
	}
	jwksURL := issuer + "/.well-known/jwks.json"
	var claims ClaimsPortal
	opcoes := []jwt.ParserOption{
		jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}),
		jwt.WithIssuer(issuer),
		jwt.WithLeeway(60 * time.Second),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(audienciaEsperada),
	}
	_, err := jwt.ParseWithClaims(credencial, &claims, func(t *jwt.Token) (any, error) {
		kid, _ := t.Header["kid"].(string)
		return chaves.Chave(ctx, jwksURL, kid)
	}, opcoes...)
	if err != nil {
		if errors.Is(err, ErrJWKSIndisponivel) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", ErrCredencialInvalida, err)
	}
	return &claims, nil
}

// MapearPapel — mapeamento `roles` (doc) -> papel nosso (skill seção 6). `attendant` e
// `receptionist` têm confiança ALTA (nome bate literalmente). `admin`/`super_admin` são o
// nível MAIS ALTO da própria Conecta Cidades — "o maior abaixo só dos devs da LegalTech"
// (confirmado pelo dono do produto, 22/10, ele mesmo é quem tem esse nível) — NUNCA é
// concedido via SSO aqui (ver bloqueio explícito em `PostSessaoPortal`): é um nível global da
// plataforma inteira, não de uma prefeitura específica, então nunca deveria virar o nosso
// `admin` (que também é global, mas da NOSSA plataforma). `contract_admin` é o admin
// tenant-scoped — "o admin da prefeitura" — confirmado pelo dono do produto (22/10) como o
// papel de quem administra UMA prefeitura dentro da Conecta Cidades; mapeia pro nosso
// `gestor` (nosso papel mais alto que ainda é escopado a uma secretaria/prefeitura, nunca
// global). `manager` continua mapeado pra `gestor` também (confiança média, mesmo papel
// "gerencial" genérico de antes — pode ser um nível abaixo de `contract_admin` dentro da
// plataforma deles, mas nosso domínio não distingue isso, só tem um nível "gestor"). Roles em
// ordem de prioridade — se o token trouxer mais de um papel reconhecido, o mais alto na lista
// vence.
func MapearPapel(roles []string) (domain.Papel, bool) {
	prioridade := []struct {
		externo string
		interno domain.Papel
	}{
		{"admin", domain.PapelAdmin},
		{"super_admin", domain.PapelAdmin},
		{"contract_admin", domain.PapelGestor},
		{"manager", domain.PapelGestor},
		{"attendant", domain.PapelAtendente},
		{"receptionist", domain.PapelRecepcionista},
	}
	conjunto := map[string]bool{}
	for _, r := range roles {
		conjunto[strings.ToLower(strings.TrimSpace(r))] = true
	}
	for _, p := range prioridade {
		if conjunto[p.externo] {
			return p.interno, true
		}
	}
	return "", false
}
