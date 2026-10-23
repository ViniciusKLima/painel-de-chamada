---
name: identidade-visual
description: Use sempre que for mexer em visual do frontend — cor, fonte, espaçamento, cabeçalho, tabela, botão, chip, ou desenhar uma tela nova (ex. futuro admin master). Define a identidade visual do sistema e o porquê de cada escolha — não introduzir cor/fonte/medida nova sem checar aqui primeiro.
---

# Identidade visual do Painel de Chamada

Criada em 21/09 a pedido explícito do dono do produto: o sistema "está com cara de
protótipo ainda, vazio, teste" — pediu pra alguém agir como especialista em
frontend/design/UI-UX, remodelar o visual e documentar numa skill pra não se perder de
novo. Fonte de verdade dos tokens é `frontend/src/index.css` (bloco `:root` no topo) — esta
skill explica as escolhas, não duplica os valores exatos (que podem ser ajustados ali sem
reescrever este documento, desde que a lógica abaixo continue valendo).

## Reestruturação pro design system do Conecta Cidades (16/10)

**A identidade visual inteira foi migrada pra seguir o design system real do Conecta
Cidades** (a empresa por trás da plataforma "Jaboatão mais fácil"/"Jornada", dona do produto
pro qual este painel é construído) — pedido direto do dono do produto depois de uma análise
de um artifact deles (design system do site do cidadão + painel administrativo, em Tailwind/
Bootstrap). **Isso substitui a seção "Princípio geral: fugir de cara de IA" logo abaixo como
critério ativo** — aquela seção documenta a identidade ORIGINAL (21/09), mantida como
histórico porque explica decisões que ainda aparecem em código/commits antigos, mas a fonte,
a cor primária e o raio de borda que ela descrevia não são mais os valores em uso.

**O que mudou, de ponta a ponta**:
- **Fonte**: `Public Sans` + `IBM Plex Mono` → **`Montserrat` sozinha**, inclusive em número
  (Montserrat tem `tabular-nums`, não precisava de uma mono separada) — o design system do
  Conecta usa uma família só, nas duas frentes deles.
- **Cor de ação** (`--cor-primaria`): teal-azulado (`#146c8c`) → **`#d52e5f`**, o valor exato
  usado nos mockups deles ("cada prefeitura tem a sua cor" — mas esse é o canônico).
  Continua sobrescrito em runtime por prefeitura (`tema.ts`), sem mudança nesse mecanismo.
- **Cores semânticas** (sucesso/erro/aviso): a HUE vem do valor exato deles (`#32c787`/
  `#ff6b68`/`#ffc721`), mas os 3 originais falham contraste WCAG AA (< 3:1) usados como texto
  pequeno sobre fundo claro — são claros/vivos demais pra esse uso. Ver "Paleta de cores"
  abaixo pro valor final (mesma família, escurecido até passar ~4.5:1+) e o raciocínio.
- **Raio de borda**: de um valor só (12px cartão) pra uma família de 3 (`--raio-pequeno` 7px,
  `--raio-cartao` 10px, `--raio-modal` 16px) — os números exatos que o painel administrativo
  deles documenta. Pílula (999px) continua permitida em chip/filtro — a proibição deles é só
  pra botão de FORMULÁRIO totalmente arredondado, não pra chip.
- **Gradiente removido**: `.cabecalho` e `.gestor-sidebar` usavam
  `linear-gradient(..., --cor-marca-clara, --cor-marca)` — viraram fundo CHAPADO
  (`--cor-marca` sozinho) — proibição explícita deles ("o produto usa cor chapada"). O token
  `--cor-marca-clara` e a função `clarear()` em `tema.ts` que o calculava foram removidos por
  completo (ficaram órfãos depois da mudança).
- `lang="en"` → `lang="pt-BR"` em `index.html` (proibição deles: "Inglês na interface" —
  isso nunca tinha sido corrigido, achado nessa auditoria).

**O que foi mantido/ficou de fora de propósito, porque o design system deles não cobre**:
- Escala de espaçamento (`--espaco-1..6`) — eles não definem uma, a nossa continua.
- Sombra — eles só proíbem "sombra pesada"; as nossas já eram leves (opacidade 0.04-0.09 na
  maioria), não precisaram de ajuste.
- `--cor-borda`/`--cor-fundo`/`--cor-superficie` (neutros) — "não há cor de fundo, borda ou
  superfície listada" no material deles; mantidos como estavam.
- O mecanismo de tema por prefeitura em runtime (`tema.ts`, `aplicarTemaPrefeitura`) — a
  prefeitura de Jaboatão já tem sua PRÓPRIA cor configurada pelo admin (um azul,
  `#0037c1`/`#cddcfe`, escolhida antes desta migração) — **não foi alterada**, é dado real
  configurado pela UI, não um default de código. O token `#d52e5f` só aparece de fato em
  telas sem prefeitura (Login, Admin) ou numa prefeitura nova que ainda não escolheu cor —
  isso é o comportamento correto do sistema, não uma migração incompleta.
- `default` de `prefeituras.cor_destaque`/`cor_clara` no schema (migration `000009`) foi
  realinhado pro novo valor — só afeta prefeitura CRIADA sem cor explícita daqui pra frente,
  não retroage em nenhuma linha existente.

Ver `frontend/src/index.css` (comentário no topo do `:root`) e `tema.ts` pros detalhes de
implementação; o resto desta skill (abaixo) foi atualizado pontualmente onde citava um valor
que mudou, mas a lógica/estrutura de cada seção (cabeçalho em 2 barras, tabela com
`.tabela-cartao`, hover suave, etc.) continua toda válida — nada disso veio do Conecta, é
convenção nossa e não foi tocado.

## Princípio geral (histórico, 21/09 — ver seção acima pro critério ATUAL)

Pedido explícito na época: "moderno mas que fuja de aparências de IA". Isso significava
evitar especificamente:

- **Inter ou Space Grotesk** como fonte "seguro" — usava-se **Public Sans** (a mesma fonte do
  design system do governo americano, USWDS) + **IBM Plex Mono** pra números. **Superado em
  16/10** — agora é `Montserrat` sozinha, pelo design system do Conecta Cidades (ver seção
  acima). A lição de fundo (não usar a fonte "óbvia" sem pensar) continua válida, só o valor
  concreto mudou.
- **Azul genérico de SaaS** (`#1d4ed8`/`#2563eb` tipo Tailwind/Bootstrap) como cor primária —
  usava-se um teal-azulado institucional. **Superado em 16/10** — agora é `#d52e5f` (rosa/
  magenta), valor canônico do Conecta Cidades, não mais um teal.
- **Cinza puro** em bordas/texto secundário — todos os neutros aqui têm uma leve inclinação
  pra o mesmo matiz do teal (ex. `--cor-borda: #dce3e0`, não `#e5e7eb`), pra parecer
  escolhido, não herdado do browser. **Ainda válido** — o Conecta Cidades não define cor de
  borda/superfície, então esse raciocínio nosso continua de pé.
- Gradiente roxo-azul, fundo creme+serif+terracota, `rounded-lg` em tudo, tudo centralizado —
  nenhum desses padrões aparece aqui. **Ainda válido** (e o gradiente do cabeçalho, que não
  era roxo-azul mas ainda assim era gradiente, foi removido em 16/10 por um motivo
  relacionado mas distinto: proibição explícita do Conecta Cidades, não da lista acima).

Se uma tela nova (ex. o admin master) precisar de uma cor/fonte que não existe ainda, ela
deve nascer como extensão da paleta ATUAL (Montserrat + a família de cor do Conecta Cidades
descrita acima), não uma escolha solta nem um retorno à paleta de 21/09.

## Tipografia

- **Base**: `Montserrat` (pesos 400/500/600/700/800), carregada via Google Fonts em
  `index.html`. Fallback `system-ui`. **Mudou em 16/10** (era `Public Sans`) — valor exato do
  design system do Conecta Cidades, usado nas duas frentes deles (site do cidadão e painel
  administrativo), ver "Reestruturação" acima.
- **Números**: não tem mais uma fonte separada (era `IBM Plex Mono` até 16/10) — Montserrat
  já suporta algarismo tabular. A classe `.numerico` continua existindo, só com
  `font-variant-numeric: tabular-nums` (sem `font-family`). Aplicar em qualquer célula/valor
  que seja essencialmente um número ou código: protocolo, horário, contagem regressiva,
  cooldown, estatística grande do dashboard. **Não** aplicar em nome de pessoa, guichê
  ("Guichê 1" é rótulo, não número solto) ou texto livre.
- Títulos (`h1`/`h2`/`h3`) usam `font-weight: 700` e `letter-spacing: -0.01em` por padrão
  (ver bloco global em `index.css`) — não redefinir peso/espaçamento por tela.

### Escala de tamanho de fonte (16/10)

Pedido direto do dono do produto depois de ver o sistema na cara nova: "achando muito
grande as fontes" — queria variável por tipo de texto pra testar tamanhos sem caçar valor
espalhado pelo CSS. Causa raiz real: `<td>` e `<h1>`/`<h2>` soltos (fora de `.cartao`) nunca
tinham tido um `font-size` explícito — caíam no padrão do navegador (~16px pra célula de
tabela, ~32px/24px pros títulos), bem maior que o resto da interface, que sempre usou
valores pequenos (0.72-0.95rem). Isso sempre existiu, mas ficou mais evidente com Montserrat
(letra mais "cheia"/pesada que a Public Sans anterior no mesmo valor em rem).

Tokens em `:root` (`index.css`), logo depois da escala de espaçamento:

| Token | Valor | Onde |
|---|---|---|
| `--fonte-tamanho-titulo-pagina` | 1.5rem | `h1` solto — "Dashboard", "Usuários internos" |
| `--fonte-tamanho-titulo-secao` | 1.05rem | `h2` — título de bloco/seção, geralmente em cima de uma tabela ("Pendentes", "Sala de espera") |
| `--fonte-tamanho-titulo-cartao` | 0.95rem | `h3` dentro de `.cartao` — bloco de configuração |
| `--fonte-tamanho-corpo` | 0.95rem | texto padrão de botão/input |
| `--fonte-tamanho-tabela` | 0.85rem | conteúdo de `<td>` |
| `--fonte-tamanho-tabela-cabecalho` | 0.72rem | `<th>` (cabeçalho de coluna, caixa alta) |

**Pra testar um tamanho novo, mexer só nesses 6 valores** — nunca num seletor específico
(`.cartao h2`, por exemplo, foi removido de propósito porque só duplicava
`--fonte-tamanho-titulo-secao`). Qualquer elemento novo que precise de um tamanho de texto
que já existe numa dessas 6 categorias deve usar o token, não um `rem` solto; só criar um
token sétimo se o novo caso for genuinamente um tipo de texto diferente dos 6 (ex. não força
"tudo cabe em título/corpo/tabela" se não couber de verdade).

**Fora dessa escala, de propósito**: o painel de TV (`telas/painel-tv`) usa `clamp()`
responsivo ao viewport, não esses tokens — é uma tela pra ler de longe numa TV física, o
tamanho precisa escalar com o tamanho da tela, não ser um valor fixo em rem.

## Paleta de cores

Todas as cores vivem como variável CSS em `:root` (`frontend/src/index.css`). Nunca usar um
hex solto num componente — se a cor que você precisa não existe como token, é sinal de que
falta um token, não que vale a pena um valor local.

**Atualizada em 16/10** pra seguir o design system do Conecta Cidades (ver "Reestruturação"
acima) — a cor de ação e as 3 semânticas de status mudaram de família (de teal/verde-floresta/
vermelho-tijolo/âmbar pra magenta/verde/coral/amarelo-terroso); os neutros (fundo/superfície/
borda) não mudaram, porque o Conecta Cidades não define esses tokens.

| Token | Uso | Por quê |
|---|---|---|
| `--cor-fundo` | fundo da página | Off-white quente — não definido pelo Conecta Cidades, mantido do original |
| `--cor-superficie` | fundo de cartão/modal | Branco puro |
| `--cor-texto` / `--cor-texto-suave` | texto principal/secundário | `#444444` — valor exato do Conecta Cidades (`--cor-texto-suave` é derivado, eles não definem um secundário) |
| `--cor-borda` | bordas, divisórias | Cinza com leve matiz — não definido pelo Conecta Cidades, mantido do original |
| `--cor-marca` | **só** o fundo do cabeçalho e da sidebar do gestor | Default `#0e2e38`, mas **segue a cor de destaque de cada prefeitura em runtime** (ver seção seguinte) — fundo **chapado** desde 16/10 (era degradê com `--cor-marca-clara`, token removido). Não usar em botão de conteúdo |
| `--cor-marca-texto` / `--cor-marca-texto-suave` | texto sobre o cabeçalho | Quase-branco com matiz teal, fixo (não acompanha a prefeitura) |
| `--cor-primaria` / `--cor-primaria-forte` | botões, links, foco, aba ativa | `#d52e5f` — valor exato do Conecta Cidades (16/10, era um teal-azulado) — a cor de "ação" do sistema, por prefeitura (mesma cor de destaque que alimenta `--cor-marca`) |
| `--cor-sucesso` / `--cor-sucesso-fundo` | chip "no horário", ícone de atendido, notificação de sucesso | Hue do Conecta Cidades (`#32c787`), mas o **token principal é uma versão escurecida** (`#218158`) — o valor exato deles falha contraste WCAG AA como texto pequeno (~2.2:1); o "fundo" claro continua usando o tom exato deles em 12% de alpha, então a cor real da marca aparece em todo chip, só o texto escureceu pra ficar legível |
| `--cor-erro` / `--cor-erro-fundo` | chip "faltou", notificação de erro | Mesmo raciocínio: hue do Conecta Cidades (`#ff6b68`), token principal escurecido (`#b3433f`, ~5.5:1) |
| `--cor-aviso` / `--cor-aviso-fundo` | chip "atrasado", linha atrasada na recepção | Mesmo raciocínio: hue do Conecta Cidades (`#ffc721`), token principal escurecido bem mais (`#946200`, ~5.2:1) — amarelo puro precisa escurecer MUITO pra virar texto legível, por isso parece mais "âmbar/marrom" que "amarelo" |
| `--cor-info` / `--cor-info-fundo` | `.aviso` (caixa informativa neutra), cabeçalho de tabela, hover de linha | Mesmo tom do primária (`#d52e5f`), fundo em 12% alpha |
| `--cor-encaixe` / `--cor-encaixe-fundo` | chip "encaixe" | Roxo (`#5b3aa6`) — cor nossa, o Conecta Cidades não define uma pra esse caso; só esse chip usa roxo no sistema inteiro |

Regra importante: **`--cor-aviso` (âmbar, "atrasado") e `.aviso` (caixa informativa neutra)
não são a mesma coisa** — os dois já usaram o mesmo amarelo numa versão anterior e ficava
ambíguo se uma caixa amarela era "informação" ou "atenção". Agora `.aviso` usa a cor de ação
(info) e só chip/linha de "atrasado" usa âmbar.

**Se precisar do valor EXATO de marca do Conecta Cidades** (não o token escurecido) — por
exemplo pra um ícone grande ou um fill onde não tem texto por cima e contraste não é
problema — use o hex literal documentado nos comentários do `:root` em `index.css`, não
`--cor-sucesso`/`--cor-erro`/`--cor-aviso` (que são a versão calibrada pra texto). Não existe
token separado pra isso hoje porque nenhum lugar do sistema precisou dele ainda — se
precisar, é um token novo a criar, documentado aqui, não um hex solto no componente.

### Raio de borda

Família de 3 tamanhos (16/10, substituindo um valor único de 12px), valores exatos do design
system do Conecta Cidades (família "painel administrativo", a mais parecida com o nosso
produto — telas internas de staff, não público):

| Token | Valor | Uso |
|---|---|---|
| `--raio-pequeno` | 7px | botão, input, select, ícone pequeno |
| `--raio-cartao` | 10px | `.cartao`, `.tabela-cartao`, tile/linha clicável, cabeçalho de tabela (`th`) |
| `--raio-modal` | 16px | modal/overlay de verdade (`.modal-cartao`, `.modal-selecao-guiche`) |

Pílula (`border-radius: 999px`, usada direto, sem token) continua permitida em chip, badge,
etiqueta, pill de guichê, barra de progresso — a proibição do Conecta Cidades é só pra
**botão de formulário** totalmente arredondado, documentada explicitamente por eles como uma
exceção ("filtros são pills, chips podem ser pill — a regra é sobre botão de formulário").
Círculo (`border-radius: 50%`, ícone circular, bolinha de cor) também não entra nessa família,
não é coberto pela regra deles.

**Painel de TV usa `--raio-modal` (16px) nos 3 cartões grandes** (`.chamada-atual`,
`.historico`, `.aguardando-bloco`), não `--raio-cartao` — o painel de TV é a tela PÚBLICA do
sistema, equivalente ao "site do cidadão" do Conecta Cidades (que usa raio maior, 8px base/
16px modal), não ao "painel administrativo" interno (7px/10px) que todas as outras telas
seguem. O cabeçalho interno da tabela do painel (`.historico-cabecalho`) continua com 8px
fixo (não um token) — está dentro do padding do cartão, não precisa casar com o raio externo.

## Botão de ícone em tabela — discreto, sem fundo sólido (16/10)

Reclamação direta: "esses botões gigantes dessas tabelas... não precisa do background, tá
feio assim, faça mais discretos". `.botao-icone` (gear de configurações, lixeira de excluir,
reativar, copiar link) herdava o `button { background: var(--cor-primaria); color: #fff }`
global — virava um quadrado de 38px sólido na cor de ação, chamativo demais numa coluna de
ação dentro de uma linha de tabela. Agora é **ghost**: 32px, sem fundo, sem borda, só o ícone
na cor (`--cor-texto-suave` por padrão, `--cor-sucesso`/`--cor-erro`/`--cor-primaria` nas
variantes), com um tingimento bem leve só no hover (`--cor-hover` na base, o "-fundo" da cor
semântica nas variantes) — mesmo princípio de "indicar clicável sem chamar atenção sozinho"
já usado em `.linha-guiche-resumo`/`.quadro-guiche`. `.botao-icone-secundario` perdeu a borda
que a distinguia das outras — não fazia mais sentido já que nenhuma variante tem fundo sólido
agora, a cor do ícone já basta pra diferenciar. Lembrar do `:not(:disabled)` em cada variante
nova (armadilha de especificidade já documentada acima) — `.botao-icone-sucesso`/`-perigo`/
`-secundario` todas precisam repetir isso no seletor de hover.

## Cabeçalho mostra o guichê resolvido, ao lado do nome (16/10)

`<Cabecalho>` ganhou um prop opcional `guiche?: string | null` — quando preenchido, aparece
concatenado no `subcabecalho-usuario`: `"ATENDENTE: NOME - GUICHÊ X"`. Só a tela do atendente
passa esse prop hoje (`SalaDeEspera.tsx`, computado juntando os nomes de guichê já resolvidos
em `guicheSelecionadoPorGrade` — um atendente com mais de uma grade pode ter mais de um
guichê ao mesmo tempo, aí os nomes vêm separados por vírgula; grade sem guichê escolhido
ainda não entra na lista). Recepção/gestor/admin não têm guichê — continuam passando o
`Cabecalho` sem esse prop, que é opcional de propósito.

## Blocos de grade sem moldura (16/10)

`.grupo-grade` (cada bloco de grade dentro de "Sala de espera", agrupado por unidade) tinha
border+padding+fundo (`--cor-fundo`) em volta — reclamação direta: "não quero essa borda e
esse espaçamento... deixe com a largura normal". Removido por completo — o bloco agora é só
`display:flex; flex-direction:column; gap` (espaço vertical entre o cabeçalho da grade e a
tabela dela), direto sobre o fundo da página, mesmo padrão de "bloco sem cartão, só a tabela
tem `.tabela-cartao`" já usado no resto da tela do atendente (ver seção "Tabelas" acima). Não
reintroduzir uma moldura aqui achando que "falta separação visual" — o espaçamento vertical
(`--espaco-5` em `.grupos-unidade`, `--espaco-3` em `.grupo-unidade`) já é o que separa um
bloco do outro.

## Cores configuráveis pelo admin, por prefeitura (22/09, ampliado 23/09)

O admin escolhe 2 cores por prefeitura, na tela "Configurações da Prefeitura"
(`/admin/prefeituras/{id}`): a **de destaque** (ações, links, aba ativa) e a **mais clara**
(fundo, cabeçalho de tabela) — nas palavras do dono do produto. Elas mapeiam DIRETO pros
tokens que já existiam: destaque → `--cor-primaria` (+ `--cor-primaria-forte`, calculada
escurecendo a cor em ~18%, pro hover dos botões), clara → `--cor-info`/`--cor-info-fundo`.

**Desde 23/09, a cor de destaque também alimenta `--cor-marca`** (fundo do `<Cabecalho>` e da
sidebar do gestor) — decisão revertida da original: antes era uma identidade fixa, mais
escura, separada de propósito, porque só existia uma prefeitura no sistema; agora que o
admin cadastra várias, cada uma com sua própria paleta, não fazia sentido todo mundo ver o
mesmo teal-marinho fixo independente da prefeitura. `--cor-marca` = a própria cor de
destaque, usada como fundo CHAPADO (sem variante clareada — `--cor-marca-clara` e a função
`clarear()` que a calculava foram removidos em 16/10 junto com o degradê, ver
"Reestruturação" no topo da skill). `--cor-marca-texto` continua fixo (quase-branco), então
cores de destaque muito claras podem ter pouco contraste no cabeçalho — risco aceito, mesmo
problema que já existe em botões com `--cor-primaria` muito clara.

Aplicadas em **runtime**, não build-time (`frontend/src/tema.ts`,
`aplicarTemaPrefeitura(corDestaque, corClara)`) — seta as CSS custom properties direto no
`document.documentElement`, chamado logo depois de resolver a sessão (`api.sessao()`/
`api.login()`) em cada tela autenticada (`GestorLayout`, `Recepcao`, `SalaDeEspera`). O admin
não tem prefeitura própria, então suas telas ficam sempre na paleta padrão do sistema. O
painel de TV (`telas/painel-tv`) usa seus próprios tokens `--painel-*`, intencionalmente
isolados (ver seção "Painel de TV — tema à parte" abaixo) — não aplicar o tema de prefeitura
lá.

## Espaçamento

Escala fixa em `:root`: `--espaco-1` (4px) até `--espaco-6` (32px). Usar essas variáveis em
`padding`/`gap`/`margin` de qualquer componente novo em vez de números soltos tipo `14px`, pra
manter o ritmo visual consistente entre telas.

## Cabeçalho (`componentes/Cabecalho.tsx`)

Estrutura fixa em 3 blocos, replicando o desenho que o dono do produto anexou na raiz do
projeto (`layout-recepção.png`/`layout atendente.png`) — **não inventar variação por tela**:

1. **Esquerda — marca, sempre igual em toda tela autenticada**: nome do produto
   ("Painel de Chamada", `NOME_PRODUTO`) em destaque, nome da prefeitura
   ("Prefeitura de Jaboatão dos Guararapes", `NOME_PREFEITURA`) embaixo, mais discreto.
   **Fixo no componente, não é prop** — é identidade visual, não dado de sessão. Se um dia o
   sistema atender mais de uma prefeitura, aí sim vira prop.
2. **Meio — contexto de trabalho**: nome da unidade em destaque, embaixo
   `"{PAPEL EM CAIXA ALTA}: {nome de quem está logado}"` (ex. "RECEPÇÃO: MARIA SILVA",
   "ATENDENTE: JOÃO PEREIRA") — é assim que o papel do usuário aparece agora, não mais como
   rótulo separado do lado esquerdo (isso mudou na reforma: antes o nome do usuário ficava à
   esquerda, o desenho do dono do produto deixa claro que é o contexto central).
3. **Direita — ações da tela**: seletor/indicador de guichê (atendente) e "Sair" por último.
   O indicador de guichê **é** o botão de trocar (22/09) — clicar nele reabre o modal de
   seleção. Não voltar a separar isso em dois elementos ("Guichê 2" + um botão "Trocar" ao
   lado) — foi assim antes e o dono do produto pediu pra fundir.

**Fundo sempre sólido, nunca transparente** — pedido explícito e repetido do dono do produto,
e desde 16/10 também proibição explícita do design system do Conecta Cidades ("o produto usa
cor chapada", ver "Reestruturação" no topo). Usa `--cor-marca` CHAPADO — nunca gradiente
(era `linear-gradient(..., --cor-marca-clara, --cor-marca)` até 16/10), nunca uma cor de
conteúdo (`--cor-primaria`) nem `transparent`/`rgba` com alpha baixo no fundo do cabeçalho em
si (os botões *dentro* do cabeçalho é que usam `rgba(255,255,255,...)` pra contrastar sobre o
fundo sólido).

**Sempre 100% da largura da viewport, não só até a borda do conteúdo** (22/09, correção de
uma primeira tentativa que só sangrava até a borda de `.pagina`): `<Cabecalho>` é renderizado
como **irmão** de `.pagina`, nunca como filho dela — ela tem `max-width` e fica centralizada,
então qualquer cabeçalho dentro dela herdaria essa largura limitada. Todo componente de tela
segue o padrão:
```tsx
return (
  <>
    <Cabecalho ... />
    <div className="pagina"> ... conteúdo da tela ... </div>
  </>
);
```

## Tabelas

Identidade visual própria (pedido: "quero mais identidade nas tabelas"), aplicada
globalmente via seletor de elemento (`table`/`th`/`td`), não precisa de classe extra:

- Cabeçalho da tabela (`th`) com fundo tintado (`--cor-info-fundo`) e texto na cor de marca,
  caixa alta, letter-spacing — nunca cabeçalho branco liso.
- Linhas pares com leve tint (zebra sutil) — ajuda a ler listas longas (recepção passa de 130
  linhas facilmente).
- Linha destaca ao passar o mouse (`tbody tr:hover`) com `--cor-hover` — **não** reaproveitar
  `--cor-info-fundo` pra isso (era o bug original: hover "muito forte", reclamação direta do
  dono do produto — `--cor-info-fundo` é pra cabeçalho de tabela/caixa informativa, não pra
  hover). `--cor-hover` é de propósito quase imperceptível (~4% de opacidade) — o objetivo é
  só indicar "você está sobre isso", nunca chamar atenção. Mesma regra vale pra qualquer
  outro hover de "linha/item selecionável" no sistema (`.quadro-guiche`,
  `.linha-guiche-resumo` etc.) — hover de **botão de verdade** (que dispara uma ação) pode
  continuar mais perceptível, a distinção é "hover de local" vs. "hover de botão".
  **Ver a armadilha de especificidade de CSS logo abaixo** — um hover de linha/item que é
  tecnicamente um `<button>` (caso comum: linha clicável que abre algo) precisa do mesmo
  cuidado ali descrito, senão o hover suave nunca aparece de verdade.
- Célula de número (protocolo, horário) leva `.numerico`.
- Coluna de ação (`.coluna-acao`) é sempre ícone puro, nunca texto+ícone dentro de uma
  tabela — texto+ícone só em botões fora de tabela (ex. "Chamar próximo", "Adicionar").
- **Blocos de tabela não precisam de cartão em volta, mas a tabela em si é branca** (22/09,
  telas do atendente, ajustado no mesmo dia depois de testar ao vivo): título solto (`<h2>`)
  direto sobre o fundo da página — isso continua — mas a **tabela** ganhou de volta uma
  superfície própria via `.tabela-cartao` (fundo `--cor-superficie`, borda, raio, sombra
  leve), porque sem nenhum fundo ela ficava "flutuando" sem estrutura nenhuma. A distinção
  que importa: **o título não tem moldura, a tabela tem** — não confundir "sem cartão" com
  "sem fundo branco". Quando a seção também precisa de uma ação (ex. "Chamar próximo"), ela
  fica ao lado do título numa `.secao-cabecalho` (`display:flex; justify-content:
  space-between`), fora de `.tabela-cartao` e nunca dentro do `<th>` da tabela.
- **Regra vale pra QUALQUER tela com busca/abas/tabela, não só o atendente** — a Recepção
  (`Recepcao.tsx`) ainda empacotava busca+abas+tabela inteiras dentro de um `<section
  className="cartao">` único até 20/10 (nunca tinha sido migrada junto com o atendente em
  21/09, ficou inconsistente sem ninguém notar até o dono do produto apontar). Corrigido:
  virou `<section className="bloco-recepcao">` (sem `.cartao`, só um `margin-bottom` pro
  respiro que o cartão dava) com busca/abas soltos e a tabela em `.tabela-cartao` do mesmo
  jeito. **Checklist pra tela nova de lista**: nunca um `<section className="cartao">`
  envolvendo busca+abas+tabela juntos — só `.tabela-cartao` na tabela, o resto solto.

## Armadilha de especificidade: hover custom num `<button>`

Bug real, encontrado 3x em rodadas diferentes antes de virar regra (22/09, `.linha-guiche-
resumo`, `.botao-voltar`, `.aba`, `.quadro-guiche` — todos afetados ao mesmo tempo, só
descobertos quando o dono do produto pediu pra revisar hover geral): o CSS global já tem
```css
button:not(:disabled):hover { background: var(--cor-primaria-forte); }
```
que é **elemento + duas pseudo-classes**. Qualquer `<button>` customizado que tente sobrescrever
o hover só com `.minha-classe:hover { background: outra-coisa; }` **perde** — uma classe +
uma pseudo-classe é menos específico. O resultado visual é sempre o mesmo sintoma: o elemento
"pisca" um fundo teal escuro sólido (`--cor-primaria-forte`) no hover, ignorando por completo a
cor pretendida (`--cor-hover`, `none`, o que for) — e isso NÃO aparece revisando o código, só
passando o mouse de verdade em cima.

**Regra fixa**: todo `:hover` customizado escrito para um elemento `<button>` (nunca `<div>`/
`<a>`, que não têm essa regra global concorrendo) precisa repetir `:not(:disabled)` no próprio
seletor pra igualar a especificidade — `.minha-classe:not(:disabled):hover { ... }`, nunca só
`.minha-classe:hover`. Ver `.acesso-secundario:not(:disabled):hover` (Login) como o padrão já
certo desde antes; `.botao-voltar`, `.linha-guiche-resumo`, `.aba` e `.quadro-guiche` foram
corrigidos pra esse padrão em 22/09. Ao criar qualquer botão novo com hover próprio, checar
isso ANTES de considerar o hover pronto, testando ao vivo (não só lendo o CSS).

**Correção adicional (23/09), meio-passo que faltava**: igualar a especificidade só resolve
propriedades que a classe customizada **realmente declara**. `.aba:not(:disabled):hover` tinha
a especificidade certa mas só declarava `color` — `background` continuou vindo do
`button:not(:disabled):hover` global (que não tem concorrente pra essa propriedade
específica), então o fundo colorido (`--cor-primaria-forte`, hoje a cor de destaque de cada
prefeitura) continuava aparecendo por baixo, com texto escuro. **Sempre declarar TODAS as
propriedades que devem ficar diferentes do padrão do botão global** (tipicamente `background`
e `color` juntos), não só a que estava visivelmente errada — meia correção ainda deixa a
outra propriedade vazando do seletor global.

## Texto sem função — não descrever o óbvio

Reclamação recorrente do dono do produto, generalizada em 22/09: caixas `.aviso` (fundo azul)
explicando o que uma tela ou seção já deixa claro pelo título, pelas colunas da tabela ou pela
própria interação (ex.: um `<h1>Usuários internos</h1>` seguido de "Nome, cargo e status de
acesso de quem trabalha nesta unidade" — a tabela logo abaixo já mostra exatamente isso; ou
"Clique numa linha pra abrir e personalizar" acima de uma lista de linhas com engrenagem, que
já é um afinidade de clique padrão no resto do sistema). Essas foram todas removidas.

**Regra**: antes de escrever uma frase de apoio abaixo de um título ou seção, perguntar "isso
já não fica óbvio pelo título, pelas colunas ou pela ação em si?" — se sim, não escrever nada.
Duas categorias que continuam válidas, com tratamento visual diferente:

- **`.aviso`** (caixa com fundo `--cor-info-fundo`) — só pra informação **dinâmica e
  funcional**: resultado de uma ação (ex. lista de unidades/grades novas criadas por um
  upload), ou o motivo de um bloqueio em tempo real (ex. `motivoBloqueio` no atendente
  explicando por que "Chamar próximo" está desabilitado agora). Se o texto é sempre igual
  independente do estado da tela, não é isso.
- **`.dica`** (texto discreto, sem fundo, `--cor-texto-suave`) — pra esclarecer um termo ou
  regra de negócio genuinamente ambíguo (ex. "Tolerância de atendimento" explicando que o
  tempo mínimo entre chamadas é uma fatia do prazo, não somado — sem essa frase o gestor
  provavelmente entenderia errado) ou mostrar um dado secundário de identificação (ex.
  "Serviço: X" abaixo do nome de uma grade). Nunca usar `.dica` pra restated o título.

## Título de bloco de configuração fica dentro do cartão

Todo bloco de configuração/formulário (SLA, painel, guichês) segue o mesmo padrão: `<h3>`
como primeiro filho de `<section className="cartao">`, nunca um `<h2>`/título solto acima do
cartão com o cartão começando só no conteúdo. Bug corrigido em 22/09: "Guichês e salas" tinha
nascido como um `<h2>` fora do cartão (padrão diferente dos outros 3 blocos da mesma tela —
SLA da recepção, SLA do atendimento, Configurações do painel, todos com `<h3>` dentro),
deixando a tela inconsistente. Essa regra é só pra blocos de **configuração** — não confundir
com a regra de "Blocos de tabela" acima, onde o título de uma lista/tabela fica **de
propósito** solto fora do cartão (`.secao-cabecalho` + `.tabela-cartao` só na tabela); são
dois padrões diferentes pra dois tipos de bloco diferentes, cada um consistente dentro do seu
próprio tipo.

## Bloco compartilhado vs. bloco pessoal (atendente)

Distinção que apareceu na prática (22/09) e vale a pena guardar: **um bloco que mostra o
mesmo dado pra todo mundo da unidade** (ex. "Sala de espera", "Pendentes"/"Chamando agora")
é diferente de **um bloco pessoal, filtrado por quem está logado** (ex. "Meus atendimentos
hoje"). A aba "Ausentes" nasceu dentro do bloco pessoal e foi movida pro bloco compartilhado
("Sala de espera") justamente porque semanticamente ela é sobre a unidade inteira, não sobre
quem chamou — o dono do produto notou isso e pediu a mudança explicitamente ("já que isso
vai ser uma tela compartilhada"). Ao desenhar uma aba/filtro novo, primeiro perguntar "isso é
sobre MIM ou sobre a UNIDADE" e colocar no bloco certo — não assumir que uma informação é
pessoal só porque nasceu dentro de uma tela de atendente.

## Busca

Todo bloco com uma lista que pode crescer (mais de ~10 itens) ganha uma busca por nome/
protocolo (ou outro campo relevante — "Grade de horário" filtra por serviço, 16/10),
reaproveitando `.barra-busca`/`.campo-busca` (mesmo padrão da recepção) — mesmo que não tenha
um botão de ação ao lado (nesse caso, `.campo-busca` sozinho dentro de `.barra-busca`, que
ainda cuida do espaçamento). Normalizar a busca (minúsculas, sem pontuação) com
`normalizarBusca`, extraída em 16/10 pra `frontend/src/busca.ts` (era copiada igual em
`Recepcao.tsx`/`SalaDeEspera.tsx`, virou uma terceira cópia em `Grades.tsx` — na hora certa
de extrair, como a versão anterior desta nota já previa) — importar de lá, nunca redeclarar
local de novo.

## Botões e estados compostos (ícone + informação extra)

Erro corrigido nesta reforma: o timer de cooldown do "Rechamar" no atendente era um ícone
desabilitado com uma legenda minúscula empilhada **embaixo** dele — ficava desalinhado e
"feio" (palavra do dono do produto). **Regra daqui pra frente**: quando um botão de ícone
precisa mostrar uma informação textual extra (contagem, valor, unidade), ele vira um botão-
pílula única com ícone + texto **lado a lado**, na mesma altura dos outros botões da mesma
linha (ver `.botao-cooldown` em `index.css`, usado em `SalaDeEspera.tsx`) — nunca ícone e
texto em linhas separadas dentro da coluna de uma tabela.

## O que decidir por conta própria vs. o que perguntar

- Escolher um tom exato dentro de uma família já definida (ex. "que verde usar pro terceiro
  ícone de status") — decidir e documentar aqui se for um token novo.
- Precisar de uma cor/família tipográfica **fora** dessa paleta pra uma tela genuinamente
  diferente (ex. um painel de admin master com outro público) — aí sim vale conversar antes,
  em vez de estender a paleta silenciosamente.

## Scroll e largura estável

`html` tem `overflow-y: scroll` + `scrollbar-gutter: stable` (em `index.css`) — de propósito
os dois juntos, não só um. Achado ao vivo (22/09): `scrollbar-gutter: stable` sozinho no
elemento raiz não reservou o espaço de forma confiável neste navegador/versão testada (o
gutter colapsava de volta a zero assim que o conteúdo cabia sem rolar). `overflow-y: scroll`
é a técnica antiga, mas garante o espaço reservado sempre, em qualquer navegador — por isso
os dois ficam juntos: o clássico como garantia, o moderno como reforço onde funcionar. Sem
isso, alternar entre uma aba/tela com lista longa (precisa rolar) e uma com lista curta
(não precisa) fazia o conteúdo centralizado "pular" horizontalmente uns pixels — reclamação
direta do dono do produto ("fica normal - empurrado - normal - empurrado"). Não remover
nenhum dos dois sem testar de verdade trocando entre uma tela comprida e uma curta.

## Painel de TV — tema à parte, mas agora também com a cor da prefeitura (22/09)

O painel de TV (`telas/painel-tv`) mantém seu próprio conjunto de tokens (`--painel-*`),
num namespace CSS isolado dos tokens operacionais (`--cor-*`) — é uma tela pública, vista de
longe, com necessidades de contraste e legibilidade diferentes das telas operacionais, e
não roda `aplicarTemaPrefeitura` (que depende de sessão autenticada). Não forçar o cabeçalho
de marca operacional nele.

**Isso não significa mais "sem cor nenhuma da prefeitura"** — pedido explícito do dono do
produto (22/09, rodada do redesenho do painel): o **fundo** da tela usa a cor clara/
secundária configurada pelo admin (`--painel-fundo`), e o **acento** (protocolo em destaque,
barra de contagem regressiva, cabeçalho da tabela, pill de guichê) usa a cor de destaque
(`--painel-acento`) — só que via uma função própria, `aplicarTemaPainel` (`tema.ts`),
separada de `aplicarTemaPrefeitura`. O backend resolve as cores no próprio endpoint público
`GET /painel/{gradeId}` (via unidade→secretaria→prefeitura, já que não há sessão pra
reaproveitar) e devolve `corDestaque`/`corClara` na resposta. Os dois blocos de conteúdo
(chamada atual + tabela de últimas chamadas) continuam **brancos** por cima dessa cor de
fundo, pra manter contraste e legibilidade a distância — o padrão geral do projeto virou
"fundo tintado da cor clara da prefeitura, conteúdo em cartão branco por cima", igual às
outras telas, só que aplicado à cor secundária em vez da cor de destaque.

Tons derivados desse acento (ex. o fundo do cabeçalho da tabela, o pill de guichê) usam
`color-mix(in srgb, var(--painel-acento) 12%, white)` calculado em CSS puro — nunca
referenciar um token operacional como `--cor-info-fundo` aqui, porque essa tela nunca roda
`aplicarTemaPrefeitura` e esse token nunca seria atualizado nela.

A tipografia (`Montserrat`, ver "Reestruturação" no topo) continua unificada com o resto do
site, como já estava.

## Responsividade mobile — regras pra não regredir (23/09)

Auditoria completa pedida pelo dono do produto ("muitas coisas quebram, dão espaço extra no
scroll lateral em mobile"). Causa raiz em quase todos os casos reais encontrados: um elemento
com largura intrínseca (min-content) maior que o espaço disponível, sem nenhum container que
soubesse conter/rolar esse excesso. Regras que saem dessa investigação, pra qualquer tela
nova não reintroduzir o mesmo problema:

- **Todo `<table>` fica dentro de `.tabela-cartao`.** Essa classe usa `overflow-x: auto` (não
  `hidden` — era `hidden` até essa rodada, e uma tabela mais larga que o cartão simplesmente
  CORTAVA as últimas colunas, geralmente a de ação com os ícones, sem scroll nenhum e sem
  aviso visual nenhum — bug real, achado numa tela de admin onde os botões de editar ficavam
  100% invisíveis e inclicáveis em mobile). Uma tabela sem esse wrapper (achado real: a
  tabela da Recepção era a única do sistema sem ele) estoura a largura da PÁGINA inteira em
  vez de só rolar dentro do próprio cartão — é essa a causa mais provável de "scroll lateral
  extra" se aparecer de novo.
- **Todo `input`/`select`/`button` tem `max-width: 100%; min-width: 0` global** (regra base em
  `index.css`). Motivo: um `<select>` nativo se auto-dimensiona pelo texto da opção MAIS
  LARGA — com grades reais importadas da planilha (nomes tipo "CRAS - Jardim Jordão/
  Guararapes (ENCAIXE)"), um `<select>` de grade sem esse limite estourava qualquer modal ou
  formulário estreito. Não remover esse `min-width: 0` achando redundante — sem ele, o campo
  não encolhe abaixo do próprio conteúdo dentro de um flexbox, mesmo com `flex: 1` no pai.
- **Numa barra `display:flex` com um campo de busca + botão** (`.barra-busca`/`.campo-busca`),
  o botão precisa de `flex-shrink: 0` e o campo precisa de `min-width: 0` explícito — sem os
  dois, é o BOTÃO que acaba saindo da tela, não o campo de texto (o campo tem menos conteúdo
  intrínseco que o botão, mas o botão não tinha `flex-shrink:0` garantindo que ele nunca
  cede espaço).
- **`html` tem `overflow-x: hidden`** (par do `overflow-y: scroll` já documentado acima) —
  rede de segurança, não solução. Se precisar disso pra esconder um scroll lateral que
  apareceu, é sinal de que o causador raiz ainda não foi achado — investigar o elemento
  específico antes de contar só com esse hidden.
- **Nunca usar CSS Grid com colunas mistas tipo `1fr auto 1fr`** pra um cabeçalho ou barra que
  precisa caber em qualquer largura — a coluna `auto` cresce pelo `min-content` do conteúdo
  sem limite superior, e MUITO conteúdo (nome completo + papel + nome da unidade) força a
  linha inteira a ficar mais larga que a tela, ou obriga um `@media` empilhando em coluna (o
  padrão antigo do `.cabecalho`, removido nessa rodada por pedido explícito do dono do
  produto: "não gosto de header em coluna, quero em linha ainda"). Ver a seção seguinte.

## Cabeçalho: 2 barras, nunca 3 blocos numa linha só (23/09)

O `.cabecalho` (recepção/atendente) e o equivalente do gestor/admin (sidebar) **não devem
virar coluna em mobile, nunca** — decisão explícita do dono do produto. Se o conteúdo não
cabe numa única barra com 2 blocos (esquerda/direita), a solução é **dividir em uma barra
nova**, nunca empilhar em coluna nem deixar a `auto` de um grid crescer sem limite. Foi
exatamente isso que resolveu o cabeçalho de recepção/atendente:

- **Barra principal** (`<header className="cabecalho">`, cor de marca): só 2 blocos — marca
  (produto + prefeitura) à esquerda, ações da tela à direita (`justify-content: space-
  between`, nunca grid de 3 colunas).
- **Subcabeçalho** (`<div className="subcabecalho">`, fundo branco, logo abaixo): só 2
  blocos — quem está logado (papel + nome) à esquerda, nome da unidade à direita.

Cada bloco de texto tem `min-width: 0` + `overflow: hidden; text-overflow: ellipsis;
white-space: nowrap` — um nome de unidade ou de usuário muito comprido trunca com reticências
em vez de forçar a barra a crescer. Isso é o que garante a linha única em qualquer largura,
não um breakpoint reorganizando o layout. Um `@media (max-width: 480px)` só aperta
padding/font-size como ajuste fino — nunca muda `display`/`flex-direction`.

Se um dia precisar adicionar um 3º bloco de informação num cabeçalho (mais um dado além do
que já existe), a resposta é uma 3ª barra, não uma 3ª coluna na mesma linha.

## Conteúdo do gestor/admin centralizado em telas largas (23/09)

`.pagina-gestor` (o wrapper de conteúdo dentro de `.gestor-conteudo`, usado por Dashboard,
Grades, Usuarios etc.) tem `max-width: 1180px` desde a rodada 18, mas faltava `margin: 0
auto` — sem isso, em qualquer monitor mais largo que ~1180px + o padding da sidebar, o
conteúdo ficava sempre colado na esquerda com um vão vazio grande à direita. `margin: 0 auto`
resolve isso da forma padrão (centraliza um bloco com `max-width` menor que o container) —
qualquer wrapper novo de conteúdo com `max-width` fixo deve seguir o mesmo padrão, a menos que
exista um motivo explícito pra grudar numa borda.
