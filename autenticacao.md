# Autenticação pelo portal externo (opcional)

O template **não traz login**. Este guia mostra como ativar a autenticação quando o projeto
precisar, seguindo o Princípio X da [constitution](../.specify/memory/constitution.md).

No modelo usado aqui, a aplicação **não tem tela de login nem senha**. A pessoa já está logada
no portal da organização cliente (ex.: uma prefeitura) e chega à aplicação com um token
assinado. A aplicação confere o token, cria ou atualiza o usuário e abre uma sessão própria
num cookie.

## 0. Antes de começar

- **Pergunte ao desenvolvedor** se o projeto usa autenticação. Sem um "sim" explícito, não
  ative nada.
- Registre a decisão no README do projeto e na spec/plano da primeira funcionalidade que usar
  o usuário.
- Defina com quem pediu o projeto:
  - quem entra: só a equipe da organização (profissionais), só cidadãos ou os dois;
  - quais papéis de profissional existem no projeto (ex.: `admin`, `gestor`, `operador`) e de
    quais `roles` do provedor cada um vem;
  - se existe admin da plataforma sem organização.

## 1. Visão geral

```text
Portal da organização (provedor de identidade: Keycloak, Auth0, Azure AD...)
   │  a pessoa clica em "Abrir aplicação"
   ▼
GET https://app.exemplo/qualquer/rota?token=<JWT do provedor>
   │
   ▼
Middleware de entrada (shared/middleware)
   1. lê o "iss" do token e acha a organização cadastrada com esse emissor
   2. confere a assinatura com as chaves públicas (JWKS) do emissor
   3. extrai a identidade (CPF/e-mail) e o papel
   4. cria ou atualiza o usuário no banco
   5. grava um cookie de sessão próprio (JWT HS256)
   6. redireciona para a MESMA URL sem o ?token
   │
   ▼
Próximas requisições: só o cookie
   - o middleware confere a assinatura do cookie (rápido, sem banco)
   - guardas e ações releem usuário, papel e organização do banco
```

Ideias centrais:

- **Dois tokens.** O do provedor (assinatura RS256/384/512) só é usado uma vez, na entrada.
  Depois vale o cookie da aplicação (HS256).
- **A organização é descoberta pelo emissor (`iss`).** Não existe "escolha sua organização".
- **O banco é a fonte da verdade.** O cookie diz só quem é a pessoa e de qual organização;
  papel e situação (ativo) são relidos a cada requisição.

## 1.1 Os dois tipos de token

O mesmo provedor emite tokens para profissionais e para cidadãos. Os exemplos abaixo têm dados
fictícios.

**Profissional** (equipe da organização): traz `roles`; a identidade é o e-mail.

```json
{
  "iss": "https://portal.exemplo.gov.br",
  "aud": "minha-aplicacao.exemplo.com.br",
  "sub": "92",
  "exp": 1792499813,
  "iat": 1792499783,
  "jti": "c98271bc-aa59-40c8-9eda-4acd0704f2d9",
  "name": "Maria Profissional",
  "email": "maria@exemplo.gov.br",
  "preferred_username": "maria@exemplo.gov.br",
  "email_verified": true,
  "roles": ["admin"],
  "department_ids": [],
  "unit_ids": [],
  "map_item_ids": []
}
```

**Cidadão**: não traz `roles`; a identidade é o CPF (em `cpf` e em `preferred_username`).

```json
{
  "iss": "https://portal.exemplo.gov.br",
  "aud": "minha-aplicacao.exemplo.com.br",
  "sub": "96398",
  "exp": 1792500075,
  "iat": 1792500045,
  "jti": "e936512a-0603-4bd4-9745-f52aa1ddd06b",
  "name": "JOAO CIDADAO DA SILVA",
  "email": "joao@exemplo.com",
  "preferred_username": "52998224725",
  "email_verified": true,
  "cpf": "52998224725"
}
```

Como decidir:

| O token traz                                  | Resultado |
| --------------------------------------------- | --------- |
| um papel de profissional reconhecido em `roles` | profissional com o papel mapeado; identidade pelo e-mail |
| nenhum papel reconhecido e um CPF válido      | cidadão; identidade pelo CPF |
| nenhum papel reconhecido e nenhum CPF válido  | recusado (`credencial_sem_identificador`) |

Mapeamento de exemplo (do mais alto para o mais baixo; ajuste aos papéis do projeto):

| `roles` do provedor        | Papel no projeto |
| -------------------------- | ---------------- |
| `super_admin` ou `admin`   | `admin`          |
| `contract_admin`           | `gestor`         |
| `manager`                  | `operador`       |
| nenhum dos anteriores      | `cidadao` (se tiver CPF válido) |

Os campos `department_ids`, `unit_ids` e `map_item_ids` dizem a que setores e unidades o
profissional está ligado. Se o projeto precisar restringir o acesso por eles, grave-os junto do
usuário e atualize a cada entrada.

O `aud` é a audiência: cada aplicação tem a sua (o domínio dela). Como o mesmo provedor emite
tokens para várias aplicações, conferir o `aud` impede que um token emitido para outra
aplicação seja aceito aqui.

## 2. Dependências

```bash
go get github.com/golang-jwt/jwt/v5
```

O `github.com/google/uuid` já está no template.

## 3. Variáveis de ambiente

Acrescente no `.env.example` e leia em `internal/core/config/config.go`, validando na subida:

```env
# Segredo do cookie de sessão (32 caracteres ou mais fora de development)
TOKEN_SEGREDO=troque-por-uma-chave-longa-e-aleatoria-com-32-ou-mais
# Duração da sessão (ex.: 8h, 30m)
TOKEN_DURACAO=8h
# "aud" que o token do portal precisa ter: o domínio desta aplicação (obrigatório fora de development)
AUDIENCIA_ESPERADA=minha-aplicacao.exemplo.com.br
# Emissores confiáveis da plataforma, separados por vírgula (só entram com papel admin).
# Obrigatório para o setup inicial: só um admin destes emissores cadastra a primeira organização.
EMISSORES_ADMIN=https://portal.exemplo.gov.br
```

Regras da validação:

- `TOKEN_SEGREDO` vazio: em `development` use um segredo fixo e registre um aviso; fora de
  `development` a aplicação **não sobe**. Com menos de 32 caracteres fora de `development`,
  também não sobe.
- `TOKEN_DURACAO` precisa passar no `time.ParseDuration`.
- `AUDIENCIA_ESPERADA` vazia: em `development` registre um aviso; fora de `development` a
  aplicação **não sobe**.

## 4. Banco

### Model `internal/models/organizacao.go`

| Campo         | Tipo                  | Observação                                                  |
| ------------- | --------------------- | ----------------------------------------------------------- |
| `ID`          | `int32`, PK           |                                                             |
| `Slug`        | texto, único          | gerado a partir do nome                                     |
| `Nome`        | texto                 | demais dados de negócio do projeto                          |
| `Emissor`     | `*string`             | `iss` do provedor, **já normalizado**; único entre as ativas |
| `URLChaves`   | `*string`             | JWKS; vazio = `{emissor}/.well-known/jwks.json`             |
| `Habilitado`  | `bool`, padrão `true` | desligar bloqueia a entrada sem apagar                      |
| `CriadoEm`, `AtualizadoEm` | data/hora |                                                       |
| `ExcluidoEm`  | `*time.Time`          | exclusão lógica                                             |

### Model `internal/models/usuario.go`

| Campo            | Observação                                                  |
| ---------------- | ----------------------------------------------------------- |
| `ID`             | UUID                                                        |
| `Nome`           | vindo do token                                              |
| `Papel`          | com `check` listando os papéis do projeto, incluindo `cidadao` |
| `OrganizacaoID`  | `*int32`, FK; nulo só para o admin da plataforma            |
| `CPF`, `Email`   | `*string`, únicos; usados para reencontrar a pessoa         |
| `Ativo`          | desativar corta o acesso na hora                            |
| `UltimoAcessoEm` |                                                             |

Não existe coluna de senha nem tabela de sessões.

### Migração

Adicione uma migração no fim de `internal/core/database/migrations.go` com o `AutoMigrate` dos
dois models e o índice único parcial do emissor, para que uma organização excluída não "ocupe"
o emissor:

```go
tx.Exec(`CREATE UNIQUE INDEX organizacoes_emissor_ativo ON organizacoes (emissor)
	WHERE emissor IS NOT NULL AND excluido_em IS NULL`)
```

## 5. Onde fica cada parte

```text
internal/
  core/config/                       TOKEN_SEGREDO, TOKEN_DURACAO, AUDIENCIA_ESPERADA, EMISSORES_ADMIN
  models/organizacao.go, usuario.go
  shared/
    repository/organizacao.go, usuario.go, auditoria.go
    services/autenticacao/
      emissor.go                     NormalizarEmissor e MesmoHost
      jwks.go                        cache das chaves públicas
      entrada.go                     EntrarComCredencial (seção 7)
      erros.go                       tabela única de códigos de erro
      teste.go                       token de teste (só development)
    services/sessao/
      sessao.go                      cookie, UsuarioDaSessao, ContextoPorOrganizacao, TokenCSRF
      recusar.go                     RecusarSemSessao (redirect na página, JSON na API)
    middleware/
      portal.go                      troca do ?token e barreira de cookie
      guardas.go                     ExigirAutenticado, ExigirAdmin..., VerificarCSRF
  modules/acesso/                    página /acesso-negado e POST /sair
cmd/
  seed/                              organização de teste
  token/                             gera o token de teste
```

## 6. Normalização do emissor

O `iss` do token e o emissor cadastrado são comparados de forma **exata**, então os dois passam
pela mesma função. Use-a no cadastro **e** na entrada.

```go
// NormalizarEmissor devolve o emissor na forma usada nas comparações, ou "" se inválido.
func NormalizarEmissor(valor string) string {
	u, err := url.Parse(strings.TrimSpace(valor))
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + hostComPorta(u) + strings.TrimRight(u.EscapedPath(), "/")
}

// MesmoHost diz se a URL é http(s) e está no mesmo host (com porta) do emissor.
func MesmoHost(endereco, emissor string) bool {
	a, err := url.Parse(strings.TrimSpace(endereco))
	if err != nil || (a.Scheme != "http" && a.Scheme != "https") {
		return false
	}
	b, err := url.Parse(emissor)
	return err == nil && hostComPorta(a) != "" && hostComPorta(a) == hostComPorta(b)
}

// hostComPorta devolve o host em minúsculas, sem a porta padrão (443 no https, 80 no http).
func hostComPorta(u *url.URL) string {
	host, porta := strings.ToLower(u.Hostname()), u.Port()
	if (u.Scheme == "https" && porta == "443") || (u.Scheme == "http" && porta == "80") {
		porta = ""
	}
	if porta != "" {
		return host + ":" + porta
	}
	return host
}
```

Query e fragmento são descartados; o caminho é mantido, porque identifica o "realm" do
provedor. `MesmoHost` impede que alguém cadastre chaves num servidor de terceiros.

## 7. Entrada pelo portal

### 7.1 Middleware (`shared/middleware/portal.go`)

Registre com `app.Use(...)` no `router/`, antes das rotas.

- **Rotas livres** (não passam pelo middleware): `/acesso-negado`, `/saude`, `/assets/*` e
  webhooks de terceiros que conferem a própria assinatura.
- Com `?token=`: chama `EntrarComCredencial`. Deu certo: grava o cookie, define
  `Referrer-Policy: no-referrer` e redireciona (302) para a URL atual sem o `token`. Deu
  errado: registra só o código e o **tamanho** do token (nunca o token), apaga o cookie e
  recusa.
- Sem `?token=`: lê o cookie com `LerTokenSessao`; inválido, recusa com `sem_sessao`.

Redirecionar sem o `?token` tira o token da barra de endereço, do histórico, do `Referer` e dos
logs de acesso.

### 7.2 `EntrarComCredencial`, nesta ordem

A ordem define qual erro a pessoa vê e evita buscar chaves para emissores desconhecidos.

1. **Ler sem confiar**: decodificar o payload sem verificar, só para pegar o `iss`
   (`jwt.NewParser().ParseUnverified`). Falhou: `credencial_invalida`.
   `NormalizarEmissor(iss)` vazio: `organizacao_nao_atendida`.
2. **Achar a organização** pelo emissor (exato, não excluída). Se não houver:
   - emissor em `EMISSORES_ADMIN`: segue, mas só entra com papel `admin`; se ainda não existir
     nenhuma organização (excluídas não contam), vai para o setup inicial (seção 11.1);
   - senão, ou se a organização estiver desabilitada: `organizacao_nao_atendida`.
3. **Conferir a assinatura**: URL das chaves = `URLChaves` ou `{emissor}/.well-known/jwks.json`;
   se não for `MesmoHost`, `identidade_indisponivel`. Depois:

   ```go
   opcoes := []jwt.ParserOption{
   	jwt.WithValidMethods([]string{"RS256", "RS384", "RS512"}), // nunca "none" nem HS*
   	jwt.WithIssuer(iss),
   	jwt.WithLeeway(60 * time.Second),
   	jwt.WithExpirationRequired(),
   }
   if audiencia != "" {
   	opcoes = append(opcoes, jwt.WithAudience(audiencia))
   }
   _, err := jwt.ParseWithClaims(credencial, &c, func(t *jwt.Token) (any, error) {
   	kid, _ := t.Header["kid"].(string)
   	return chaves.Chave(ctx, urlChaves, kid)
   }, opcoes...)
   ```

   Falha de rede, timeout ou JWKS ilegível: `identidade_indisponivel` (503). Assinatura,
   expiração ou `aud` inválidos: `credencial_invalida` (401).
4. **Identidade** (só depois da assinatura): CPF de `cpf` ou de `preferred_username` (11
   dígitos que passam na validação) e e-mail em minúsculas.
5. **Papel** (seção 1.1): mapear `roles` (ou `realm_access.roles`). Com papel de profissional
   reconhecido, é profissional e precisa ter e-mail. Sem papel reconhecido, é cidadão e precisa
   ter CPF válido. Sem nenhum dos dois: `credencial_sem_identificador`. Emissor da plataforma
   só entra com papel `admin`.
6. **Usuário (upsert)**: buscar por CPF ou e-mail. Existe e está inativo:
   `usuario_desativado`. Atualizar nome, papel, organização e último acesso, ou inserir com
   `ativo = true`. Se o e-mail do token já for de **outro** usuário, manter o antigo. Registrar
   auditoria (criado, papel ou organização mudou, entrada).

O papel é **reaplicado a cada entrada**: se a organização promover ou rebaixar a pessoa, vale
no próximo acesso.

### 7.3 Cache das chaves (JWKS)

- Um conjunto de chaves por URL, protegido por `sync.Mutex`.
- Timeout curto na busca (3 s) e no máximo uma nova busca a cada 5 minutos por URL.
- Ler no máximo 1 MB da resposta (`io.LimitReader`).
- Usar só chaves `kty=RSA` com `use` vazio ou `sig`.

## 8. Sessão própria (cookie)

```go
// Claims do cookie: sem papel (papel e situação vêm do banco a cada requisição).
type Claims struct {
	jwt.RegisteredClaims
	Org *int32 `json:"org"`
}

func (g *Gerenciador) EmitirTokenSessao(usuarioID uuid.UUID, orgID *int32) (string, error) {
	agora := time.Now()
	c := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer: "nome-da-aplicacao", Subject: usuarioID.String(), ID: uuid.NewString(),
			IssuedAt: jwt.NewNumericDate(agora), ExpiresAt: jwt.NewNumericDate(agora.Add(g.duracao)),
		},
		Org: orgID,
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(g.segredo)
}
```

- **Leitura rápida** (`LerTokenSessao`, usada no middleware): só HS256, `Issuer` da aplicação,
  `exp` obrigatório, `sub` UUID e `jti` preenchido. Qualquer falha devolve `nil`. Não toca no
  banco.
- **Cookie**: nome `sessao`, `HTTPOnly: true`, `Secure` fora de `development`,
  `SameSite: fiber.CookieSameSiteLaxMode` (a entrada é uma navegação vinda de outro site),
  `Path: "/"`, `MaxAge` igual a `TOKEN_DURACAO`.
- **Leitura completa** (`UsuarioDaSessao`, guardada em `c.Locals` para não repetir na mesma
  requisição): relê o usuário pelo `sub`; recusa se não existir, estiver inativo ou se a
  organização mudou em relação ao cookie. Sem organização, só o admin passa. Com organização,
  ela precisa existir e estar habilitada.
- **Saída** (`POST /sair`): apagar o cookie, registrar auditoria e ir para
  `/acesso-negado?motivo=sessao_encerrada`. Para cortar alguém na hora, desative o usuário ou
  desabilite a organização.

## 9. Autorização

- **Guardas por rota** em `shared/middleware/guardas.go`, aplicadas no `router/`:

  | Guarda               | Aceita                       | Os demais |
  | -------------------- | ---------------------------- | --------- |
  | `ExigirAutenticado`  | qualquer sessão válida       | acesso negado |
  | `ExigirCidadao`      | só cidadão                   | profissional vai para o painel |
  | `ExigirProfissional` | admin, gestor e operador     | cidadão vai para a área dele |
  | `ExigirGestor`       | admin e gestor               | acesso negado (`papel_sem_acesso`) |
  | `ExigirAdmin`        | só o admin da plataforma     | volta para o painel |

  ```go
  painel := app.Group("/painel", guardas.ExigirProfissional())
  admin := app.Group("/admin", guardas.ExigirAdmin())
  // Área do cidadão: a guarda vai em cada rota (um grupo sem prefixo viraria middleware global).
  app.Get("/", guardas.ExigirCidadao(), inicioHdl.Index)
  ```

- **Cidadão só vê o que é dele**: as consultas do cidadão filtram pelo CPF (ou ID) da sessão,
  incluindo os dependentes dele; nunca por um CPF vindo do navegador.

- **Organização sempre da sessão**: `ContextoPorOrganizacao(c, pedido)` devolve o ID da
  organização do usuário e **ignora** o que veio do navegador; só o admin da plataforma usa o
  pedido. Assim um gestor nunca lê nem altera dados de outra organização.
- **Toda ação confere a permissão no servidor.** Esconder um botão não é autorização.
- **API interna**: quando a requisição é para `/api/`, recuse com JSON (status 401/403) e o
  campo `redirecionar` apontando para a página de acesso negado, em vez de redirecionar; o
  `$sharedApi` leva a pessoa para lá. Um redirecionamento HTTP faria o `fetch` receber a
  página de erro como se fosse a resposta.

### CSRF

Métodos que alteram dados (`POST`, `PUT`, `PATCH`, `DELETE`) exigem um token derivado da
sessão:

```go
func (g *Gerenciador) TokenCSRF(c fiber.Ctx) string {
	cl := g.ClaimsDaRequisicao(c)
	if cl == nil {
		return ""
	}
	mac := hmac.New(sha256.New, g.segredo)
	mac.Write([]byte("csrf:" + cl.ID))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
```

- O middleware `VerificarCSRF` compara com `subtle.ConstantTimeCompare` o valor do cabeçalho
  `X-CSRF-Token` ou do campo `_csrf`.
- No layout, coloque o token numa meta tag; o `$sharedApi` lê e manda no cabeçalho
  `X-CSRF-Token` em todo pedido que altera dados:

  ```html
  <meta name="csrf-token" content={ csrf }/>
  ```

- Em formulários normais (sem a API), use `<input type="hidden" name="_csrf" value={ csrf }/>`.

## 10. Erros e página de acesso negado

Uma tabela única em `shared/services/autenticacao/erros.go`:

| Código                         | HTTP | Quando                                              |
| ------------------------------ | ---- | --------------------------------------------------- |
| `sem_sessao`                   | 401  | sem cookie válido                                   |
| `sessao_encerrada`             | 200  | depois do logout                                    |
| `credencial_invalida`          | 401  | token malformado, assinatura errada, expirado, `aud` errado |
| `credencial_sem_identificador` | 401  | token sem CPF nem e-mail                            |
| `organizacao_nao_atendida`     | 403  | emissor não cadastrado ou organização desabilitada  |
| `papel_sem_acesso`             | 403  | nenhum papel reconhecido                            |
| `usuario_desativado`           | 403  | usuário com `ativo = false`                         |
| `identidade_indisponivel`      | 503  | não foi possível buscar as chaves do provedor       |

`RecusarSemSessao(c, codigo)` apaga o cookie e leva a `/acesso-negado?motivo=<codigo>` (na API,
por JSON com `redirecionar`). A página, no módulo `acesso`, mostra título e mensagem da tabela e
orienta a voltar pelo portal da organização.

## 11. Cadastro de organizações

É o cadastro que **libera a entrada**: sem organização habilitada com o emissor certo, ninguém
dela entra. No painel do admin (rotas com `ExigirAdmin`):

- Lista das organizações não excluídas, com busca por nome ou emissor, mostrando o emissor (ou
  "sem integração") e um selo quando desabilitada.
- Formulário com nome, emissor (opcional), URL das chaves (opcional) e "habilitado".
- Validação no servidor:
  - emissor preenchido e `NormalizarEmissor` vazio: "Emissor inválido: informe a URL http(s)
    do realm do provedor.";
  - URL das chaves sem emissor: "Informe o emissor antes da URL das chaves.";
  - URL das chaves fora do host do emissor: "A URL das chaves precisa estar no mesmo host do
    emissor.";
  - grave o emissor **já normalizado**;
  - violação do índice único: "Este emissor já está cadastrado em outra organização."
- Excluir é exclusão lógica (`excluido_em`). Todas as buscas, inclusive a da entrada e a da
  sessão, filtram as excluídas.
- Desabilitar uma organização derruba as sessões dela na próxima requisição; trocar o emissor
  deixa de aceitar tokens do antigo.

### 11.1 Setup inicial (primeira organização)

Numa instalação nova o banco não tem nenhuma organização, então não há emissor cadastrado para
conferir. Quem faz o primeiro cadastro é um admin de um emissor de `EMISSORES_ADMIN`, que é a
lista de provedores em que a aplicação confia desde a instalação.

```text
token de um emissor de EMISSORES_ADMIN + banco sem nenhuma organização
   │  assinatura, exp e aud conferidos (passos 3 e 4)
   │  papel mapeado precisa ser admin; senão papel_sem_acesso
   ▼
cookie temporário de setup (HS256, 15 min) com o iss e os dados do token
   │  redireciona para /setup
   ▼
tela de setup: emissor já preenchido e sem edição + nome da organização + URL das chaves
(opcional) + demais dados do projeto
   │  POST /setup
   ▼
transação: trava, confere de novo que não existe organização, cria a organização,
cria o usuário admin vinculado, auditoria, abre a sessão normal
```

Regras:

- Token de emissor fora de `EMISSORES_ADMIN` nunca vê o setup: com o banco vazio, ele recebe
  `organizacao_nao_atendida`. Por isso `EMISSORES_ADMIN` precisa estar preenchido na primeira
  subida em produção.
- Cidadão e profissional sem papel `admin` nunca veem a tela de setup (`papel_sem_acesso`).
- A tela usa o layout padrão, o CSRF e as mensagens em modal ou no formulário (sem `alert()`).
- A transação usa `clause.Locking` (ou um advisory lock) para dois admins não criarem duas
  "primeiras" organizações ao mesmo tempo; se já existir uma, o setup é recusado.
- Depois da primeira organização, `/setup` responde acesso negado e as próximas são cadastradas
  pelo painel do admin (seção 11).

## 12. Testes sem o portal real

Como não existe tela de login, os testes usam um token fabricado.

1. **Atalho na verificação** (`shared/services/autenticacao/teste.go`): uma constante com um
   sufixo fixo (ex.: `.ZGVzZW52b2x2aW1lbnRv`, "desenvolvimento" em base64url). Com
   `APP_ENV=development` e o token terminando nesse sufixo, **só a assinatura** é dispensada:
   emissor cadastrado e habilitado, `exp`, identificador e papel continuam obrigatórios. Fora
   de `development` o sufixo não tem efeito e o token é recusado.
2. **Organização de teste** (`cmd/seed`): cria uma organização fictícia, habilitada, com um
   emissor fictício (ex.: `http://localhost:8080/realms/teste`) numa constante exportada. O
   seed **não cria usuários**: eles nascem no primeiro acesso.
3. **Gerador de token** (`cmd/token`): monta `cabeçalho.payload` em base64url no formato da
   seção 1.1, com `iss`, `sub`, `aud`, `iat`, `exp` (+1 h) e dados fictícios válidos;
   acrescenta o sufixo no lugar da assinatura e imprime a URL de entrada.
   - Papel `cidadao`: sem `roles`, com `cpf` e `preferred_username` iguais ao CPF.
   - Papéis de profissional: com `roles` (ex.: `["admin"]`) e e-mail em `email` e
     `preferred_username`.

Tarefas no `Taskfile.yml`:

```yaml
seed:
  desc: Cria a organização de teste (só APP_ENV=development).
  cmd: go run ./cmd/seed

token:dev:
  desc: "Imprime a URL de entrada com um token de teste. Uso: task token:dev -- [papel] [emissor] [cpf]"
  cmd: go run ./cmd/token {{.CLI_ARGS}}
```

Abra a URL impressa no navegador: a aplicação troca o token pelo cookie e você já entra.

Dicas:

- Para testar os erros, gere tokens com emissor não cadastrado, papel desconhecido,
  organização desabilitada ou usuário desativado.
- CPFs fictícios precisam passar na validação de dígitos.

**Homologação:** prefira um provedor de teste de verdade (ex.: um realm de homologação no
Keycloak) cadastrado numa organização de teste. Subir homologação com `APP_ENV=development`
deixa **qualquer pessoa que conheça o sufixo entrar como qualquer usuário**; só faça isso num
ambiente fechado e sem dados reais. Em produção, `APP_ENV` nunca é `development`.

## 13. Checklist

1. Confirmar com o desenvolvedor que o projeto usa autenticação e registrar a decisão.
2. `go get github.com/golang-jwt/jwt/v5`.
3. Variáveis no `.env.example` e validação em `core/config`.
4. Models `Organizacao` e `Usuario`, migração com o índice parcial do emissor.
5. `NormalizarEmissor` e `MesmoHost`, usados no cadastro e na entrada.
6. Cache de JWKS com timeout, intervalo mínimo e limite de tamanho.
7. `EntrarComCredencial` na ordem da seção 7.2.
8. Gerenciador de sessão: cookie HS256 sem papel, `UsuarioDaSessao`, `ContextoPorOrganizacao` e
   `TokenCSRF`.
9. Middleware de entrada com `app.Use`, guardas por rota e `VerificarCSRF`; meta tag com o
   token no layout.
10. Módulo `acesso` com a página de acesso negado e o `POST /sair`.
11. Cadastro de organizações no admin, com validação e exclusão lógica, e o setup inicial
    (seção 11.1), só para admin de um emissor de `EMISSORES_ADMIN`.
12. Auditoria de entrada, saída e mudanças de usuário e organização.
13. Atalho de teste (só development), `cmd/seed`, `cmd/token` e as tarefas no Taskfile.
14. Testes: entrada com token válido, cada código de erro, guardas e CSRF.

## 14. Cuidados de segurança

- Nunca confiar no payload antes de conferir a assinatura; o `iss` lido antes serve **só** para
  achar a organização.
- Emissor desconhecido nunca é aceito, senão qualquer um assinaria um token com papel admin.
- Algoritmos fixos: RS* para o token do provedor, HS256 para o cookie.
- Chaves só do mesmo host do emissor.
- Nunca registrar o token em log; no máximo o tamanho.
- Tirar o token da URL logo na entrada, com `Referrer-Policy: no-referrer`.
- Permissão sempre conferida no servidor, em cada ação, com o banco como fonte da verdade.
