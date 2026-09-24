# Autenticação por URL: validando a credencial do Conecta Cidades

Como o Emissão de Carteiras recebe a credencial do portal da prefeitura pela URL, como valida esse
token e como o troca por uma sessão própria.

Público: quem for implementar ou revisar a validação da credencial em outro serviço, ou entender
por que uma entrada foi recusada.

> ⚠ O contrato com o Conecta Cidades ainda não foi confirmado formalmente. Os nomes de parâmetro e
> claims abaixo são o desenho implementado hoje (`internal/constants/conecta_constant.go`);
> confirmar antes de integrar com prefeitura nova.

---

## 1. O fluxo em uma frase

O portal da prefeitura já identificou a pessoa, reabre esta aplicação com um JWT na query string, e
a aplicação **verifica esse JWT offline** (assinatura contra o JWKS do realm) e o troca por um
**cookie de sessão próprio**. A credencial do portal é usada **uma única vez**, na abertura.

```
Portal da prefeitura                Frontend (SPA)                 API Go
        │                                 │                            │
        │  abre  https://app/?token=JWT   │                            │
        ├────────────────────────────────>│                            │
        │                                 │ POST /api/v1/sessao/portal │
        │                                 │      { "credencial": JWT } │
        │                                 ├───────────────────────────>│
        │                                 │                            │ 1. lê `iss` (sem confiar)
        │                                 │                            │ 2. iss → município
        │                                 │                            │ 3. baixa JWKS do realm
        │                                 │                            │ 4. verifica assinatura,
        │                                 │                            │    exp, aud
        │                                 │                            │ 5. resolve CPF/e-mail
        │                                 │                            │ 6. resolve papel (roles)
        │                                 │                            │ 7. cria/atualiza usuário
        │                                 │  Set-Cookie: sessao=<JWT>  │
        │                                 │<───────────────────────────┤
        │                                 │ limpa ?token= da URL       │
```

A partir daí, **quem identifica é o cookie** — a credencial do Conecta não volta a ser usada.

---

## 2. Como a credencial chega pela URL

| Item | Valor |
| --- | --- |
| Parâmetro | `token` (`constants.ParamCredencial`, espelhado em `web/src/constants/index.ts` → `paramCredencial`) |
| Onde | Query string, **só** na abertura da aplicação |
| Formato | JWT compacto assinado pelo realm da prefeitura |
| Exemplo | `https://emissao.exemplo.gov.br/?token=eyJhbGciOiJSUzI1NiIsImtpZCI6...` |

**Por que query string e não cabeçalho:** a abertura acontece por navegação — o portal abre um
link, e navegação de navegador não carrega cabeçalho próprio.

**Por que só na abertura:** query string fica no histórico, no `Referer` e no log de proxy. É um
custo aceitável uma vez, na entrada, e não em toda requisição (spec 002 FR-009, FR-010).

### O consumo no cliente

Na guarda de navegação (`web/src/router/index.ts`), antes de qualquer decisão de rota:

1. lê `?token=` da query;
2. chama `POST /api/v1/sessao/portal` com a credencial no corpo;
3. **remove o parâmetro do endereço** com `replace: true`, para não deixá-lo no histórico.

Isso fica na guarda, e não em `onMounted` de componente: o `onMounted` roda depois da primeira
navegação, e a guarda já teria mandado para `/entrar` quem acabara de chegar com credencial válida.

---

## 3. O token esperado

```json
{
  "sub": "ID do servidor",
  "iss": "https://rede.ipojuca.pe.gov.br",
  "aud": "emissao.exemplo.gov.br",
  "iat": 1755720000,
  "exp": 1755720900,
  "jti": "ID único do token",
  "preferred_username": "servidor@prefeitura.gov.br",
  "email": "servidor@prefeitura.gov.br",
  "email_verified": false,
  "name": "Nome do servidor",
  "roles": ["manager"],
  "department_ids": [3, 7],
  "unit_ids": [12],
  "cpf": "12345678909"
}
```

| Claim | Uso | Obrigatório |
| --- | --- | --- |
| `iss` | resolve o município, por comparação **exata** com os emissores cadastrados | **sim** |
| `exp` | prazo; tolerância de 60s de relógio | **sim** |
| `aud` | conferido contra `CONECTA_AUDIENCIA`, quando configurada | condicional |
| `roles` | perfil de acesso | **sim** |
| `cpf` | identidade da pessoa; chave preferida do vínculo | não, se houver `email` |
| `email` | identifica a pessoa quando não há CPF | não, se houver `cpf` |
| `preferred_username` | segunda tentativa de CPF, **só** se passar no dígito verificador | não |
| `name` | nome exibido | não |
| `sub`, `jti`, `email_verified`, `department_ids`, `unit_ids` | **não são lidos**, por decisão | — |

**Assinatura:** `RS256`, `RS384` ou `RS512` — lista fechada (`constants.AlgoritmosAceitos`).

---

## 4. A validação, passo a passo

Implementação: `internal/services/conecta_service.go` (`VerificarCredencial`).
A ordem não é livre — ela define qual recusa a pessoa vê quando mais de uma se aplica.

### 4.1 Ler o `iss` sem confiar nele

O payload é decodificado em base64url **sem verificar assinatura**, apenas para descobrir *contra
qual chave* conferir. Nada desse payload é usado como verdade: logo em seguida o parser exige que o
`iss` do token seja exatamente esse mesmo valor, então um emissor mentiroso só consegue ser mandado
para a chave errada — que não confere.

### 4.2 Normalizar o `iss`

`utils.NormalizarEmissor` (`internal/utils/emissor.go`):

- esquema e host viram minúsculas (insensíveis a caixa por RFC 3986);
- barra final é removida;
- query e fragmento são descartados;
- **o caminho é preservado como veio** — path é sensível a caixa, e um realm do Keycloak mora
  justamente no caminho (`/auth/realms/Servidor`); rebaixá-lo recusaria todo token daquela cidade.

Sem esquema ou sem host → `ErrEmissorInvalido`.

### 4.3 Resolver `iss` → município

Consulta à tabela `municipio_emissores` (`MunicipioService.PorEmissor`), **a cada entrada** — não é
um mapa montado no arranque, e por isso alterar um emissor vale sem reiniciar.

- Comparação **exata**. Não há casamento por hostname, por prefixo nem por parecença.
- Origem desconhecida é recusa seca: **não existe município padrão** e não há palpite a partir do
  nome do host. Aceitar de qualquer origem faria a credencial de uma prefeitura valer na outra.
- Uma cidade pode ter **vários** emissores (produção e homologação apontam para o mesmo município).
- Município desabilitado é recusado **antes** de qualquer consulta ao provedor.

### 4.4 Obter a chave pública (JWKS)

`internal/services/chaves_service.go`:

- Endereço: `{iss}/.well-known/jwks.json`, ou o `url_chaves` declarado no cadastro daquela origem.
- **Trava de mesmo host** (`utils.MesmoHost`): o endereço das chaves precisa estar no mesmo hostname
  do emissor. Sem ela, um endereço cadastrado por engano — ou copiado de um documento de descoberta
  adulterado — mandaria buscar a chave num servidor escolhido por quem forjou o token, e a
  assinatura **conferiria**. A checagem existe no cadastro e de novo aqui, para valer também quando
  a linha for alterada fora da tela.
- Cache em memória por emissor. `kid` não encontrado dispara **uma** rebusca, respeitando o
  intervalo mínimo de 5 minutos — o realm troca de chave sem avisar, mas um `kid` inventado não
  pode virar uma requisição ao provedor por tentativa.
- Só chaves `kty: RSA` e `use` vazio ou `sig`. A chave de criptografia (`use: enc`) nunca assinou
  token nenhum.
- `kid` vazio só é aceito quando o conjunto tem **uma** chave.
- Timeout de 3s, resposta limitada a 1 MiB.
- **Não existe modo degradado**: provedor inalcançável devolve `ErrIdentidadeIndisponivel` → `503`,
  e nega o acesso.

### 4.5 Verificar o token

```go
opcoes := []jwt.ParserOption{
    jwt.WithValidMethods(constants.AlgoritmosAceitos), // RS256/384/512 — lista fechada
    jwt.WithIssuer(emissor),                           // iss exatamente o resolvido
    jwt.WithLeeway(constants.ToleranciaRelogio),       // 60s
    jwt.WithExpirationRequired(),                      // exp obrigatório
}
if audiencia := c.config.Audiencia(); audiencia != "" {
    opcoes = append(opcoes, jwt.WithAudience(audiencia))
}
```

Os quatro pontos que **não** podem ser afrouxados:

1. **Lista fechada de algoritmos.** Sem ela, um token com `alg: none`, ou assinado com HMAC usando a
   própria chave pública como segredo, passaria. O token não pode escolher como será conferido.
2. **`iss` exigido no parser.** Evita que um token de outro emissor seja aceito contra a chave certa
   por acaso.
3. **`exp` obrigatório.** Token sem prazo seria credencial eterna. Tolerância de 60s porque relógio
   de servidor de prefeitura não é sincronizado com o nosso.
4. **`aud` conferida quando configurada.** É o que impede um token emitido para **outro sistema da
   mesma prefeitura** — assinado pelo mesmo realm, portanto de assinatura válida — de abrir sessão
   aqui. A audiência é da **instalação** (hostname deste serviço), não da cidade.

### 4.6 Resolver a identidade — depois da assinatura

CPF primeiro, e-mail depois:

1. `cpf`, se passar na validação de dígito verificador;
2. senão `preferred_username`, **só** se passar no dígito verificador (em vários realms de governo o
   login é o próprio CPF);
3. senão, o `email` (minúsculo, sem espaços) identifica a pessoa.

Sem CPF **e** sem e-mail: `ErrCredencialSemIdentificador`. Nenhuma sessão, nenhum usuário criado.

### 4.7 Resolver o papel — depois da assinatura

`roles` é uma lista; vale o **mais alto** entre os reconhecidos:

| Papel na credencial | Perfil aqui | Alcance |
| --- | --- | --- |
| `super_admin` | `admin` (apelido) | tudo dentro do município emissor |
| `admin` | `admin` | tudo dentro do município emissor, e nada fora |
| `contract_admin` | `contract_admin` | administração municipal |
| `manager` | `manager` | o do operador, mais editar e ativar template existente |
| `operator` | `operator` | emitir lote e baixar carteirinha |

Qualquer outro papel da plataforma mãe (`document_validator`, `event_manager`, `attendant`,
`receptionist`, `facilitator`, `operational_viewer`, `management_viewer`) **não concede nada**, e
papel criado depois desta entrega cai no mesmo lugar: **falha fechada**, não um nível mínimo
silencioso. Nenhum papel reconhecido → `ErrPapelSemAcesso` → `403`.

> Os passos 4.6 e 4.7 vêm **depois** da assinatura de propósito: antes de saber quem assinou, nada
> no token é verdade, e responder "falta CPF" a um token forjado entregaria o formato esperado a
> quem não devia tê-lo.

---

## 5. Ordem completa das checagens

1. `iss` presente e legível → senão, **credencial inválida**.
2. `iss` casa exatamente com um emissor cadastrado → senão, **cidade não atendida**.
3. Município habilitado → senão, **cidade não atendida**.
4. Chaves obtidas do emissor → senão, **identificação indisponível**.
5. Assinatura, `exp` e (quando configurada) `aud` conferem → senão, **credencial inválida**.
6. CPF **ou** e-mail utilizável → senão, **credencial sem identificador**.
7. Papel reconhecido em `roles` → senão, **sem permissão de acesso**.
8. Usuário não está desativado aqui → senão, **acesso revogado**.

O passo 3 vem antes do 4 para não consultar o provedor de uma cidade desligada.

---

## 6. O que acontece depois: usuário e sessão

`internal/services/identidade_service.go` (`EntrarPeloPortal`).

**Usuário.** Busca por CPF, depois por e-mail; não encontrado, **cria na hora**, sem espera por
aprovação. O contrapeso é a administração de usuários, que desativa acesso indevido. Usuário
desativado **não é recriado** — e recebe `ErrIdentidadeInvalida`, não um erro próprio: dizer "sua
conta foi desativada" confirmaria que ela existe.

**Perfil e município são cópia da última credencial**, reescritos a cada entrada. Quem muda o perfil
é a prefeitura, no realm dela; toda mudança entra na trilha de auditoria com valor anterior e novo.

**Sessão própria.** Um JWT desta aplicação (`internal/services/token_service.go`):

| Item | Valor |
| --- | --- |
| Cookie | `sessao` — `HttpOnly`, `Secure` (fora de dev), `SameSite=Lax` |
| Algoritmo | `HS256`, segredo `TOKEN_SEGREDO` |
| `iss` | `emissao-carteiras` (exigido também na leitura) |
| Claims | `sub` = id do usuário, `mun` = município do vínculo, `jti`, `iat`, `exp` |
| Duração | `TOKEN_DURACAO`, padrão `8h` — dimensionada para uma jornada de trabalho |

`SameSite=Lax` e não `Strict`: a entrada vem de navegação a partir do portal da prefeitura, que é
outro domínio — `Strict` bloquearia justamente a porta principal do produto.

**O perfil NÃO entra no token de sessão.** Perfil e situação do usuário são lidos do banco a cada
requisição (`IdentidadeService.Autenticar`), e é isso que faz uma promoção ou uma desativação valer
na requisição seguinte, em vez de esperar o token expirar.

**Não há estado de sessão no servidor.** Sair remove o cookie mas não invalida o token, que vale até
expirar; o que corta acesso na hora é desativar o usuário. Decisão explícita: a API precisa ser
alcançável por outras aplicações, e uma tabela de sessões tornaria o acesso dependente de um
registro que só este produto cria.

---

## 7. As recusas

| Situação | Código | HTTP | O que a pessoa lê |
| --- | --- | --- | --- |
| Sem credencial | `identidade_ausente` | `401` | precisa abrir o serviço pelo portal da prefeitura |
| Credencial inválida ou expirada | `identidade_invalida` | `401` | identificação inválida; reabrir pelo portal |
| Cidade não atendida ou desabilitada | `cidade_nao_atendida` | `403` | o serviço ainda não está disponível nesta cidade |
| Credencial sem CPF nem e-mail | `credencial_sem_identificador` | `401` | a credencial não traz CPF nem e-mail; a prefeitura precisa incluir um dos dois |
| Nenhum papel reconhecido | `papel_sem_acesso` | `403` | não há papel atribuído para este serviço |
| Provedor inalcançável / JWKS ilegível | `identidade_indisponivel` | `503` | não foi possível confirmar a identidade agora |
| Usuário desativado aqui | `usuario_desativado` | `403` | acesso revogado nesta aplicação |
| Cookie ausente ou inválido | `sessao_invalida` | `401` | sessão expirada |

Tabela única de tradução em `internal/routers/middleware_erro.go` — nenhum service conhece código
HTTP.

As duas linhas de credencial válida que não entra (`credencial_sem_identificador` e
`papel_sem_acesso`) quebram a regra de mensagem genérica **de propósito**: nos dois casos a pessoa
não tem o que corrigir, a correção é da prefeitura, e uma mensagem genérica aqui não protege nada —
quem chegou até esse ponto já apresentou credencial com assinatura válida.

**"Não vale" e "não consegui conferir" são distintos de propósito.** O primeiro é `401` e recai
sobre quem tentou entrar; o segundo é `503`, nega igual, mas sai **destacado** no log, porque
costuma ser configuração errada e derruba a cidade inteira. Toda recusa vai para a trilha de
auditoria com município, motivo e momento, com o identificador pessoal mascarado.

---

## 8. Configuração

| Variável | Papel |
| --- | --- |
| `CONECTA_AUDIENCIA` | identificador desta instalação, esperado no `aud`. **Em branco, qualquer token do realm da prefeitura é aceito — inclusive o emitido para outro sistema dela.** O arranque avisa quando falta. |
| `TOKEN_SEGREDO` | assina o cookie de sessão. Mínimo 32 caracteres fora de dev. Quem tem esse valor emite sessão para qualquer usuário; trocá-lo derruba todas as sessões abertas. |
| `TOKEN_DURACAO` | prazo da sessão própria (padrão `8h`) |
| `APP_ENV` | `development` libera os atalhos da seção 9 |

Por município, no cadastro (`municipios` / `municipio_emissores`):

- `emissor` — o `iss` completo, normalizado. Cidade **sem emissor não aceita ninguém**.
- `url_chaves` — opcional; só para provedor que publique o JWKS fora do caminho padrão. Precisa
  estar no mesmo host do emissor.
- `url_portal` — para onde a tela de login manda a pessoa. Vazio, a cidade não aparece na lista.
- `habilitado` — desligar bloqueia inclusive quem já estava dentro.

---

## 9. Atalhos de desenvolvimento (não existem em produção)

Ambos presos a `APP_ENV=development`, com a tranca no **service** e não no controller, para que uma
rota nova desprotegida não consiga abri-los por acidente.

1. **Credencial de exemplo** — token terminado em `.ZGVzZW52b2x2aW1lbnRv` tem o payload lido **sem
   verificação de assinatura nenhuma**. Continua exigindo CPF ou e-mail.
2. **Identidade fictícia** — `POST /api/v1/sessao/desenvolvimento` com `municipioSlug` e `perfil`
   cria sessão para uma das três pessoas fictícias do código, sem provedor e sem token. Fora de
   `development` responde `404`, como se a rota não existisse. Sai marcada como tal na auditoria.

---

## 10. Endpoints

| Método | Rota | Sessão | O que faz |
| --- | --- | --- | --- |
| `POST` | `/api/v1/sessao/portal` | não | troca a credencial do portal por cookie de sessão |
| `POST` | `/api/v1/sessao/login` | não | devolve a URL do portal da cidade escolhida (não cria sessão) |
| `GET` | `/api/v1/sessao` | sim | diz quem está do outro lado do cookie |
| `DELETE` | `/api/v1/sessao` | sim | remove o cookie |
| `POST` | `/api/v1/sessao/desenvolvimento` | não | só em `development` |

Requisições autenticadas passam pelo `MiddlewareIdentidade`, que lê o cookie, chama
`IdentidadeService.Autenticar` e deposita usuário e município ativo em `Locals` — **sem decidir
nada**. Quem responde "pode?" é o `AutorizacaoService`, na entrada de cada controller.

O cabeçalho `X-Municipio-Ativo` permite **pedir** outro município; pedir não é obter — para quem não
é `admin`, o `AutorizacaoService` recusa qualquer município diferente do vínculo.

---

## 11. Checklist para quem for validar um token do Conecta

- [ ] Ler o `iss` sem confiar — só para escolher a chave.
- [ ] Normalizar o `iss` (caixa no host, sem barra final, **path preservado**).
- [ ] Casar o `iss` com uma origem **cadastrada**, por comparação exata. Sem padrão, sem palpite.
- [ ] Recusar cidade desabilitada antes de chamar o provedor.
- [ ] Buscar o JWKS no host **do emissor**, e só nele.
- [ ] Fixar a lista de algoritmos (`RS256/384/512`). Nunca deixar o token escolher.
- [ ] Exigir `exp`, com tolerância de relógio.
- [ ] Conferir `aud` contra o identificador da instalação.
- [ ] Só depois de tudo isso, ler `cpf` / `email` / `roles`.
- [ ] Papel fora da lista fechada não concede nada.
- [ ] Provedor fora do ar **nega** o acesso — nunca há modo degradado.
- [ ] Consumir a credencial uma vez e limpar a URL.

---

## Arquivos de referência

| Assunto | Arquivo |
| --- | --- |
| Verificação da credencial | `internal/services/conecta_service.go` |
| JWKS e cache de chaves | `internal/services/chaves_service.go` |
| Usuário, papel e sessão | `internal/services/identidade_service.go` |
| Token próprio (cookie) | `internal/services/token_service.go` |
| Rotas e cookie | `internal/controllers/auth_controller.go`, `internal/routers/router.go` |
| Middleware de identidade | `internal/routers/middleware_identidade.go` |
| Erros → HTTP | `internal/routers/middleware_erro.go` |
| Constantes da integração | `internal/constants/conecta_constant.go` |
| Normalização do emissor | `internal/utils/emissor.go` |
| Consumo do `?token=` no cliente | `web/src/router/index.ts`, `web/src/stores/sessao.ts` |
| Contrato formal | `specs/011-emissor-por-cidade/contracts/credencial.md` |
