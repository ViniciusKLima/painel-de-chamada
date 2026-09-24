# Feature Specification: Painel de TV (exibição pública de chamadas)

**Feature Branch**: `003-painel-tv`

**Created**: 2026-09-19

**Last updated**: 2026-09-24

**Status**: Implementado (piloto)

**Input**: User description original (19/09): levantamento do fluxo do painel de TV feito diretamente com o dono do produto: tela dividida entre "quem está sendo chamado agora" e uma lista de "últimos chamados" à direita; a lista se expande pra tela inteira em modo de descanso quando não há chamada em exibição, mostrando ícone de status (atendido/esperando) por linha; ao chamar alguém, exibe nome (parcial), protocolo e guichê; toda chamada simultânea de atendentes diferentes deve ser exibida em sequência (fila de exibição), com duração normal e uma duração reduzida enquanto há backlog; a tela é única, parametrizada pela unidade, sem exigir senha.

**Mudanças desde a versão original (ver skill `painel-tv`)**:
- A tela é parametrizada por **GRADE**, não por unidade (`/painel/{gradeId}`) — uma unidade pode
  ter mais de uma fila (uma por serviço), cada uma com seu próprio painel/link público.
- **Nome completo, não mais parcial** (decisão revertida 20/09, pedido direto do dono do
  produto) — ver Assumptions pra a tensão com privacidade que essa mudança implica.
- Novo **modo "aguardando"**: quem já foi chamado e ainda está dentro do prazo (não órfão) fica
  visível num bloco de destaque, sem narração, em vez de sumir direto pra tabela.
- Tema claro com cores por prefeitura, barra de contagem regressiva visual, e uma **central de
  painéis com kiosk público por unidade** (`/tv/{unidadeId}`) que lista as grades daquele local.
- Repetição do anúncio de voz (quantas vezes e de quanto em quanto tempo) é configurável por
  grade, não mais fixa em código.
- Os dois `[NEEDS CLARIFICATION]` da versão original (texto exato da voz, som antes da fala)
  foram resolvidos com o dono do produto — ver User Story 5.

**Relação com outros specs**: consome os agendamentos e chamadas produzidos pelos fluxos de
`specs/001-fila-atendimento-presencial` — não redefine regras de fila/chamada, só como elas são
**exibidas** publicamente.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Painel exibe a chamada atual em destaque (Priority: P1)

Um atendente chama um cidadão. A tela pública da grade (fila de um serviço dentro de uma
unidade) passa a exibir, em destaque, o nome completo da pessoa, o protocolo do agendamento, e
o guichê pra onde ela deve ir — nome maior e no topo, protocolo logo abaixo, guichê por último.

**Why this priority**: é o motivo do painel existir — sem isso, a chamada continua dependendo de
alguém gritar o nome na sala.

**Independent Test**: chamar um agendamento e verificar que a tela do painel daquela grade passa
a exibir nome completo + protocolo + guichê dentro do intervalo de atualização esperado.

**Acceptance Scenarios**:

1. **Given** um agendamento chamado por um atendente num guichê específico, **When** o painel da
   grade correspondente atualiza, **Then** ele exibe o nome completo da pessoa, o protocolo, e o
   guichê chamado, nessa ordem visual (nome → protocolo → guichê).
2. **Given** a chamada em destaque, **When** o tempo de exibição configurado passa, **Then** uma
   barra de contagem regressiva visual acompanha o tempo restante daquela exibição (puramente
   decorativa — quem decide de fato quando trocar de exibição continua sendo só o backend).

---

### User Story 2 - Chamadas simultâneas entram numa fila de exibição (Priority: P2)

Dois atendentes chamam pessoas diferentes quase ao mesmo tempo, na mesma grade. O painel não
tenta mostrar as duas de uma vez nem descarta uma delas — exibe uma de cada vez, em sequência,
sem perder nenhuma chamada.

**Why this priority**: numa grade com vários guichês ativos, chamadas simultâneas são esperadas;
perder ou sobrepor uma chamada no painel frustraria o propósito da tela.

**Independent Test**: disparar duas chamadas em guichês diferentes da mesma grade em sequência
rápida, e confirmar que o painel exibe as duas, uma após a outra, sem sobrepor.

**Acceptance Scenarios**:

1. **Given** duas chamadas feitas quase ao mesmo tempo na mesma grade, **When** o painel
   processa a exibição, **Then** ambas aparecem em destaque, uma depois da outra, cada uma pelo
   tempo configurado.
2. **Given** uma fila de chamadas aguardando exibição (mais de uma chamada pendente), **When** o
   painel exibe cada uma, **Then** o tempo de exibição de cada chamada é reduzido (duração de
   "fila") até a fila de exibição se esvaziar, voltando então à duração normal.
3. **Given** uma chamada em exibição e nenhuma outra na fila, **When** ela termina seu tempo de
   exibição, **Then** a próxima exibição usa a duração normal (não a reduzida).

---

### User Story 3 - Modo "aguardando" para quem já foi chamado mas ainda não apareceu (Priority: P3)

Quando a janela de destaque de uma chamada (30s/15s) expira e não há chamada nova pra assumir o
lugar, a pessoa não desaparece direto pra tabela se o atendente ainda estiver genuinamente
esperando por ela (chamado, dentro do prazo, não órfão) — ela fica visível num bloco de
destaque próprio, sem narração nem barra de tempo, puramente informativo. Se houver de 1 a 4
pessoas nessa situação ao mesmo tempo, todas aparecem lado a lado, cada uma em seu próprio
cartão (mais recente primeiro), num arranjo que se adapta à quantidade.

**Why this priority**: sem isso, alguém que já foi chamado "desaparecia" da área de destaque
assim que o tempo de exibição passava, mesmo que o atendente ainda estivesse esperando por ela —
dando a falsa impressão de que a chamada já tinha sido resolvida.

**Independent Test**: chamar uma pessoa, esperar a janela de exibição normal expirar sem uma
chamada nova, e confirmar que ela continua aparecendo em destaque (modo aguardando), não na
tabela — e que um chamado órfão nunca aparece nesse modo, só na tabela.

**Acceptance Scenarios**:

1. **Given** uma pessoa chamada, ainda dentro do prazo, sem chamada nova disputando o destaque,
   **When** a janela de exibição normal expira, **Then** ela continua visível num cartão de
   destaque próprio, sem barra de contagem nem narração.
2. **Given** até 4 pessoas nessa situação ao mesmo tempo, **When** o painel exibe o modo
   aguardando, **Then** todas aparecem lado a lado, mais recente primeiro, sem competir por
   prioridade entre si.
3. **Given** um chamado que virou órfão (dono já rechamou e seguiu em frente), **When** o painel
   decide quem entra no modo aguardando, **Then** o órfão NUNCA aparece nesse modo — continua
   só na tabela de últimas chamadas, com o ícone de status correspondente.

---

### User Story 4 - Modo de descanso quando não há chamada nem aguardando pra exibir (Priority: P4)

Quando não há nenhuma chamada em destaque nem ninguém no modo aguardando, o painel entra em modo
de descanso: a lista de últimos chamados ocupa a tela inteira, mostrando o status de cada um.

**Why this priority**: evita uma tela de destaque vazia/estática por longos períodos ociosos, e
aproveita o espaço pra mostrar informação útil (quem já foi atendido, quem ainda está a
caminho do guichê, quem é órfão).

**Independent Test**: esperar o painel ficar sem nenhuma chamada em exibição nem aguardando, e
confirmar que a lista de últimos chamados se expande pra tela inteira, com um indicador visual
de status em cada linha.

**Acceptance Scenarios**:

1. **Given** nenhuma chamada em exibição nem aguardando no momento, **When** o painel atualiza,
   **Then** a área de destaque não fica vazia — a lista de últimos chamados ocupa a tela inteira,
   centralizada.
2. **Given** o modo de descanso ativo, **When** uma nova chamada acontece, **Then** o painel sai
   do modo de descanso e volta a exibir a área de destaque (70% da largura) com a lista reduzida
   ao lado (30%) — a transição é suave (animada), não uma troca abrupta.
3. **Given** a lista completa (modo de descanso ou reduzida ao lado da chamada em destaque),
   **When** o gestor/atendente olha pra ela, **Then** cada linha indica visualmente um de quatro
   estados possíveis: atendido, aguardando comparecer (chamado, dentro do prazo, não órfão),
   chamado órfão (sem resposta, precisa ser assumido), ou ausente — cada um com ícone e cor
   próprios.

---

### User Story 5 - Chamada é anunciada em voz, não só exibida na tela (Priority: P5)

Quando uma chamada entra em destaque, o painel também anuncia em voz (texto-para-voz): um beep
curto, seguido de "{NOME}. Guichê: {número por extenso}" — nunca o protocolo, que soa estranho
falado em voz alta. O anúncio repete um número configurável de vezes (por grade), com um
intervalo configurável entre repetições, sem duplicar nem sobrepor mesmo sob chamadas em
sequência rápida.

**Why this priority**: reforça o propósito do painel (menos gente perdendo a própria chamada por
não estar de olho na tela), mas o sistema já funciona sem isso — é um complemento, não um
bloqueador do restante do painel.

**Independent Test**: disparar uma chamada e confirmar, via inspeção do áudio produzido, que o
painel fala exatamente nome + guichê (nunca o protocolo), o número de vezes configurado, sem
duplicar.

**Acceptance Scenarios**:

1. **Given** uma chamada entra em destaque no painel, **When** o anúncio dispara, **Then** o
   painel toca um beep e fala "{NOME}. Guichê: {número por extenso}" — nunca o protocolo.
2. **Given** a grade configurada pra repetir 3 vezes com 5s de intervalo, **When** uma chamada
   entra em destaque, **Then** o anúncio completo (beep + fala) se repete exatamente 3 vezes,
   uma a cada ~5s, nunca mais nem menos.
3. **Given** uma chamada nova entra em destaque enquanto o anúncio da chamada anterior ainda
   está tocando ou repetindo, **When** o novo anúncio começa, **Then** a fala anterior é
   cancelada na hora (nunca sobrepõe) e só as repetições da chamada nova continuam.
4. **Given** o dispositivo do painel não consegue produzir áudio falado (recurso indisponível ou
   sem voz no idioma), **When** uma chamada entra em destaque, **Then** a exibição visual
   continua funcionando normalmente.
5. **Given** a tela do painel acabou de carregar pela primeira vez num dispositivo, **When** o
   primeiro anúncio em voz é esperado, **Then** uma interação inicial na tela ("toque pra
   iniciar") libera o áudio antes de qualquer chamada real acontecer — considerado parte da
   instalação física, não uma falha do sistema.

---

### User Story 6 - Central de painéis e acesso direto pela TV do local, sem login (Priority: P6)

Como cada grade tem seu próprio painel, uma unidade com várias filas (ou uma secretaria com
várias unidades) pode ter muitos painéis. O gestor tem uma central de painéis (autenticada) que
lista todos, agrupados por unidade, com busca e um botão de tela cheia. Além disso, cada unidade
tem um link **público, sem login**, que lista só as grades ATIVAS daquele local físico — pensado
pra ficar fixo na TV de verdade do CRAS, sem depender de ninguém logado pra abrir o painel certo.

**Why this priority**: sem isso, alguém precisaria saber de cor o link de cada grade — inviável
assim que o número de filas cresce.

**Independent Test**: acessar o link público de uma unidade sem nenhum cookie de sessão, e
confirmar que ele lista só as grades ativas daquele local, sem expor nenhum dado de
agendamento/cidadão (isso só aparece dentro do painel de cada grade específica, que já era
público desde sempre).

**Acceptance Scenarios**:

1. **Given** uma unidade com 3 grades ativas e 1 inativa, **When** alguém acessa o link público
   daquela unidade, **Then** vê só as 3 grades ativas, sem precisar de login.
2. **Given** o gestor autenticado, **When** ele abre a central de painéis da secretaria, **Then**
   vê todas as grades agrupadas por unidade (grades da mesma unidade aparecem juntas, sob um
   cabeçalho comum com o nome dela) — nunca como uma lista plana sem agrupamento.
3. **Given** um painel aberto "em foco" dentro da central (via iframe), **When** o gestor aciona
   "tela cheia", **Then** o painel ocupa a tela física inteira, não só o cartão dentro da tela do
   gestor.

---

### Edge Cases

- **Resolvido**: uma linha da lista de últimos chamados/modo de descanso cujo status virou
  "ausente" ganha seu próprio indicador visual, distinto de atendido, aguardando e órfão (ver
  FR-009).
- **Resolvido**: um chamado órfão nunca entra no modo "aguardando" (User Story 3) — só aparece
  na tabela, com o ícone de órfão específico.
- O que acontece se o processo que mantém a fila de exibição reiniciar com chamadas pendentes de
  exibição? (assumido: a fila de exibição se perde, mas o histórico de chamadas não é afetado —
  ver Assumptions)
- O que acontece se uma grade tiver um identificador inválido ou inexistente na URL do painel?
- A lista de "últimas chamadas" mantém até 10 entradas (modo normal e de descanso); o modo
  aguardando mantém até 4 pessoas simultâneas — quem excede esses limites é priorizado por quem
  está esperando/foi chamado há mais tempo.
- **Resolvido**: o anúncio fala exatamente "{NOME}. Guichê: {número por extenso}", nunca o
  protocolo — decisão final do dono do produto depois de testar formatos mais longos.
- **Resolvido**: um beep curto (Web Audio API) sempre antes da fala.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema DEVE exibir publicamente, sem exigir autenticação, uma tela de painel
  identificada pela GRADE a que pertence (`/painel/{gradeId}`).
- **FR-001a**: O sistema DEVE oferecer um link público adicional, por UNIDADE, sem autenticação,
  listando só as grades ativas daquele local físico (pra acesso direto pela TV do CRAS).
- **FR-002**: O sistema DEVE exibir, para a chamada atualmente em destaque, o nome COMPLETO do
  cidadão, o protocolo do agendamento, e o guichê que realizou a chamada, nessa ordem visual.
- **FR-003**: O sistema DEVE exibir, ao lado da chamada em destaque (ou centralizada, em modo de
  descanso), uma lista dos até 10 chamados mais recentes daquela grade, com nome e protocolo
  completos.
- **FR-004**: O sistema DEVE processar chamadas simultâneas de atendentes diferentes da mesma
  grade em sequência, exibindo cada uma por sua vez, sem perder nem sobrepor nenhuma.
- **FR-005**: O sistema DEVE reduzir a duração de exibição de cada chamada enquanto houver outras
  chamadas aguardando para serem exibidas, e retomar a duração normal assim que não houver mais
  backlog.
- **FR-006**: O sistema DEVE permitir que a duração normal e a duração reduzida de exibição sejam
  configuradas individualmente por GRADE.
- **FR-007**: O sistema DEVE apresentar um modo "aguardando" (até 4 pessoas chamadas, dentro do
  prazo, ainda não em destaque e não órfãs) quando não houver chamada nova disputando o destaque
  — sem narração nem barra de tempo, cada pessoa em seu próprio cartão visual.
- **FR-007a**: Um chamado órfão NUNCA DEVE aparecer no modo "aguardando" — só na lista de
  últimas chamadas, com indicador visual próprio.
- **FR-008**: O sistema DEVE apresentar um modo "descanso" quando não houver nenhuma chamada em
  destaque nem ninguém no modo aguardando, expandindo a lista de chamados recentes para ocupar
  todo o espaço da tela, centralizada.
- **FR-009**: O sistema DEVE indicar visualmente, para cada item da lista de chamados recentes
  (inclusive em modo de descanso), qual dos quatro estados se aplica: atendido, aguardando
  comparecimento (chamado, dentro do prazo, não órfão), órfão (sem resposta, precisa ser
  assumido), ou ausente — cada um com ícone e cor próprios, distintos dos demais.
- **FR-010**: O sistema DEVE voltar automaticamente do modo de descanso (ou aguardando) para a
  exibição normal assim que uma nova chamada acontecer, com uma transição visual suave.
- **FR-011**: O sistema DEVE produzir um anúncio falado (texto-para-voz) toda vez que uma chamada
  entrar em destaque, repetindo um número de vezes e com um intervalo configuráveis por grade.
- **FR-011a**: O sistema DEVE cancelar imediatamente qualquer anúncio (ou repetição pendente) de
  uma chamada anterior assim que uma chamada nova entrar em destaque — nunca sobrepor duas falas.
- **FR-012**: O sistema NÃO DEVE falar o protocolo do agendamento no anúncio — só nome e guichê.
- **FR-013**: O sistema DEVE continuar exibindo a chamada visualmente mesmo quando o anúncio
  falado não puder ser produzido (dispositivo sem suporte, sem voz no idioma disponível, etc.) —
  o anúncio é reforço, não pré-requisito da exibição.
- **FR-014**: O sistema DEVE oferecer, pro gestor autenticado, uma central listando todos os
  painéis (grades) da secretaria, agrupados visualmente por unidade — grades da mesma unidade
  nunca aparecem como cartões soltos e independentes.
- **FR-015**: O sistema DEVE permitir abrir qualquer painel "em foco" e em tela cheia de verdade
  a partir da central de painéis do gestor.
- **FR-016**: O sistema DEVE aplicar as cores de identidade (destaque/clara) da prefeitura dona
  da grade ao tema visual do painel público, resolvidas via unidade → secretaria → prefeitura.

### Key Entities

- **Chamada** (já definida em `specs/001-fila-atendimento-presencial`): este spec consome o
  histórico de chamadas para compor tanto a exibição em destaque quanto a lista de recentes.
- **Grade** (já definida em `specs/001-fila-atendimento-presencial`): a unidade de escopo do
  painel — cada grade tem seu próprio painel público e sua própria configuração de duração/
  repetição de anúncio.
- **Fila de exibição**: conceito de apresentação, em memória (não é tabela de banco) — a ordem e
  o tempo que cada chamada ocupa o destaque do painel de uma grade; derivado do histórico de
  chamadas mais as configurações de duração daquela grade.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Toda chamada feita por um atendente aparece em destaque no painel da grade correta
  dentro do intervalo de atualização configurado, sem exceção.
- **SC-002**: Nenhuma chamada deixa de ser exibida no painel, mesmo sob chamadas simultâneas de
  múltiplos atendentes na mesma grade.
- **SC-003**: A tela do painel nunca fica com a área de destaque vazia — mostra uma chamada, o
  modo aguardando, ou está em modo de descanso mostrando a lista expandida.
- **SC-004**: Uma nova grade pode ganhar seu próprio painel sem exigir nenhuma tela nova —
  apenas apontando pro próprio identificador de grade.
- **SC-005**: Uma pessoa que não está olhando diretamente pra tela no momento da própria chamada
  ainda consegue ser avisada, através do anúncio falado, sem risco de perder o anúncio por
  sobreposição com outra chamada.
- **SC-006**: Um gestor consegue encontrar e abrir o painel de qualquer grade da secretaria em
  menos de dois cliques a partir da central de painéis, sem precisar guardar links de cor.

## Assumptions

- **Privacidade (revisão explícita 20/09)**: a decisão de exibir nome parcial (privacidade, versão
  original deste spec) foi revertida a pedido direto do dono do produto — o painel público agora
  mostra o nome COMPLETO do cidadão, numa tela sem autenticação, visível a qualquer um fisicamente
  presente na sala de espera. Essa é uma tensão real com a intenção original de privacidade;
  documentado aqui pra qualquer revisão futura confirmar se essa implicação continua aceitável.
- A "fila de exibição" (ordem e tempo de destaque de cada chamada) é um estado transitório,
  reconstruível a partir do histórico de chamadas — não precisa sobreviver a um reinício do
  sistema sem perda aceitável (na pior hipótese, uma chamada é exibida por menos tempo).
- O identificador de grade usado na URL/configuração do painel não precisa de proteção adicional
  (senha) — mesmo com nome completo, é o mesmo padrão de exposição que uma recepção física já
  teria (chamar o nome em voz alta).
- O canal de atualização do painel usa polling (2s, reduzido de 4s numa rodada posterior à versão
  original pra diminuir o delay percebido) — sem WebSocket nesta fase.
- O anúncio falado é produzido pelo próprio dispositivo que exibe o painel (Web Speech API do
  navegador, sem depender de um serviço externo de conversão de texto em voz).
- Liberar áudio automático num dispositivo exige uma interação inicial na própria tela ("toque
  pra iniciar"), tratada como parte da instalação física do painel, não uma responsabilidade do
  backend.
