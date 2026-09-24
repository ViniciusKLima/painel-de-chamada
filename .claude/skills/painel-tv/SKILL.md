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

## Modo "aguardando" — terceiro estado do painel, entre ativo e descanso (23/09)

Pedido do dono do produto depois de ver o painel em uso: quando a exibição de uma chamada
(30s/15s, ver seção anterior) expirava, a pessoa desaparecia do destaque e ia direto pra
tabela — mas o atendente muitas vezes ainda estava genuinamente esperando por ela (dentro do
cooldown, ainda não virou órfã). Ficava estranho sumir do destaque só porque o TEMPO DE
EXIBIÇÃO acabou, quando a pessoa continua sendo esperada de verdade.

**Desenho**: um terceiro modo, só ativado quando a fila de exibição em memória está vazia
(`Atual()` retornou nil — nenhuma chamada NOVA pra anunciar agora). Nesse caso, o backend
consulta separadamente (`ListarAguardandoExibicao`, sem tocar na fila de exibição em memória)
até 3 agendamentos com status `chamado`, **não órfãos**, ordenados pela chamada mais antiga
primeiro, e devolve em `aguardando` na resposta de `GET /painel/{gradeId}`. Regras
importantes:

- **Nunca compete com uma chamada nova real** — só aparece quando não há nada novo pra
  anunciar. Uma chamada nova sempre tem prioridade total (com narração, barra de tempo).
- **Chamado órfão NUNCA aparece aqui** — "se estiver órfão não fica no painel" (pedido
  explícito). Órfão continua só na tabela de últimas chamadas, com o ícone âmbar de sempre
  (ver seção de ícones abaixo) — ele já não é mais "alguém que o atendente está esperando
  normalmente", é alguém que precisa que outro atendente assuma.
- **Sem narração, sem barra de contagem** — é presença visual pura, não uma "chamada" no
  sentido de evento sonoro. `PainelTV.tsx` nunca chama `iniciarAnuncio` pros itens de
  `aguardando`.
- **Até 4 blocos, arranjo específico por quantidade** (23/09, `aguardando-grade-1/2/3/4` em
  `index.css`, CSS Grid): 1 ocupa tudo; 2 empilham numa coluna só (mais recente em cima); 3
  ficam com a mais recente em cima (largura toda) + as outras duas lado a lado embaixo; 4
  viram uma grade 2x2. Ordem sempre do mais recente pro mais antigo (`PainelTV.tsx` inverte a
  lista que o backend devolve — o backend ordena do mais antigo pro mais novo, porque é assim
  que ele decide quem entra quando há mais candidatos que o limite). Tamanho de fonte adapta
  à quantidade (bloco sozinho maior que quando 4 dividem o espaço). Nome, protocolo e guichê
  ficam empilhados em coluna dentro de cada bloco, reaproveitando as mesmas classes CSS da
  tabela padrão (`historico-protocolo`/`historico-guiche-pill`) em vez de uma tipografia
  própria — pedido explícito: "mesma estrutura do padrão".
- **A tabela de últimas chamadas exclui quem estiver no destaque** (a chamada ativa OU os
  blocos de aguardando) — o backend já filtra isso (`protocolosEmDestaque` em
  `handlers/painel.go`), o frontend só renderiza o que vier.

**Ícone cinza novo** (`Clock`, lucide-react) — quarto ícone de status na tabela, pro caso
"chamado, ainda não apareceu, mas não é órfão" (`ChamadaPainel.ehOrfao === false`). O ícone
âmbar (`Volume2`) que já existia fica reservado só pra órfão de verdade — antes ele
representava "chamado" genericamente, sem distinguir os dois casos.

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
