---
name: auth-conecta-cidades
description: Referência do fluxo real de SSO (JWT via URL) da plataforma Conecta Cidades — ADIADO PRO PÓS-MVP (decisão de 19/09, ver skill `auth-login-mvp` pro que o piloto usa de verdade). Consultar só quando essa integração for retomada.
---

# Autenticação real — SSO da plataforma Conecta Cidades (pós-MVP)

⚠️ **Adiado pro pós-MVP (decisão de 19/09)**: "essas questões do JWT ainda serão discutidas, mas
pro MVP não vamos usar, vamos passar com login mesmo". O piloto usa login simples (email+senha) —
ver skill `auth-login-mvp`. Este documento continua valendo como pesquisa/desenho de referência
pra quando a integração real com o SSO da prefeitura for retomada, mas **não implementar nada
daqui durante o MVP**.

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

## 6. Mapeamento de papel — ⚠️ ainda não confirmado com a plataforma pro nosso serviço

O produto irmão reconhece só os papéis do domínio dele (`admin`, `contract_admin`, `manager`,
`operator`) e trata qualquer outro papel da "plataforma mãe" — inclusive, no texto original,
**`attendant`, `receptionist`** — como "não concede nada" (porque não são papéis daquele
produto). Isso é evidência forte de que a plataforma usa `attendant`/`receptionist` como nomes
de papel globais, prováveis candidatos pro nosso mapeamento:

| `roles` (JWT) | Papel no nosso painel | Confiança |
| --- | --- | --- |
| `attendant` | `atendente` | alta — nome bate literalmente |
| `receptionist` | `recepcionista` | alta — nome bate literalmente |
| `manager` | `gestor` | média — é o papel "gerencial" genérico da plataforma, mas seu significado exato varia por serviço (no produto irmão, `manager` faz outra coisa) |

**Confirmar isso com a equipe da plataforma antes de travar como regra definitiva** — mesmo
espírito da nota já existente na skill `papeis-e-telas`.

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
