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

## Princípio geral: fugir de "cara de IA"

Pedido explícito: "moderno mas que fuja de aparências de IA". Isso significa evitar
especificamente:

- **Inter ou Space Grotesk** como fonte "seguro" — usamos **Public Sans** (a mesma fonte do
  design system do governo americano, USWDS — combina tematicamente com "sistema de uma
  prefeitura" sem ser óbvio) + **IBM Plex Mono** pra números.
- **Azul genérico de SaaS** (`#1d4ed8`/`#2563eb` tipo Tailwind/Bootstrap) como cor primária —
  usamos um teal-azulado mais institucional (`--cor-primaria`), com um teal-marinho mais
  escuro reservado só pro cabeçalho (`--cor-marca`).
- **Cinza puro** em bordas/texto secundário — todos os neutros aqui têm uma leve inclinação
  pra o mesmo matiz do teal (ex. `--cor-borda: #dce3e0`, não `#e5e7eb`), pra parecer
  escolhido, não herdado do browser.
- Gradiente roxo-azul, fundo creme+serif+terracota, `rounded-lg` em tudo, tudo centralizado —
  nenhum desses padrões aparece aqui.

Se uma tela nova (ex. o admin master que o dono do produto mencionou que vem por aí) precisar
de uma cor/fonte que não existe ainda, ela deve nascer como uma extensão desta paleta (mesmo
matiz, mesmo par de fontes), não uma escolha solta.

## Tipografia

- **Base**: `Public Sans` (pesos 400/500/600/700/800), carregada via Google Fonts em
  `index.html`. Fallback `system-ui`.
- **Números**: `IBM Plex Mono` (pesos 500/600) — usada via a classe utilitária `.numerico`
  (`font-family` + `font-variant-numeric: tabular-nums`). Aplicar em qualquer célula/valor
  que seja essencialmente um número ou código: protocolo, horário, contagem regressiva,
  cooldown, estatística grande do dashboard. **Não** aplicar em nome de pessoa, guichê
  ("Guichê 1" é rótulo, não número solto) ou texto livre.
- Títulos (`h1`/`h2`/`h3`) usam `font-weight: 700` e `letter-spacing: -0.01em` por padrão
  (ver bloco global em `index.css`) — não redefinir peso/espaçamento por tela.

## Paleta de cores

Todas as cores vivem como variável CSS em `:root` (`frontend/src/index.css`). Nunca usar um
hex solto num componente — se a cor que você precisa não existe como token, é sinal de que
falta um token, não que vale a pena um valor local.

| Token | Uso | Por quê |
|---|---|---|
| `--cor-fundo` | fundo da página | Off-white quente, não cinza-azulado puro — evita o "branco hospitalar" do protótipo original |
| `--cor-superficie` | fundo de cartão/modal | Branco puro — contraste limpo contra o fundo quente |
| `--cor-texto` / `--cor-texto-suave` | texto principal/secundário | Quase-preto com leve matiz teal, não `#000`/`#1a1a1a` genérico |
| `--cor-borda` | bordas, divisórias | Cinza com leve matiz teal, nunca cinza puro |
| `--cor-marca` / `--cor-marca-clara` | **só** o fundo do cabeçalho e da sidebar do gestor | Default teal-marinho escuro, mas **segue a cor de destaque de cada prefeitura em runtime desde 23/09** (ver seção seguinte) — não usar em botão de conteúdo |
| `--cor-marca-texto` / `--cor-marca-texto-suave` | texto sobre o cabeçalho | Quase-branco com matiz teal, fixo (não acompanha a prefeitura) |
| `--cor-primaria` / `--cor-primaria-forte` | botões, links, foco, aba ativa | Teal-azulado por padrão — a cor de "ação" do sistema, também por prefeitura (mesma cor de destaque que alimenta `--cor-marca`, mas calculada separada) |
| `--cor-sucesso` / `--cor-sucesso-fundo` | chip "no horário", ícone de atendido, notificação de sucesso | Verde natural, não neon |
| `--cor-erro` / `--cor-erro-fundo` | chip "faltou", notificação de erro | Vermelho-tijolo institucional |
| `--cor-aviso` / `--cor-aviso-fundo` | chip "atrasado", linha atrasada na recepção | Âmbar — reservado pra semântica de atraso/atenção, não usar em "informativo neutro" |
| `--cor-info` / `--cor-info-fundo` | `.aviso` (caixa informativa neutra), cabeçalho de tabela, hover de linha | Mesmo tom do primária, fundo bem claro |
| `--cor-encaixe` / `--cor-encaixe-fundo` | chip "encaixe" | Roxo — só esse chip usa roxo no sistema inteiro |

Regra importante: **`--cor-aviso` (âmbar, "atrasado") e `.aviso` (caixa informativa neutra)
não são a mesma coisa** — antes da reforma os dois usavam o mesmo amarelo e ficava ambíguo
se uma caixa amarela era "informação" ou "atenção". Agora `.aviso` é azul-teal (info) e só
chip/linha de "atrasado" usa âmbar.

## Cores configuráveis pelo admin, por prefeitura (22/09, ampliado 23/09)

O admin escolhe 2 cores por prefeitura, na tela "Configurações da Prefeitura"
(`/admin/prefeituras/{id}`): a **de destaque** (ações, links, aba ativa) e a **mais clara**
(fundo, cabeçalho de tabela) — nas palavras do dono do produto. Elas mapeiam DIRETO pros
tokens que já existiam: destaque → `--cor-primaria` (+ `--cor-primaria-forte`, calculada
escurecendo a cor em ~18%, pro hover dos botões), clara → `--cor-info`/`--cor-info-fundo`.

**Desde 23/09, a cor de destaque também alimenta `--cor-marca`/`--cor-marca-clara`** (fundo do
`<Cabecalho>` e da sidebar do gestor) — decisão revertida da original: antes eram uma
identidade fixa, mais escura, separada de propósito, porque só existia uma prefeitura no
sistema; agora que o admin cadastra várias, cada uma com sua própria paleta, não fazia
sentido todo mundo ver o mesmo teal-marinho fixo independente da prefeitura. `--cor-marca` =
a própria cor de destaque; `--cor-marca-clara` = a mesma cor clareada ~22% em direção ao
branco (função `clarear` em `tema.ts`, o inverso de `escurecer`) — pra manter um degradê de
duas pontas a partir de só uma cor escolhida pelo admin. `--cor-marca-texto` continua fixo
(quase-branco), então cores de destaque muito claras podem ter pouco contraste no cabeçalho —
risco aceito, mesmo problema que já existe em botões com `--cor-primaria` muito clara.

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

**Fundo sempre sólido, nunca transparente** — pedido explícito e repetido do dono do produto.
Usa `--cor-marca`/`--cor-marca-clara` em gradiente sutil, nunca uma cor de conteúdo
(`--cor-primaria`) nem `transparent`/`rgba` com alpha baixo no fundo do cabeçalho em si (os
botões *dentro* do cabeçalho é que usam `rgba(255,255,255,...)` pra contrastar sobre o fundo
sólido).

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
protocolo, reaproveitando `.barra-busca`/`.campo-busca` (mesmo padrão da recepção) — mesmo
que não tenha um botão de ação ao lado (nesse caso, `.campo-busca` sozinho dentro de
`.barra-busca`, que ainda cuida do espaçamento). Normalizar a busca (minúsculas, sem
pontuação) com a mesma função `normalizarBusca` já repetida em `Recepcao.tsx`/
`SalaDeEspera.tsx` — se aparecer uma terceira vez, vale extrair pra um util compartilhado.

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

A tipografia (`Public Sans`/`IBM Plex Mono`) continua unificada com o resto do site, como já
estava.

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
