---
name: auth-conecta-cidades
description: Fluxo real de SSO (JWT via URL) da plataforma Conecta Cidades — IMPLEMENTADO (22/10). ÚNICO método de login do sistema desde 22/10 (login por senha foi removido por completo, ver seção "Login por senha removido por completo").
---

# Autenticação real — SSO da plataforma Conecta Cidades

⚠️ **Decisão de 19/09 revertida em 22/10**: estava "adiado pro pós-MVP" até a credencial de
introspecção (`introspect_api`) chegar via Léo — assim que confirmou funcionar de verdade
contra a API real, o dono do produto pediu implementação imediata ("pode meter marcha!!!!").
**Implementado e no ar** (ver seção "Implementação real (22/10)" mais abaixo). **Atualização,
mesmo dia**: o login por senha não ficou convivendo em paralelo por muito tempo — o dono do
produto pediu a remoção completa dele na sequência ("derrube toda aquela autenticação antiga
que tinhamos feito e suba apenas com o metodo jwt"). Este SSO é hoje o **único** jeito de entrar
no sistema — ver seção "Login por senha removido por completo" no fim deste documento, e a
skill `auth-login-mvp` (agora marcada obsoleta) pro registro histórico de como funcionava antes.

Baseado em `_docs-externas/autenticacao-por-url-conecta.md` (documentação de um produto irmão nessa mesma
plataforma, "Emissão de Carteiras", que já valida essa credencial em produção — adaptado aqui
pro nosso domínio: papéis `gestor/atendente/recepcionista`, hierarquia
prefeitura→secretaria→unidade). **Contrato ainda não confirmado formalmente com a equipe da
Conecta Cidades pro nosso serviço especificamente** — os nomes de claim abaixo são o desenho já
validado por um serviço irmão, mas nosso `iss`/`aud` própios ainda precisam ser cadastrados por
eles.

⚠️ **Não confundir com CPF do cidadão.** Este documento é sobre autenticar o **funcionário**
(gestor/atendente/recepcionista) que opera o painel — não tem relação com o CPF do **cidadão**
vindo da planilha de agendamento (esse não precisa mais de máscara nenhuma, decisão de 19/09, ver
`CLAUDE.md` raiz).

## 1. O fluxo, em uma frase

O portal da prefeitura já autenticou a pessoa, abre nosso painel com um JWT na **query string**,
nosso backend **verifica esse JWT offline** (assinatura contra o JWKS do realm da prefeitura) e
o troca por um **cookie de sessão próprio**. A credencial do portal é usada **uma única vez**, na
abertura — depois disso, quem identifica é o nosso cookie.

```
Portal da prefeitura          Frontend (React)              Backend (Go)
      │                              │                            │
      │ abre /?token=JWT             │                            │
      ├─────────────────────────────>│                            │
      │                              │ POST /api/v1/sessao/portal │
      │                              │      { "credencial": JWT } │
      │                              ├───────────────────────────>│
      │                              │                            │ valida (seção 3)
      │                              │  Set-Cookie: sessao=<próprio JWT>
      │                              │<───────────────────────────┤
      │                              │ remove ?token= da URL      │
```

## 2. Consumo do `?token=` no frontend

- Lido **na guarda de navegação** (antes de decidir a rota), não em `onMounted`/`useEffect` de
  componente — senão a guarda já teria mandado pra tela de login quem chegou com credencial
  válida.
- Envia pro backend, e **remove o parâmetro da URL logo em seguida** (`history.replaceState` ou
  equivalente do router) — query string fica no histórico do navegador, no `Referer` e em log de
  proxy; não pode ficar lá além do instante da troca.

## 3. Claims esperadas no token do portal

```json
{
  "sub": "id do servidor",
  "iss": "https://jaboataomaisfacil.jaboatao.pe.gov.br",
  "aud": "painel.jaboatao.pe.gov.br",
  "iat": 1755720000,
  "exp": 1755720900,
  "jti": "id único do token",
  "preferred_username": "servidor@prefeitura.gov.br",
  "email": "servidor@prefeitura.gov.br",
  "email_verified": false,
  "name": "Nome do servidor",
  "roles": ["attendant"],
  "department_ids": [3],
  "unit_ids": [12],
  "cpf": "12345678909"
}
```

**Confirmado com um token real de teste (22/09)** — dono do produto gerou um JWT de verdade
logando como atendente na plataforma real e decodificou. Exemplo real (conta de teste dele):

```json
// header
{ "alg": "RS256" }
// payload
{
  "sub": "590",
  "exp": 1790083603,
  "preferred_username": "reilobovine@gmail.com",
  "email": "reilobovine@gmail.com",
  "email_verified": true,
  "name": "Vinícius K Lima - Testes ADM",
  "roles": ["attendant"],
  "department_ids": [469],
  "unit_ids": [],
  "iat": 1790083573,
  "jti": "eabfc54e-1a28-47f5-9409-e8eede48ac33",
  "aud": "google.com",
  "iss": "https://jaboataomaisfacil.jaboatao.pe.gov.br"
}
```

**Correção (22/09, mesmo dia)**: o `aud: "google.com"` desse exemplo foi um artefato do
**próprio teste** — o dono do produto mandou o token pra URL `google.com` só pra decodificar,
não é o `aud` que a plataforma emitiria numa integração de verdade. **A suposição original
(seção 3: `aud` dedicado tipo `painel.jaboatao.pe.gov.br`, cadastrado pra nós) continua de pé**
— o teste real só confirma que `aud` É um campo configurável do lado de quem pede o token (por
isso apareceu com o valor pro qual foi mandado), o que é uma boa notícia: dá pra pedir à equipe
da plataforma pra emitir com o nosso `aud` de verdade quando essa integração for registrada.
Lição pra não repetir: **um valor de claim que parece estranho/genérico (tipo um domínio de
terceiro) pode ser só um artefato de como o teste foi feito — perguntar a origem antes de
documentar como comportamento fixo da plataforma**, o que eu não fiz da primeira vez.

Achados que continuam de pé desse token real:

- **`iss` confirmado exatamente** como o valor que já estava assumido aqui
  (`https://jaboataomaisfacil.jaboatao.pe.gov.br`) — item que estava pendente na seção 7,
  agora resolvido.
- **`unit_ids` veio vazio pro papel `attendant`** — mas isso também pode ser só um dado dessa
  conta de teste específica (pode não ter unidade cadastrada nela) em vez de uma regra geral do
  papel `attendant`. Não tratar como confirmado até ver mais de um exemplo — mas vale já prever
  um fallback (ex: perguntar a unidade no primeiro login) pro caso de vir vazio de verdade.
- **Sem claim `cpf`** nesse exemplo — só `preferred_username`/`email` (o mesmo valor nos dois).
  Consistente com o desenho de já tratar `cpf` como opcional.
- **Janela `exp - iat` de só 30 segundos** (`1790083603 - 1790083573`) — confirma na prática o
  desenho de "token de troca única, usado uma vez na abertura" (seção 1): não dá nem pra cogitar
  usar esse JWT como sessão de verdade, ele expira quase imediatamente por natureza.

| Claim | Uso no nosso painel | Obrigatório |
| --- | --- | --- |
| `iss` | resolve a prefeitura, comparação **exata** contra `prefeituras.issuer_jwt` cadastrado | **sim** |
| `exp` | prazo; tolerância de 60s de relógio — mas na prática já vem com só ~30s de validade | **sim** |
| `aud` | conferido contra o `aud` da nossa instalação (ver "Pendências") | **sim assim que tivermos um `aud` cadastrado** |
| `roles` | papel de acesso — ver mapeamento na seção 6 | **sim** |
| `department_ids` | secretaria(s) do usuário, via `mapeamento_externo` (tipo `department`) | condicional — ver seção 6 |
| `unit_ids` | unidade(s) do usuário, via `mapeamento_externo` (tipo `unit`) — **um teste veio vazio pra atendente, mas pode ser só dado dessa conta especifica, não confirmado como regra** | condicional — ver seção 6, considerar fallback |
| `cpf` | identidade da pessoa (funcionário); chave preferida do vínculo | não, se houver `email` |
| `email` | identifica quando não há `cpf` | não, se houver `cpf` |
| `preferred_username` | segunda tentativa de identidade, só se passar no dígito verificador do CPF | não |
| `name` | nome exibido | não |
| `sub`, `jti`, `email_verified` | não precisam ser lidos, seguindo a decisão do produto irmão | — |

**Assinatura:** lista fechada `RS256`/`RS384`/`RS512` — nunca deixar o token escolher o algoritmo
(evita ataque de `alg: none` ou HMAC com a chave pública como segredo).

## 4. Validação passo a passo (ordem importa)

1. **Ler o `iss` sem confiar nele** — decodificar o payload sem verificar assinatura, só pra
   saber contra qual chave conferir. Em seguida o parser exige que o `iss` do token verificado
   seja exatamente esse mesmo valor — um emissor mentiroso só erra a chave, não engana o parser.
2. **Normalizar o `iss`**: esquema e host em minúsculas, sem barra final, sem query/fragmento.
   **Preservar o caminho como veio** (path é sensível a caixa — um realm pode morar no caminho,
   ex: `/auth/realms/Jaboatao`).
3. **Resolver `iss` → prefeitura**: comparação **exata** contra `prefeituras.issuer_jwt`. Sem
   casamento por prefixo, sem palpite por hostname. Origem desconhecida = recusa seca — não
   existe prefeitura padrão.
4. **Buscar o JWKS**: `{iss}/.well-known/jwks.json` (ou uma URL alternativa cadastrada, mas
   **travada ao mesmo host do emissor** — sem essa trava, uma URL de chaves adulterada faria a
   assinatura conferir contra uma chave escolhida por quem forjou o token). Cache em memória;
   `kid` desconhecido dispara no máximo uma rebusca a cada 5min. Timeout curto (3s), resposta
   limitada em tamanho. **Provedor fora do ar = nega o acesso** (503) — nunca existe modo
   degradado que aceita sem verificar.
5. **Verificar**: algoritmo da lista fechada, `iss` exato, `exp` obrigatório (60s de tolerância),
   `aud` conferido contra o nosso identificador de instalação (`painel.jaboatao.pe.gov.br` ou o
   que for cadastrado com a equipe da plataforma — ver "Pendências").
6. **Só depois da assinatura conferir**, resolver identidade (`cpf` → `preferred_username` → `email`)
   e papel (`roles`, ver seção 6). Nessa ordem de propósito: antes de validar a assinatura, nada
   no token é confiável, e recusar por "falta CPF" a um token forjado entregaria de graça o
   formato esperado pra quem não devia saber.
7. Papel não reconhecido = **falha fechada** (nenhum acesso), nunca um nível mínimo silencioso.

## 5. Sessão própria (depois da troca)

- Cookie próprio (ex: `sessao`), `HttpOnly`, `Secure` (fora de dev), `SameSite=Lax` — **não**
  `Strict`, porque a entrada vem de navegação a partir de outro domínio (o portal), e `Strict`
  bloquearia justamente essa entrada.
- JWT próprio assinado com segredo simétrico nosso (`HS256`), claims mínimas (`sub`=id do
  usuário local, tenant ativo, `jti`, `iat`, `exp`), duração pensada pra uma jornada de trabalho
  (produto irmão usa 8h).
- **Papel e vínculo NÃO ficam gravados no token de sessão** — relidos do banco a cada requisição,
  pra uma promoção ou desativação valer imediatamente, sem esperar o token expirar.
- Sem estado de sessão no servidor (sem tabela de sessões) — sair remove o cookie mas não invalida
  o token; o que corta acesso de verdade é desativar o usuário no banco.
- Usuário: busca por CPF, depois por email; se não existir, **cria na hora**, sem aprovação
  prévia (mesmo padrão de `criadoViaSso` que já está no schema). Usuário desativado não é
  recriado.

## 6. Mapeamento de papel

**Confirmado diretamente pelo dono do produto (22/10)**, que tem visão de dentro da própria
hierarquia de papéis da Conecta Cidades (não é mais inferência por nome parecido):

- **`admin`/`super_admin`** são o nível MAIS ALTO da plataforma inteira — "o maior abaixo só
  dos devs, nós da LegalTech" (o dono do produto é quem detém esse nível, usado só pra testar
  — não é um papel de prefeitura nenhuma). Mapeado pro nosso `domain.PapelAdmin`, mas **NUNCA
  concedido via SSO** (ver `PostSessaoPortal`, bloqueio explícito) — `admin` no token significa
  "admin de toda a plataforma Conecta Cidades", não "admin da nossa instalação", e conceder
  nosso admin global (que enxerga TODAS as prefeituras cadastradas aqui) a esse nível seria
  escalação de privilégio entre tenants.
- **`contract_admin`** é o admin DA PREFEITURA especificamente — tenant-scoped, confirmado
  como "o admin da prefeitura vai vir como contract_admin". Mapeado pro nosso `gestor` (nosso
  papel mais alto que ainda é escopado a uma secretaria, nunca global).
- **`manager`** continua mapeado pra `gestor` também — mesmo papel "gerencial" genérico de
  antes, confiança média (pode ser um nível intermediário dentro da plataforma deles, mas
  nosso domínio só tem um nível "gestor", não distingue).
- **`attendant`/`receptionist`** continuam `atendente`/`recepcionista`, confiança alta (nome
  bate literalmente, e o produto irmão já tratava esses dois como papéis "de fora" do domínio
  dele, evidenciando que são nomes globais da plataforma).

| `roles` (JWT) | Papel no nosso painel | Confiança |
| --- | --- | --- |
| `admin`, `super_admin` | `admin` (mas BLOQUEADO via SSO, só manual) | confirmado — mas escopo global incompatível com SSO |
| `contract_admin` | `gestor` | confirmado (22/10) — "admin da prefeitura" |
| `manager` | `gestor` | média — papel gerencial genérico, significado exato ainda pode variar |
| `attendant` | `atendente` | alta — nome bate literalmente |
| `receptionist` | `recepcionista` | alta — nome bate literalmente |

`department_ids`/`unit_ids`: diferente do produto irmão (que não lê esses campos porque o
domínio dele não precisa de sub-hierarquia dentro do município), **o nosso painel precisa
deles** — mapear via `mapeamento_externo` (`tipo = 'department'` → secretaria, `tipo = 'unit'` →
unidade), como já estava previsto no `CLAUDE.md`. Vazios é esperado só se o papel resolvido for
algo com alcance de prefeitura inteira (não é bem o caso do nosso MVP de unidade única).

**`department_ids` = secretaria, achado uma confirmação de verdade (22/09)**: o OpenAPI da
própria plataforma (`_docs-externas/conecta-cidades-openapi.json`), num endpoint bem diferente
(enquetes, não autenticação), tem um campo `department_name` com o exemplo
`"Secretaria de Atendimento"` — ou seja, a própria plataforma usa "department" como o nome em
inglês de "secretaria" no domínio dela, fora do contexto do JWT. Isso eleva a confiança do
mapeamento `department_ids → secretaria` de "suposição por nome parecido" pra "confirmado por
outra parte da API real" — ainda não é 100% (não vimos o ID numérico 469 do token de teste
"traduzido" pra um nome de secretaria de verdade), mas é uma evidência concreta, não só palpite.

**`unit_ids` continua sem nenhuma confirmação equivalente** — o OpenAPI (que é escopado pra
ações do cidadão, via `citizen_cpf`) não tem nenhuma referência a "unit" em lugar nenhum. Não
achamos nada na documentação disponível que confirme "unit" = nossa "unidade" (local físico) —
continua sendo só a suposição mais razoável dado o nome, não confirmada em fonte nenhuma.

## 7. Recusas (adaptar a mesma tabela)

| Situação | Código sugerido | HTTP |
| --- | --- | --- |
| Sem credencial | `identidade_ausente` | 401 |
| Credencial inválida ou expirada | `identidade_invalida` | 401 |
| Prefeitura não cadastrada/desabilitada | `prefeitura_nao_atendida` | 403 |
| Sem CPF nem e-mail utilizável | `credencial_sem_identificador` | 401 |
| Papel não reconhecido | `papel_sem_acesso` | 403 |
| JWKS/provedor inalcançável | `identidade_indisponivel` | 503 |
| Usuário desativado aqui | `usuario_desativado` | 403 |
| Cookie de sessão ausente/inválido | `sessao_invalida` | 401 |

`identidade_indisponivel` (503) é distinto de propósito dos `401`/`403` — o primeiro é "a
prefeitura está fora do ar", o segundo é "essa pessoa/token não vale". Vale logar o 503 destacado
(costuma indicar configuração errada, derruba a prefeitura inteira, não só uma pessoa).

## Pendências

- ~~Confirmar `iss`~~ **✅ resolvido (22/09)** — token real confirmou
  `https://jaboataomaisfacil.jaboatao.pe.gov.br` exatamente como suposto.
- **Ainda precisa cadastrar um `aud` dedicado pro painel com a equipe da plataforma** — um teste
  real (22/09) mostrou `aud: "google.com"` num token, mas era só porque o dono do produto mandou
  esse token de teste pra `google.com` de propósito pra decodificar, não o valor real que a
  plataforma emitiria numa integração registrada. O desenho original (pedir um `aud` tipo
  `painel.jaboatao.pe.gov.br`) continua sendo o caminho — o teste só confirma que `aud` É um
  campo que a plataforma deixa configurável por quem solicita o token, o que é bom sinal.
- **Confirmar o mapeamento de `roles`** (seção 6) — `attendant`/`receptionist` são só inferência
  por nome parecido, não confirmação oficial. O token real confirmou que `roles: ["attendant"]`
  é exatamente o formato (array com essa string), então a FORMA está certa — só falta confirmar
  que o SIGNIFICADO (attendant = nosso atendente) é mesmo esse com a equipe da plataforma.
- **`unit_ids` vazio num teste com `attendant`** (22/09) — pode ser só porque essa conta de teste
  específica não tem unidade cadastrada, não necessariamente uma regra geral do papel. Vale
  perguntar à equipe da plataforma se `unit_ids` sempre vem preenchido pra atendente de verdade,
  e mesmo assim prever um fallback (perguntar unidade no primeiro login?) por segurança.
- O schema atual (`modelo-dados`) tem `prefeituras.issuer_jwt` como **um único** valor único —
  o produto irmão permite uma prefeitura ter **vários** emissores (produção + homologação
  apontando pro mesmo município). Se isso for necessário aqui também, o schema precisa de uma
  tabela `prefeitura_emissores` em vez de uma coluna única — não mexer nisso sem necessidade
  confirmada.

## Confirmação por documentação OFICIAL real (16/10) — deixa de ser inferência de produto irmão

Até aqui, tudo acima vinha de um produto IRMÃO (Emissão de Carteiras) + um token real decodificado
manualmente — nunca da documentação oficial do fluxo em si. Em 16/10, o dono do produto deu acesso
a uma instância admin real da MESMA plataforma (`pvhmais.portovelho.ro.gov.br/docs/integracoes` —
Porto Velho-RO, não Jaboatão, mas é o doc genérico "Conecta Cidades — Integrações Externas"
v1.0.0, o mesmo produto, versionado — valores como `iss` mudam por município, o FLUXO e os
endpoints não). Essa doc tem uma seção dedicada **"Login Integrado (SSO) — Servidor"** (distinta
de "— Cidadão", que é outro fluxo, pra autenticar munícipe, não funcionário) — confirma o desenho
desta skill quase exatamente, e acrescenta uma peça que faltava:

**Fluxo, confirmado literalmente**: "Servidor faz login no painel administrativo da prefeitura →
clica num link customizado com integração JWT habilitada → sistema gera um token JWT no momento
do clique → servidor é direcionado pro sistema externo com o token na URL
(`https://exemplo.com/painel-bi?token=...`) → sistema externo valida o token e autentica
automaticamente." Validade de **30 segundos**, **uso único**, compatível com Keycloak, RS256.
**A integração é habilitada individualmente por link, na configuração do painel** — ou seja,
alguém com acesso admin na plataforma da prefeitura precisa CADASTRAR o painel de chamada como
uma integração antes de qualquer token real ser emitido pra ele (ver "O que falta pra implementar"
abaixo).

**Claims, confirmadas literalmente** (bate com a seção 3 acima, sem surpresa):
`sub, iss, aud, iat, exp, jti, preferred_username, email, email_verified, name, roles[],
department_ids[], unit_ids[], cpf?`. Exemplo real da doc: `"department_ids": [3, 7]`,
`"unit_ids": [12]` — **confirma que os dois são arrays** (um servidor pode ter mais de uma
secretaria/unidade), consistente com o que já estava assumido aqui. `cpf`: "é OPCIONAL e
**normalmente ausente** — nunca assuma presença. Use `sub` ou `preferred_username` como
identificador" — bate exatamente com a seção 3.

**Peça nova que a skill não tinha: a validação PRECISA de uma segunda chamada, obrigatória,
pra um endpoint de introspection — não é só verificação de assinatura offline.**

```
POST /oauth/introspect
Authorization: Bearer <TOKEN DE API COM PERMISSÃO introspect_api>  -- token nosso, não o do usuário
Content-Type: application/json

{ "token": "<JWT recebido no ?token= da URL>" }
```

Resposta (200): `{ "active": true|false, "jti", "aud", "exp", "iat", "payload": {...claims} }` —
**uma segunda chamada com o MESMO token responde `active: false`** (é isso que implementa o
"uso único" de verdade — decodificar e verificar a assinatura localmente não é suficiente pra
saber se aquele token específico já foi consumido antes). Doc oficial, literal: **"Todos os
sistemas integrados DEVEM chamar este endpoint após as verificações locais"** — não é opcional,
nem um "nice to have" de segurança extra. Os dois passos são complementares: **"valide localmente
E DEPOIS chame a introspection (obrigatória)"**. A seção 4 desta skill (validação passo a passo)
precisa ganhar um passo 8: depois de tudo validado localmente, chamar `/oauth/introspect` e só
aceitar `active: true` antes de criar a sessão própria — um 429 aqui tem o mesmo tratamento de
rate limit já documentado na skill `api-conecta-cidades`.

**Descoberta automática, padrão OIDC** (preferir a isso em vez de URL fixa, pra sobreviver rotação
de chave): `GET /.well-known/openid-configuration` (discovery) e `GET /.well-known/jwks.json`
(chaves públicas) — ambos sem autenticação, mesmo padrão que a seção 4 já assumia.

## O que falta pra implementar de verdade (resposta direta: dá, mas falta registro, não código)

**Sim, dá pra implementar** — o desenho já é sólido e agora tem confirmação oficial, não só
inferência. O que falta não é mais "descobrir como funciona", é um registro administrativo do
lado da plataforma, que só quem tem acesso ao painel admin da prefeitura consegue fazer:

1. **Cadastrar o painel de chamada como uma "integração"/link** no painel administrativo da
   prefeitura de Jaboatão (`jaboataomaisfacil.jaboatao.pe.gov.br`, não o Porto Velho usado pra
   ler a doc) — é isso que gera o `aud` de verdade e habilita a emissão de token pra essa
   integração especificamente.
2. **Conseguir um token de API com a permissão `introspect_api`**, específico do município de
   Jaboatão — **distinto** de qualquer token de `scheduling_api`/`dependents_api` já discutido
   pra outra finalidade (agendamento). Sem esse token, o backend não consegue completar o passo
   obrigatório de introspection, então não dá pra aceitar nenhum login SSO de verdade.
3. Confirmar com a equipe da plataforma o mapeamento de `roles` (pendência já listada acima) —
   agora com uma pergunta concreta pra fazer, já que o formato está 100% confirmado.

Depois desses 3 itens resolvidos (são burocracia/contato com a equipe da plataforma, não
trabalho de código), a implementação em si segue o desenho que já está pronto nesta skill
(seções 1-7) mais o passo de introspection acima — **não precisa mais de nenhuma pesquisa**,
só decidir quando puxar isso do pós-MVP pra dentro do escopo.

## Ambiente de teste real: prefeitura "Smartândia" (16/10)

A Conecta Cidades mantém uma cidade de testes pública de verdade — `mock.conectacidades.app`
("Smartândia"), com `.well-known/jwks.json` e `.well-known/openid-configuration` reais,
alcançável sem precisar de credencial de nenhum município real. O dono do produto já tem
acesso lá (gerou um token de teste real, `roles: ["admin"]`, `aud` já apontando pro domínio
real do nosso Vercel — confirma que ele já cadastrou o painel como integração lá, resolvendo
sozinho o item 1 da lista acima, pelo menos pro ambiente de teste).

**Criamos a prefeitura correspondente no nosso banco** (16/10, não é descartável — fica como
tenant real): `nome='Smartândia'`, `slug='smartandia'`,
`issuer_jwt='https://mock.conectacidades.app'` — é esse campo que resolve `iss → prefeitura`
na validação (seção 4, passo 3). Junto, uma secretaria ("Secretaria de Atendimento ao
Cidadão"/SAC), uma unidade ("CRAS Smartândia Centro"), uma grade ("Atendimento Geral") e um
guichê padrão — ambiente mínimo já pronto pra testar não só login de admin, mas também
atendente/recepcionista/gestor via SSO quando o mapeamento de `roles` for exercido de
verdade.

**Mapeamento `admin` confirmado**: diferente das outras linhas da tabela da seção 6 (ainda
inferência), o dono do produto confirmou diretamente que `roles: ["admin"]` do token mapeia
pro nosso `domain.PapelAdmin` (acesso master, sem vínculo de prefeitura/secretaria) — não é
mais suposição marcada como pendente.

Ainda falta, mesmo pra testar contra a Smartândia: o token de API com permissão
`introspect_api` (item 2 da lista acima) — sem ele, o passo 8 da validação (introspection
obrigatória) não tem como ser implementado nem testado de ponta a ponta.

## Confirmação final: doc oficial do PRÓPRIO Jaboatão, não mais de outro município (22/10)

Até aqui a confirmação "oficial" vinha de uma instância de outro município (Porto Velho-RO,
16/10) — o mesmo produto, mas não o nosso `iss` de verdade. Em 22/10, lida a documentação
pública direto em `jaboataomaisfacil.jaboatao.pe.gov.br/docs/integracoes` (a doc é renderizada
pelo Scalar a partir de um YAML OpenAPI injetado no HTML da página — `curl` simples já traz o
spec inteiro, sem precisar de JS/navegador). **Confirma tudo que já estava documentado aqui
sem nenhuma contradição**, e acrescenta 4 coisas novas de verdade:

1. **Regra exata do `aud`, antes só aproximada**: "`aud` deve ser exatamente o seu domínio,
   **sem http/https, páginas ou parâmetros**" — ou seja, hostname puro (ex. `painel-de-
   chamada-kappa.vercel.app`), nunca uma URL completa nem um caminho. Refina a seção 3/seção
   7 passo 5 desta skill (que já assumia isso, mas sem a confirmação literal "sem protocolo/
   caminho/query").
2. **Chave pública RSA real de Jaboatão**, publicada em texto puro na própria doc (pra
   verificação local sem precisar buscar o JWKS toda vez) — a doc mesmo recomenda preferir
   `GET /.well-known/jwks.json` em vez de fixar essa chave no código, por causa de rotação
   futura; a chave fixa existe só como fallback/exemplo. Não reproduzida aqui por ser long-
   lived e rotacionável — pegar direto da doc ou do JWKS no momento de implementar.
3. **Schema completo do `POST /oauth/introspect` e dos dois endpoints de descoberta**
   (`/.well-known/jwks.json`, `/.well-known/openid-configuration`) — confirma exatamente o
   que a seção acima já tinha (request `{token}`, resposta `{active, jti, aud, exp, iat,
   payload}`), e acrescenta os casos de erro: `400` token ausente, `401` token de API inválido
   ou sem permissão `introspect_api`, `429` rate limit (ver item 4).
4. **Rate limit por token de API**: 300 requisições/minuto e 30000/dia (ajustável por token),
   com headers `X-RateLimit-Limit`/`X-RateLimit-Remaining`/`X-RateLimit-Reset` em toda
   resposta autenticada; excedeu, `429` com corpo `RateLimitError` e header `Retry-After`
   (segundos). **A janela diária zera às 00:00 UTC (21:00 horário de Brasília), não à meia-
   noite local** — detalhe fácil de errar num agendador de limpeza/retry. Formato de erro
   padrão da API inteira (não só introspection): `{error, message, error_code, request_id}`
   — vale logar o `request_id` em qualquer chamada que falhar, pra reportar pra equipe da
   plataforma com algo rastreável.

**Checklist "nunca faça" da própria doc** (reforça o que a seção 4/14 desta skill já diz, sem
nada novo em espírito, mas vale ter listado literal): usar dado do token sem verificar
assinatura; não validar `exp`/`iat`; não verificar `iss`/`aud`; não chamar introspection;
aceitar token expirado; reutilizar token já processado; assumir presença do claim `cpf`.

**Sobre o `autenticacao.md` que o Léo mandou (22/10)**: comparado com essa doc oficial real,
confirma a suspeita — aquele arquivo **não é documentação da Conecta Cidades** (usa
framework/ORM diferentes dos nossos, nomes genéricos "organização"/papéis `admin/gestor/
operador`, referencia `.specify/memory/constitution.md` que é coisa do NOSSO spec-kit, e
principalmente **não menciona a introspection obrigatória em nenhum lugar** — que a doc oficial
trata como requisito não-negociável). É mais provável que seja um template genérico de "como
implementar esse padrão de SSO-via-URL" que o Léo tem/usa, não escrito pela equipe da Conecta
Cidades. Dono do produto vai confirmar com ele antes de decidir o que fazer com aquele arquivo
— até lá, a fonte de verdade pra implementação continua sendo exclusivamente a doc oficial
(esta seção + "Confirmação por documentação OFICIAL real", acima).

## Implementação real (22/10)

Credencial de introspecção testada e confirmada real (curl direto contra `POST /oauth/
introspect` com um token inválido de propósito → `200 {"active":false}`, não `401` — confirma
que o token de API em si é válido e tem a permissão certa). Guardada em
`backend/.env` → `CONECTA_INTROSPECT_API_TOKEN` (NUNCA versionada — `.env` está no
`.gitignore`; só o placeholder vazio existe em `.env.example`).

**Schema**: nenhuma migration nova precisou ser escrita — `prefeituras.issuer_jwt`,
`usuarios.external_sub` e a tabela `mapeamento_externo` (`tipo` department/unit →
`secretaria_id`/`unidade_id`) já existiam no banco desde o design original da hierarquia
Admin/Prefeitura (22/09), só nunca tinham sido usadas por código nenhum até agora.

**Backend, arquivos novos**:
- `internal/auth/jwks.go` — cache de JWKS em memória (processo inteiro, por URL), sem
  biblioteca externa (conversão JWK→`*rsa.PublicKey` feita à mão — `n`/`e` em base64url,
  poucas linhas, não justificava uma dependência nova). `kid` desconhecido numa busca FRESCA
  vira `ErrChaveDesconhecida` (credencial inválida, 401); `kid` desconhecido mas cache ainda
  dentro da janela de 5min sem rebuscar vira `ErrJWKSIndisponivel` (503) — achado real
  testando contra a JWKS de verdade da Smartândia: a primeira versão classificava os dois
  casos igual, fazendo um token claramente forjado (kid inexistente) devolver "serviço
  indisponível" em vez de "credencial inválida".
- `internal/auth/portal.go` — `NormalizarEmissor`, `LerEmissorSemVerificar` (lê `iss` sem
  confiar), `VerificarCredencialPortal` (assinatura RS256/384/512 contra o JWKS do emissor
  JÁ resolvido no banco — nunca uma URL vinda de fora, diferente do desenho genérico de
  referência que previa uma `URLChaves` configurável; aqui é sempre
  `{issuer}/.well-known/jwks.json`, derivado do `issuer_jwt` cadastrado), `MapearPapel`
  (`attendant`→atendente e `receptionist`→recepcionista, confiança alta; `manager`→gestor,
  confiança média; `admin`/`super_admin`→admin, confirmado direto pelo dono do produto).
- `internal/auth/introspect.go` — `POST {issuer}/oauth/introspect`, Bearer
  `CONECTA_INTROSPECT_API_TOKEN`. 429/401/erro de rede do lado da Conecta Cidades viram
  `identidade_indisponivel` (503) — são problema de infraestrutura/configuração nossa, não
  credencial ruim do usuário.
- `internal/auth/portal_test.go` — 10 testes de verdade (não mockados) contra uma chave RSA
  gerada na hora + um `httptest.Server` fazendo de JWKS: token válido passa; `aud` errado,
  token expirado, `iss` errado, assinatura forjada (chave diferente da publicada) e algoritmo
  HS256 (nunca aceito, mesmo pedido pelo header) todos recusados; normalização de emissor;
  mapeamento de papel (incluindo prioridade quando mais de um papel reconhecido); introspecção
  (ativo/inativo/token de API inválido). Não dá pra testar contra a chave PRIVADA real da
  Conecta Cidades (não temos acesso a ela, só a pública) — por isso a chave de teste é gerada
  na hora, mas o formato (JWKS em `/.well-known/jwks.json`, claims, RS256) é exatamente o
  documentado oficialmente.
- `internal/repository/mapeamento_externo.go` — `ResolverSecretariaPorExternalIDs`/
  `ResolverUnidadePorExternalIDs`: um usuário só tem UMA secretaria/unidade no nosso schema,
  mas o token pode trazer vários `department_ids`/`unit_ids` — usa o PRIMEIRO que tiver
  mapeamento cadastrado, ignora o resto. Simplificação deliberada (documentada no próprio
  arquivo), não bug.
- `internal/controllers/sso.go` — `PostSessaoPortal`, `POST /api/v1/sessao/portal`
  (público, sem sessão prévia — mesma rota pública que `/login`). Orquestra a ordem exata da
  seção 4: ler `iss` sem confiar → resolver prefeitura → verificar assinatura/claims →
  mapear papel → checar identificador → **introspecção obrigatória** → resolver secretaria/
  unidade → upsert → cookie de sessão. Novo código de erro, não previsto na tabela original:
  `vinculo_nao_encontrado` (403) — papel reconhecido mas `department_ids`/`unit_ids` não
  batem com nenhum `mapeamento_externo` cadastrado.

**Backend, arquivos alterados**: `internal/repository/prefeituras.go`
(`BuscarPrefeituraPorIssuer`), `internal/repository/usuarios.go`
(`BuscarUsuarioPorExternalSub`, `UpsertUsuarioPortal` — busca por `external_sub` primeiro,
`email` depois; papel/vínculo são REAPLICADOS a cada entrada; usuário desativado não é
recriado nem reativado sozinho, quem barra é o controller checando `Ativo` depois do upsert,
igual já fazia o login por senha), `internal/controllers/controllers.go` (3 campos novos:
`ChavesPublicas`, `ConectaAudiencia`, `ConectaIntrospectToken`), `cmd/api/main.go` (rota nova
+ variáveis `CONECTA_AUDIENCIA`/`CONECTA_INTROSPECT_API_TOKEN`).

**Bug real corrigido no processo, sem relação direta com SSO**: `.env` nunca foi de fato
carregado pelo processo Go — todo valor em uso até 22/10 só "funcionava" porque o default
hardcoded em `envOu()` coincidia com o do arquivo. Ficou visível ao adicionar
`CONECTA_INTROSPECT_API_TOKEN`, que não tem (nem deveria ter) um default seguro. Corrigido
adicionando `github.com/joho/godotenv` (`godotenv.Load()` logo no início de `main()`, erro
ignorado de propósito — em produção as variáveis reais vêm do host, sem arquivo `.env`).

**Frontend**: `api.ts` ganhou `loginPortal(credencial)` (`POST /api/v1/sessao/portal`).
`Login.tsx` lê `?token=` da URL numa guarda de entrada (não um componente aninhado) — a URL é
limpa (`history.replaceState`) IMEDIATAMENTE, antes mesmo da resposta do backend chegar
(query string não pode sobreviver no histórico/`Referer`/log de proxy). Sucesso: redireciona
pelo papel, mesmo mapa `DESTINO_POR_PAPEL` que o login por senha já usa. Falha: mostra o erro
e volta pro formulário normal — login por senha continua funcionando como alternativa, nunca
foi removido.

**Testado de ponta a ponta, o que dava pra testar sem navegador**: token de API de
introspecção confirmado real contra o endpoint de produção; os 10 testes automatizados
(`go test ./internal/auth/...`) cobrindo toda a lógica criptográfica; um token forjado com
`iss` real da Smartândia mas assinatura/kid falsos mandado pro endpoint de verdade — confirma
que a busca da prefeitura por emissor E a busca/parsing da JWKS real da Smartândia funcionam
ponta a ponta contra dado externo genuíno (não só contra o JWKS fabricado dos testes),
recusando corretamente com `identidade_invalida`/401.

**Não testado ainda — pendente**: um login completo com um token REAL, recém-gerado,
assinado de verdade pela Smartândia (precisa de alguém logar lá e pegar o token fresco — only
válido por 30s, não dá pra reaproveitar um antigo) — isso cobriria a verificação de assinatura
real passando, a introspecção com um token genuíno, o upsert de usuário e a sessão sendo
criada de ponta a ponta. Também não testado: a resolução de secretaria/unidade via
`mapeamento_externo` com dado de verdade (a tabela está vazia agora — precisa cadastrar pelo
menos uma linha de teste pra isso rodar, ver pendência abaixo). `CONECTA_AUDIENCIA` ainda
vazio no `.env` — e, desde a revisão de segurança abaixo, isso agora DESLIGA o endpoint
inteiro (503), nunca pula a checagem de `aud` — preencher com o valor real assim que um token
de teste revelar qual é.

## Revisão de segurança, mesmo dia (22/10) — 3 achados reais, todos corrigidos

Uma revisão automática rodou logo depois da implementação e achou 3 problemas genuínos —
nenhum "nitpick", os 3 eram caminhos reais de falha de isolamento/autenticação. Corrigidos na
hora, com migration nova (`000011_sso_escopo_tenant`) e 5 testes automatizados novos (4 contra
Postgres real, 1 de unidade) cobrindo especificamente os 3 cenários.

1. **Account takeover entre tenants via `external_sub` sem escopo** (HIGH) — `sub` é um ID
   interno POR MUNICÍPIO na Conecta Cidades (inteiro pequeno, tipo "42"), não globalmente
   único. `usuarios.external_sub` tinha unique GLOBAL — duas prefeituras diferentes com um
   servidor "sub=42" cada uma colidiriam no mesmo registro. Corrigido: coluna nova
   `usuarios.external_issuer`, índice único composto `(external_issuer, external_sub)`
   (migration 000011), toda busca/gravação agora usa o par, nunca o `sub` sozinho.
   Adicionalmente, o fallback de busca por e-mail (quando `external_sub` não bate com nada)
   ganhou duas travas: só roda se `email_verified` vier `true` no token, e nunca REBINDA um
   usuário que já tem um vínculo externo diferente gravado (`ErrVinculoExternoConflitante`,
   recusa em vez de sobrescrever silenciosamente).
2. **Vazamento de acesso entre tenants via `mapeamento_externo` sem escopo** (HIGH) — mesma
   causa raiz do item 1, mas em `department_id`/`unit_id`: a tabela resolvia `(tipo,
   external_id)` → secretaria/unidade SEM considerar de qual prefeitura o token era. Um
   `department_id=3` de uma prefeitura podia resolver pro mapeamento cadastrado por outra.
   Corrigido: coluna `mapeamento_externo.prefeitura_id` (obrigatória), índice único composto
   `(prefeitura_id, tipo, external_id)`, resolução sempre escopada pela prefeitura já
   resolvida via `issuer` — mais um join de defesa em profundidade confirmando que a
   secretaria/unidade apontada realmente pertence àquela prefeitura (protege contra erro de
   cadastro manual, já que não existe tela pra isso ainda).
3. **Checagem de `aud` falhando ABERTA** (MEDIUM) — `VerificarCredencialPortal` só aplicava
   `jwt.WithAudience(...)` quando `audienciaEsperada != ""`; com `CONECTA_AUDIENCIA` vazio (o
   estado real do `.env` até agora), a checagem de audiência era simplesmente PULADA — abria
   caminho pro clássico "ataque de confusão de audiência" (um token válido emitido pra OUTRA
   aplicação integrada ao mesmo emissor seria aceito aqui também). Corrigido em duas camadas:
   `VerificarCredencialPortal` agora recusa (`ErrAudienciaNaoConfigurada`) se chamada sem
   audiência, e `PostSessaoPortal` já barra antes disso (503) se `ConectaAudiencia` estiver
   vazia — mesmo padrão de "falha fechada" que já existia pro token de introspecção.

**Testes novos**: `internal/auth/portal_test.go` ganhou
`TestVerificarCredencialPortal_AudienciaVaziaFalhaFechado`; `internal/repository/sso_test.go`
(novo arquivo) cobre os outros 4 cenários contra Postgres real — dois usuários com o mesmo
`sub` em emissores diferentes nunca colidem; tentar rebindar um vínculo externo já
estabelecido recusa; fallback por e-mail não-verificado falha (constraint) em vez de
vincular/criar; o mesmo `external_id` mapeado em duas prefeituras resolve pra secretaria
certa em cada uma.

**Pendências explícitas**:
- Cadastrar pelo menos uma linha em `mapeamento_externo` (agora com `prefeitura_id`) pra
  testar atendente/recepcionista/gestor via SSO de verdade (hoje só dá pra testar `admin`,
  que não precisa de vínculo).
- ~~Confirmar `CONECTA_AUDIENCIA` com um token real~~ ✅ resolvida, mesmo dia — ver abaixo.
- Mapeamento de papel (`manager`→gestor) continua confiança MÉDIA, não confirmado
  oficialmente — mesma pendência de antes, agora com código de verdade em cima dela.
- Nenhuma tela de UI pra cadastrar `mapeamento_externo` — só dá pra inserir direto no banco
  por enquanto (fora de escopo desta rodada).

## Login por senha removido por completo (22/10, mesmo dia)

Pedido direto do dono do produto, na sequência da implementação acima: **"isso mesmo, derrube
toda aquela autenticação antiga que tinhamos feito e suba apenas com o metodo jwt"**. Diferente
do que a seção anterior registrava (os dois métodos convivendo), o login por email+senha foi
removido por completo do sistema — não é mais um fallback, nem um método alternativo. Hoje a
única forma de entrar é `?token=` na URL (SSO via portal).

**`CONECTA_AUDIENCIA` confirmado e preenchido** no mesmo dia, via token real gerado pelo dono do
produto em `mock.conectacidades.app` (prefeitura de teste "Smartândia") — valor real:
`painel-de-chamada-kappa.vercel.app` (o domínio Vercel do próprio frontend). O endpoint de SSO
deixou de estar 503 por audiência vazia.

**Removido do backend**: `PostLogin`/`corpoLogin` (`internal/controllers/auth.go`), a rota
`POST /api/v1/login`, `auth.HashSenha`/`auth.VerificarSenha` (bcrypt) e a dependência
`golang.org/x/crypto/bcrypt` (demovida pra indireta no `go.mod`), o campo `SenhaHash` de
`domain.Usuario`, e toda leitura/escrita de `senha_hash` em `internal/repository/usuarios.go`
(`NovoUsuario`/`AtualizacaoUsuario`/`CriarUsuario`/`AtualizarUsuario`) e nos controllers que
criam/editam usuário (`admin.go`, `usuarios.go`). A coluna `usuarios.senha_hash` **continua
existindo no banco**, só não é mais lida/escrita por nenhum código — não valia uma migration só
pra isso. Criar um gestor/atendente/recepcionista agora é "pré-provisionar": só cria o registro
(nome, email, papel, vínculo) — a identidade real só se liga a ele no primeiro login SSO (match
por `external_sub`/e-mail verificado, ver seção de segurança acima).

**Removido do frontend**: `Login.tsx` reescrito — sem formulário nenhum. Só 2 estados: tela de
"entrando pelo portal…" (quando chega com `?token=`) e uma tela única de fallback pra qualquer
outro caso (sem token, token inválido, sessão expirada/trocada) — visual de "painel de TV sem
sinal" com a mensagem "Acesse pelo portal da sua prefeitura". `api.login()` removido de
`api.ts`; campos de senha removidos dos formulários de criar/editar usuário
(`telas/gestor/Usuarios.tsx`, `telas/admin/prefeitura/PrefeituraUsuarios.tsx`).

**Verificado ao vivo**: `POST /api/v1/login` → 404 (rota não existe mais);
`POST /api/v1/sessao/portal` com credencial inválida → 401 (SSO continua funcionando
normalmente). `go build`/`go vet`/`gofmt`/`go test ./...` e `tsc --noEmit`/`npm run build`
limpos.

⚠️ **Consequência real, ainda sem solução definida**: o papel `admin` é explicitamente
BLOQUEADO no SSO (ver seção 6, "escalação de privilégio entre tenants") — e agora que o login
por senha também não existe mais, **não existe nenhum caminho de UI pra chegar em `/admin`**.
Isso não foi resolvido nesta rodada — só identificado. Nenhuma mudança de código foi feita em
cima disso até o dono do produto decidir como quer resolver (ex: um mecanismo de acesso de
emergência separado, reconsiderar o bloqueio especificamente pra um caso bem delimitado, ou um
jeito manual via banco mesmo). Sinalizar isso de volta pro dono do produto é o próximo passo,
não uma decisão a tomar sozinho.
