---
name: painel-tv
description: Use ao implementar a tela pública do painel de TV (sem login) ou a lógica de exibição de chamadas/fila de exibição. Contém as regras de negócio definidas pelo dono do produto em 19/09 — não inferir regra nova sem confirmar.
---

# Painel de TV (tela pública, sem login)

## Uma tela só, parametrizada por unidade

**Decisão (19/09)**: não existe uma tela por unidade — é **um componente só**, rota
`/painel/:unidadeId`. Cada TV física (Chromecast/smart TV na sala de espera) aponta pra sua
própria URL, com o `id` (uuid) da unidade dela. **Sem senha** — é uma tela pública, só leitura,
configurada uma vez na instalação física, não algo que alguém "loga" repetidamente. O uuid não
sequencial da unidade já é obfuscação suficiente pro nível de sensibilidade do dado exibido (ver
próxima seção).

## O que aparece na tela

Duas áreas:

1. **Chamada atual, em destaque**: **nome parcial** do cidadão (mesma função já usada antes —
   primeiro nome + inicial do sobrenome, ex: "Maria S."), protocolo, e guichê que chamou.
   - **Decisão (19/09): nome parcial, não completo** — mantém a decisão original do projeto.
     Justificativa do usuário: o protocolo já aparece junto, então dá pra associar a pessoa certa
     sem precisar do nome completo — e como a tela é pública e sem senha, nome completo seria
     desproporcional ao que a tela precisa mostrar.
2. **Últimos chamados**, numa lista à direita.

## Modo "descanso" (idle)

Quando não tem ninguém ocupando o slot de exibição atual (fila de exibição vazia, ver seção
seguinte), a tela **entra em descanso**: a lista de "últimos chamados" se expande pra ocupar a
tela inteira, e cada linha ganha um ícone de status — **três estados possíveis, confirmado
19/09**: `atendido`, `esperando ainda` (chamada, atendente ainda não marcou como atendida), ou
**`ausente`** (não compareceu a tempo — ícone próprio, distinto dos outros dois, não some da
lista). Ver skill `regras-negocio-fila` pros status possíveis de um agendamento — usar o `status`
atual do agendamento por trás de cada linha da lista pra decidir o ícone (`atendido` → ícone de
atendido; `chamado` → ícone de esperando; `ausente` → ícone de ausente).

## Fila de exibição — decisão de arquitetura (19/09)

O usuário descreveu inicialmente um campo booleano (`painel_chama: true/false`) por chamada, com
fila manual quando duas chamadas competem. **Recomendação dada e aceita**: não usar um booleano
persistido. Motivo: um boolean não modela fila (não tem ordem nem identidade de quem está
esperando a vez), e ficaria sujeito a race condition quando dois atendentes chamam ao mesmo
tempo — o mesmo tipo de problema que `FOR UPDATE SKIP LOCKED` já resolve pro "chamar próximo",
mas aqui reaparecendo de outro jeito se não for evitado de propósito.

**Desenho adotado**:

- `chamadas` continua sendo a fonte de verdade — cada `chamar`/`rechamar` cria uma linha lá, sem
  mudança nenhuma (ver `modelo-dados`).
- A "fila de exibição" (o que a tela mostra agora, e por quanto tempo) é **efêmera, em memória,
  dentro do próprio processo do backend Go** — não é uma tabela no Postgres. Um `chamar`/
  `rechamar` empilha um item nessa fila (protegida por mutex ou canal, por unidade); um
  temporizador interno decide o que está em exibição a cada momento.
- **Se o servidor reiniciar, a fila de exibição se perde** — aceitável: o pior caso é uma chamada
  aparecer por menos tempo que o normal; o histórico de verdade (`chamadas`) não é afetado.
- Duração de exibição: `unidades.duracao_chamada_painel_segundos` (padrão 30s) quando a fila de
  exibição está vazia no momento do push; `unidades.duracao_chamada_painel_fila_segundos` (padrão
  15s, mais curta) enquanto há mais chamadas aguardando a vez — drena o backlog mais rápido até
  esvaziar, depois volta pro padrão de 30s. Ambos configuráveis por unidade pelo gestor, mesmo
  padrão de `sla_chegada_minutos`/`sla_atendimento_minutos` (ver skill `modelo-dados`).

## Barra de contagem regressiva + redesenho visual (22/09)

Pedido explícito do dono do produto depois de ver o painel em uso: a chamada em destaque
ganhou uma **barra de progresso regressiva**, mostrando visualmente quanto tempo falta até
sair de exibição (a mesma duração calculada na seção anterior — normal ou "com fila").

- **Puramente decorativa/informativa** — quem decide de verdade quando trocar de exibição
  continua sendo só o backend (a fila de exibição em memória). A barra só reflete essa
  decisão, nunca influencia ela.
- Backend: `GET /painel/{gradeId}` passou a devolver `chamadaAtual.expiraEm` (timestamp) e
  `chamadaAtual.duracaoTotalSegundos`, calculados a partir do mesmo `Atual(...)` que já
  decidia a exibição — sem tabela nova, sem mudar a lógica de decisão em si.
- Frontend: a duração restante é calculada **uma única vez**, no primeiro render daquela
  chamada (`useState` com inicializador lazy) — nunca recalculada nos polls seguintes da
  mesma chamada (o polling roda a cada 2s; recalcular a cada poll reiniciaria a barra
  visualmente). O componente só remonta quando a chamada muda de verdade, via uma `key`
  baseada em protocolo+guichê no componente pai. A barra em si é CSS puro
  (`animation` + `transform: scaleX()`, GPU-acelerada) com a duração vindo de uma variável
  inline — sem JS atualizando estado a cada frame.

**Redesenho geral do layout, mesmo pedido**: fundo da tela agora usa a cor clara/secundária
da prefeitura (ver skill `identidade-visual`, seção "Painel de TV"), com os dois blocos de
conteúdo (chamada atual + tabela de últimas chamadas) como cartões brancos por cima — mais
contraste e legibilidade a distância do que o fundo escuro original. A tabela de últimas
chamadas ganhou cabeçalho de coluna (Nome/Protocolo/Guichê), zebra striping, protocolo em
destaque (cor de acento, fonte mono) e o guichê como "pill" (badge arredondado) — antes era
texto corrido sem nenhuma dessas separações visuais, "feia e sem graça" nas palavras do dono
do produto, especialmente quando as 10 linhas estavam preenchidas. Protocolo e guichê também
ficaram maiores na chamada em destaque (`clamp()` responsivo), pensado pra leitura a
distância numa TV, sem exagerar a ponto de não caber. Ícones de status (`CheckCircle2`/
`Volume2`/`XCircle`, lucide-react) mantidos das rodadas anteriores, só reorganizados dentro
do novo layout de tabela. Tipografia continua `Public Sans`/`IBM Plex Mono` (mesma do resto
do site) — decisão explícita de não usar uma fonte "clássica"/genérica só pro painel.
Verificado ao vivo no Chrome nos dois modos (chamada ativa e descanso).

## Modo "aguardando" — introduzido 23/09, REMOVIDO 20/10

Existiu um terceiro estado do painel (entre ativo e descanso): quando a exibição de uma
chamada (30s/15s) expirava e o atendente ainda estava genuinamente esperando a pessoa (dentro
do cooldown, não órfã), ela ficava visível em destaque indefinidamente em vez de cair pra
tabela — até ser marcada ou o atendente chamar outra. **Removido a pedido do dono do produto
(20/10)**: "fica parecendo que travou a tela" — passados os 30s normais, continuar vendo a
mesma pessoa parada em destaque dava a impressão de painel travado, não de alguém sendo
esperado.

**Como ficou**: só dois modos de novo — ativo (chamada nova em destaque, com narração/barra de
tempo) e descanso (tabela cheia). Assim que a exibição de 30s/15s expira, a pessoa cai direto
pra "últimas chamadas" **sem nenhuma exclusão** — ela já aparece lá mesmo enquanto ainda está
em destaque (não precisa mais esperar o destaque esvaziar). O ícone cinza (ver abaixo)
continua sendo o que distingue visualmente "ainda dentro do prazo" de órfão/atendido/ausente —
não precisava de um modo à parte pra isso, só do ícone certo na linha certa.

**Removido de vez** (não ficou como código morto): `ListarAguardandoExibicao` e
`AguardandoExibicao` (`internal/repository/agendamentos.go`), o parâmetro `aguardando` de
`views.PainelResponse`, o campo `aguardando` da resposta de `GET /painel/{gradeId}` e do tipo
`RespostaPainel` (frontend), o bloco JSX `emAguardando`/`aguardando-grade` de `PainelTV.tsx`, e
as classes CSS `.modo-aguardando`/`.aguardando-grade*`/`.aguardando-bloco`/`.aguardando-nome`.

**Ícone cinza** (`Clock`, lucide-react) — continua existindo e sendo usado, independente do
modo removido: quarto ícone de status na tabela de últimas chamadas, pro caso "chamado, ainda
não apareceu, mas não é órfão" (`ChamadaPainel.ehOrfao === false`). O ícone âmbar (`Volume2`)
fica reservado só pra órfão de verdade.

**Bug real corrigido junto (23/09): "últimas chamadas" mostrava a mesma pessoa duas vezes**
depois de uma rechamada. Causa: `UltimasChamadas` lia direto o log `chamadas` (uma linha por
chamar/rechamar, de propósito append-only — ver skill modelo-dados) sem deduplicar por
agendamento. Corrigido pra uma linha por agendamento (a chamada mais recente de cada um),
reaproveitando os mesmos fragmentos SQL (`juncoesUltimaChamada`/`condicaoOrfao`) já usados em
`ChamarProximo`/`ListarChamadosCompartilhados` — o mesmo cálculo de "é órfão?" usado em toda
parte do sistema agora também alimenta o ícone da tabela do painel, em vez de inventar uma
lógica separada.

**Bug real corrigido junto (23/09): narração podia sobrepor sob chamadas em sequência
rápida.** `speechSynthesis.speak()` do navegador enfileira por padrão, não cancela a fala
anterior — se dois atendentes chamavam quase ao mesmo tempo, o painel podia já estar
mostrando a pessoa B na tela enquanto o áudio ainda terminava de falar o nome de A. Corrigido
com `speechSynthesis.cancel()` no início de cada novo anúncio (`iniciarAnuncio`,
`PainelTV.tsx`) — cada chamada nova descarta imediatamente qualquer fala em andamento ou
enfileirada da anterior antes de começar a sua própria.

## Aviso sonoro (voz) — decisão de arquitetura (19/09)

O `CLAUDE.md` original já previa "considerar aviso sonoro (texto-para-voz) no futuro" — decidido
agora, pro MVP.

**Decisão: Web Speech API do navegador (`speechSynthesis`), não um serviço de TTS de nuvem.**
Motivo: roda inteiramente no frontend (dispara ao entrar uma nova chamada no destaque), zero
custo, zero chamada de API externa, zero credencial — bate com o orçamento zero do piloto. Nativa
em qualquer navegador Chromium (Chromecast/Google TV/smart TV moderna). Trade-off aceito: a
qualidade/disponibilidade de voz pt-BR varia por dispositivo — se isso incomodar depois do
piloto, trocar por TTS de nuvem (Google Cloud TTS/Azure/ElevenLabs) é só trocar a *fonte do
áudio*, não a arquitetura de disparo.

**Disparo**: uma vez por slot de exibição da fila (ver seção anterior) — quando uma chamada
**entra** em destaque, não a cada ciclo de polling que ainda mostra a mesma chamada.

**Dois pontos ainda sem confirmação final do usuário — documentados aqui como recomendação, não
como decisão travada:**
- **O que falar**: recomendado nome parcial + guichê (ex: "Maria S., guichê 3") — **não** o
  protocolo (estranho/longo de ouvir em voz alta; fica só na tela, visual, pra conferência).
- **Toque de atenção antes de falar**: recomendado um "ding" curto antes da fala, pra chamar
  atenção de quem está de costas pra tela.

**Restrição técnica de navegador, não é decisão de produto**: navegadores bloqueiam áudio
automático até haver uma interação do usuário na página carregada. Isso implica uma tela de
"toque pra iniciar o painel" (ou similar) na primeira carga, resolvida uma vez na instalação
física da TV, não contornável só no código.

**Fallback**: se `speechSynthesis` não existir ou não tiver voz pt-BR disponível naquele
navegador/dispositivo, degrada pra só o "ding" (ou silêncio) — a tela visual funciona de qualquer
jeito, o áudio é reforço, não dependência.

## Endpoint (visão geral, sem código)

Um endpoint de leitura pro painel consultar, por polling (3-5s, mesmo padrão do resto do
projeto — sem WebSocket na fase gratuita), retornando: a chamada atualmente em exibição (ou nulo,
se em modo descanso) e a lista de últimos chamados com seus status. O cálculo de "o que está em
exibição agora" é feito no backend a partir da fila de exibição em memória — o frontend só
renderiza o que vier.

## Pendências

- Formato exato do "nome parcial" já está implementado numa versão anterior do projeto
  (`_archive/node-legado`) — reaproveitar a mesma função ao portar pro Go.
