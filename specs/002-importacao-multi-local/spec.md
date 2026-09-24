# Feature Specification: Importação de planilha com roteamento automático por local e por grade

**Feature Branch**: `002-importacao-multi-local`

**Created**: 2026-09-19

**Last updated**: 2026-09-24

**Status**: Implementado (piloto)

**Input**: User description original (19/09): "Importação de planilha de agendamentos com separação automática por local. O gestor sobe uma única planilha (associada à secretaria, não a uma unidade específica) contendo agendamentos de múltiplos locais físicos de atendimento (unidades/CRAS) misturados — cada linha tem uma coluna 'Local no mapa' que identifica a unidade correspondente. O sistema deve identificar automaticamente a qual unidade cada linha pertence e distribuir os agendamentos para a fila correta de cada unidade, sem que o gestor precise separar manualmente. Linhas com local não cadastrado no sistema devem gerar erro específico daquela linha, sem descartar as linhas válidas do resto do arquivo. Uma linha pode representar um agendamento cancelado pelo cidadão antes de comparecer (diferente de ausência, que só acontece depois que a pessoa já está na fila física) — esses não devem entrar na fila de atendimento, mas devem ficar registrados para relatórios futuros."

**Mudança de modelo desde a versão original (rodada de 22/09, ver skill `modelo-dados` e
`importacao-planilha`)**: a versão original deste spec tratava "Grade de horários" como um
atributo puramente informativo, sem efeito no roteamento — só o "Local no mapa" separava a fila.
Isso mudou: confirmado com o dono do produto que uma unidade física pode oferecer **vários
serviços ao mesmo tempo, cada um com sua própria fila** (a entidade `grades`). Cada linha da
planilha agora resolve **duas coisas de roteamento**, não uma: a **unidade** (via "Local no
mapa", sem mudança) e a **grade** dentro dela (via "Serviço" — chave de roteamento — e "Grade de
horários" — nome de exibição). O fluxo também ganhou uma etapa de **preview em duas fases**
(analisa sem gravar → gestor confirma) que não existia na versão original.

**Relação com outros specs**: este spec cobre a importação e o roteamento por local e por grade.
A fila de atendimento em si (recepção, sala de espera, chamada) está em
`specs/001-fila-atendimento-presencial` — os agendamentos criados aqui alimentam aquele fluxo,
mas as regras de prioridade/chamada não são redefinidas neste documento.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Gestor importa uma planilha com vários locais e serviços misturados (Priority: P1)

O gestor sobe uma única planilha do dia contendo agendamentos de várias unidades diferentes
(vários CRAS) e de mais de um serviço, sem separar nada manualmente. O sistema identifica a
unidade **e** a grade (serviço) de cada linha e distribui cada agendamento pra fila correta.

**Why this priority**: é o comportamento central desta funcionalidade — sem ele, o gestor
voltaria a precisar separar manualmente a planilha antes de subir, o que anula o ganho de trazer
tudo numa importação só.

**Independent Test**: subir uma planilha com linhas de pelo menos duas unidades e dois serviços
diferentes e confirmar que cada agendamento aparece na fila da unidade e grade certas, sem
nenhuma ação manual de separação.

**Acceptance Scenarios**:

1. **Given** uma planilha com agendamentos de duas unidades cadastradas ("CRAS A" e "CRAS B") e
   dois serviços diferentes em cada uma, **When** o gestor confirma a importação, **Then** cada
   agendamento aparece na fila da unidade E da grade (serviço) corretas — quatro filas distintas
   no total, sem nenhuma separação manual.
2. **Given** a mesma planilha, **When** a importação termina, **Then** o sistema reporta quantas
   unidades e grades diferentes foram identificadas/criadas e quantos agendamentos foram criados
   em cada uma.

---

### User Story 2 - Gestor analisa antes de confirmar (preview em duas fases) (Priority: P2)

O gestor escolhe o arquivo, e o sistema analisa a planilha inteira **sem gravar nada** —
mostrando quantos agendamentos válidos, cancelados e com erro existem, uma lista de agendamentos
por dia, quais unidades e grades novas seriam criadas, e a tabela de erros linha a linha. Só
depois de ver esse resumo o gestor confirma, reenviando o mesmo arquivo pra gravação de verdade.

**Why this priority**: evita que um erro de planilha (arquivo errado, local mal escrito em massa)
vire uma importação real sem chance de revisão antes.

**Independent Test**: subir um arquivo, conferir que a tela de preview mostra os números certos
sem nada ter sido gravado no banco, e só depois confirmar e ver os mesmos números persistidos.

**Acceptance Scenarios**:

1. **Given** um arquivo escolhido, **When** o gestor aciona a análise, **Then** o sistema mostra
   o resumo completo (válidos/cancelados/erros, por dia, unidades/grades novas) sem criar nenhum
   agendamento no banco.
2. **Given** o resumo já exibido, **When** o gestor confirma, **Then** o mesmo arquivo é
   reenviado e processado de verdade, com os números batendo com o que foi mostrado no preview.
3. **Given** um arquivo já importado antes (protocolos repetidos), **When** o gestor roda o
   preview de novo, **Then** as linhas aparecem como erro de duplicidade, sem gravar nada —
   confirmando que a checagem de duplicidade funciona igual nas duas fases.

---

### User Story 3 - Local novo é criado automaticamente, sem travar a importação (Priority: P3)

Uma planilha traz um local que ainda não existe cadastrado no sistema (unidade nova, ou primeira
vez que aparece). O sistema cria essa unidade automaticamente a partir do nome encontrado na
planilha, e os agendamentos daquele local passam a ser roteados pra ela normalmente — inclusive
os de linhas seguintes com o mesmo nome de local, dentro da mesma importação.

**Why this priority**: sem isso, toda unidade nova exigiria um cadastro manual prévio antes de
qualquer importação poder incluí-la — o que atrasaria a operação sempre que a rede de
atendimento crescer.

**Independent Test**: subir uma planilha com um local que nunca apareceu antes no sistema, e
confirmar que uma unidade nova é criada com esse nome, e que todos os agendamentos daquele local
(inclusive múltiplas linhas com o mesmo nome) caem na fila dessa mesma unidade nova.

**Acceptance Scenarios**:

1. **Given** uma planilha com 3 linhas de um local que não existe ainda no sistema, **When** o
   gestor importa, **Then** uma única unidade nova é criada (não três), e as 3 linhas caem na
   fila dela.
2. **Given** um local cujo nome varia só por formatação (espaço extra, acentuação, caixa) em
   relação a uma unidade já cadastrada, **When** o gestor importa, **Then** o sistema reconhece
   como a mesma unidade existente, sem criar uma duplicata.
3. **Given** uma planilha com uma linha cujo campo de local está vazio (não só com nome diferente,
   mas ausente), **When** o gestor importa, **Then** essa linha é rejeitada com erro específico —
   local ausente não gera criação automática, só nome de local preenchido e não encontrado.

---

### User Story 4 - Grade nova (serviço) é criada automaticamente dentro da unidade certa (Priority: P4)

Dentro de uma unidade (já existente ou recém-criada pela User Story 3), uma linha traz um
serviço que ainda não tem grade cadastrada ali. O sistema cria a grade automaticamente — usando
"Serviço" como chave de roteamento (comparação normalizada) e "Grade de horários" como nome de
exibição, com um guichê padrão já configurado — sem travar a importação esperando cadastro
manual prévio.

**Why this priority**: mesma lógica da User Story 3, mas no nível de serviço dentro do local —
sem isso, um serviço novo dentro de uma unidade já existente também exigiria cadastro manual
antes de qualquer planilha poder incluí-lo.

**Independent Test**: importar uma planilha com uma linha de um serviço novo numa unidade já
existente, e confirmar que uma grade nova é criada, com nome vindo da coluna "Grade de horários"
(não do texto do serviço), e já com um guichê padrão.

**Acceptance Scenarios**:

1. **Given** uma unidade já existente sem nenhuma grade pro serviço "Bolsa Família", **When** o
   gestor importa uma linha desse serviço, **Then** uma grade nova é criada dentro daquela
   unidade, com o nome vindo de "Grade de horários" (ou do próprio serviço, se essa coluna vier
   vazia) e um guichê padrão ("Guichê 1") já configurado.
2. **Given** duas linhas do mesmo serviço na mesma unidade, **When** o gestor importa, **Then**
   as duas caem na mesma grade — não são criadas duas grades pro mesmo serviço.
3. **Given** o mesmo nome de serviço em DUAS unidades diferentes, **When** o gestor importa,
   **Then** duas grades distintas são criadas, uma em cada unidade — a chave de roteamento da
   grade é sempre (unidade, serviço), nunca só o serviço isolado.

---

### User Story 5 - Cancelamento prévio não entra na fila física (Priority: P5)

Uma linha da planilha representa um agendamento que o cidadão cancelou antes mesmo de ir à
unidade. Esse registro precisa existir no sistema pra fins de relatório, mas nunca deve aparecer
como algo que a recepção precisa confirmar chegada, nem que o atendente possa chamar.

**Why this priority**: sem essa distinção, cancelamentos prévios inflariam artificialmente a
lista de "aguardando chegada" da recepção, e poderiam ser contados incorretamente como ausência
num relatório futuro.

**Independent Test**: importar uma planilha com uma linha cancelada e confirmar que ela nunca
aparece na tela da recepção nem na sala de espera, mas existe no sistema pra consulta.

**Acceptance Scenarios**:

1. **Given** uma linha da planilha marcada como cancelada pelo cidadão, **When** o gestor
   importa, **Then** o agendamento é registrado com um status de cancelado, distinto de
   "aguardando chegada" e de "ausente".
2. **Given** esse mesmo agendamento cancelado, **When** a recepção olha a lista do dia, **Then**
   ele não aparece como algo pendente de confirmação de chegada.

---

### Edge Cases

- **Resolvido (19/09)**: quando o nome do local na planilha não bate com nenhuma unidade já
  cadastrada, o sistema **cria a unidade automaticamente** com esse nome, na secretaria do lote —
  não é mais um erro de linha (ver FR-002a).
- **Resolvido (22/09)**: quando o serviço de uma linha não bate com nenhuma grade já cadastrada
  DENTRO daquela unidade, o sistema **cria a grade automaticamente** — não é mais tratado como
  "grade de horários é só informativa" (decisão revertida da versão original deste spec, ver
  FR-002c/FR-002d).
- **Resolvido (19/09)**: uma linha cancelada cujo motivo indique reagendamento (não desistência
  de verdade) é importada **igual a qualquer outra cancelada** por enquanto — sem detecção nem
  tratamento especial. Explicitamente registrado que isso **não é o mesmo** que um cancelamento
  de verdade pra fins de relatório futuro (taxa de não comparecimento etc.) — a forma de separar
  os dois casos fica pra uma discussão futura, fora deste spec.
- O que acontece se a mesma planilha for importada duas vezes por engano? A checagem de
  protocolo duplicado (escopada pela SECRETARIA inteira, não só a unidade) rejeita cada linha
  repetida individualmente — confirmado via teste real: reimportar a planilha piloto inteira
  (175 linhas) resulta em 175 erros de "protocolo já existe", zero linha nova criada.
- O que acontece com uma linha que tem local preenchido mas todos os outros campos obrigatórios
  ausentes (nome, protocolo, data/horário)?
- O que acontece se duas linhas do mesmo arquivo tiverem o mesmo protocolo?
- O que acontece se o nome do local ou do serviço vier com variação de formatação (espaços
  extras, maiúscula/minúscula, acentuação) em relação a um já existente? A comparação normaliza
  esses casos pra não criar unidades/grades duplicadas por diferença puramente cosmética — ver
  skill `importacao-planilha`.
- A coluna "Atendido por" é reconhecida pelo leitor da planilha, mas não é mapeada pra nenhum
  campo do sistema ainda (sempre vazia nos exports vistos até agora) — quem atendeu de fato é
  resolvido pela última linha de `chamadas` daquele agendamento, não por essa coluna.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema DEVE permitir que o gestor analise uma planilha (sem gravar nada) antes
  de confirmar a importação de verdade, mostrando um resumo de linhas válidas/canceladas/com
  erro, agendamentos por dia, e unidades/grades que seriam criadas.
- **FR-001a**: O sistema DEVE permitir que o gestor confirme a importação depois de ver o
  preview, reenviando o mesmo arquivo pra gravação — os números da gravação real DEVEM bater com
  os do preview pro mesmo arquivo (nenhum dado muda entre as duas fases).
- **FR-002**: O sistema DEVE identificar, pra cada linha da planilha, a qual unidade ela
  pertence, usando a informação de local presente na própria linha, comparando de forma
  insensível a espaços extras, maiúscula/minúscula e acentuação.
- **FR-002a**: Quando o local de uma linha não corresponder a nenhuma unidade já cadastrada, o
  sistema DEVE criar uma nova unidade automaticamente com esse nome, associada à secretaria do
  lote, e usá-la para essa e quaisquer outras linhas da mesma importação (ou de importações
  futuras) com o mesmo nome de local — local não cadastrado deixa de ser um erro de linha.
- **FR-002b**: O sistema DEVE identificar, pra cada linha, a qual **grade** (serviço) ela
  pertence DENTRO da unidade já resolvida, usando o texto da coluna "Serviço" como chave de
  roteamento (comparação normalizada, igual à de local).
- **FR-002c**: Quando o serviço de uma linha não corresponder a nenhuma grade já cadastrada
  dentro daquela unidade, o sistema DEVE criar uma grade nova automaticamente, com o nome vindo
  da coluna "Grade de horários" (caindo de volta pro texto do serviço se essa coluna vier vazia)
  e um guichê padrão já configurado — sem travar a importação esperando cadastro manual prévio.
- **FR-002d**: O mesmo texto de serviço em unidades DIFERENTES DEVE resultar em grades distintas,
  uma por unidade — a chave de roteamento de uma grade é sempre o par (unidade, serviço), nunca
  o serviço isolado.
- **FR-003**: O sistema DEVE atribuir cada agendamento importado à fila da unidade E grade
  correspondentes, disponível imediatamente para o fluxo operacional daquela grade.
- **FR-004**: O sistema DEVE processar cada linha de forma independente: uma linha inválida não
  pode impedir o processamento das demais linhas válidas do mesmo arquivo.
- **FR-005**: O sistema DEVE reportar, para cada linha rejeitada, o número da linha e o motivo
  específico da rejeição (campo obrigatório ausente — incluindo local vazio — ou protocolo
  duplicado; local ou serviço preenchidos mas não encontrados não são mais motivo de rejeição,
  ver FR-002a/FR-002c).
- **FR-006**: O sistema DEVE reportar, ao final da importação (ou do preview), um resumo com
  total de linhas processadas, total de agendamentos válidos/criados, total de cancelados, total
  de erros, quebra por dia, e lista de unidades/grades novas identificadas.
- **FR-007**: O sistema DEVE rejeitar uma linha cujo identificador de agendamento (protocolo) já
  exista no sistema em QUALQUER unidade da mesma secretaria (não só na mesma unidade da linha),
  evitando duplicar o mesmo agendamento em reimportações.
- **FR-008**: O sistema DEVE registrar um agendamento cancelado pelo cidadão antes da chegada com
  um status distinto de "aguardando chegada" e de "ausente".
- **FR-009**: O sistema NÃO DEVE exibir um agendamento cancelado antes da chegada nas telas de
  recepção (lista de confirmação de chegada) nem na sala de espera dos atendentes.
- **FR-010**: O sistema DEVE manter o registro de um agendamento cancelado disponível pra consulta
  e relatórios futuros, mesmo não participando do fluxo de atendimento.

### Key Entities

- **Agendamento**: já definido em `specs/001-fila-atendimento-presencial` — este spec adiciona a
  origem (importação), o status de cancelado prévio, e o vínculo com a grade resolvida.
- **Unidade**: local físico de atendimento — a primeira chave de roteamento desta funcionalidade.
- **Grade**: a fila de um serviço específico dentro de uma unidade (ver
  `specs/001-fila-atendimento-presencial`) — a segunda chave de roteamento, resolvida a partir de
  "Serviço" (chave) e "Grade de horários" (nome de exibição).
- **Lote de importação**: um evento de upload de planilha, pertencente à secretaria (não a uma
  unidade específica, já que cobre várias), com o resumo de resultado (linhas, unidades/grades
  tocadas, erros) — o preview usa a mesma lógica de resolução sem persistir o lote.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Um gestor consegue importar agendamentos de qualquer número de unidades e serviços
  da sua secretaria numa única operação de upload, sem nenhuma etapa manual de separação por
  local/serviço nem de cadastro prévio de unidade ou grade nova.
- **SC-002**: Uma planilha com linhas inválidas tem 100% das linhas válidas importadas com
  sucesso, independente da quantidade ou posição das linhas inválidas no arquivo.
- **SC-003**: Todo erro de importação identifica a linha exata e o motivo, permitindo ao gestor
  corrigir e reimportar sem precisar adivinhar o que falhou.
- **SC-004**: Nenhum agendamento cancelado antes da chegada aparece nas telas operacionais de
  recepção ou atendimento.
- **SC-005**: Reimportar a mesma planilha (ou uma sobreposta) nunca duplica um agendamento já
  existente, em qualquer unidade da secretaria.
- **SC-006**: O resultado mostrado no preview (antes de gravar) é idêntico ao resultado da
  gravação real do mesmo arquivo, dando ao gestor confiança total de que o que ele vê é o que
  vai acontecer.

## Assumptions

- Unidades e grades **não** precisam estar pré-cadastradas — a importação cria automaticamente
  qualquer unidade ou grade nova encontrada via nome de local/serviço. Cadastro manual continua
  existindo como capacidade de gestão separada, mas não é mais pré-requisito da importação.
- Local **vazio** (campo ausente, não só nome diferente) continua sendo erro de linha — a criação
  automática só se aplica a um nome preenchido que não bate com nada existente. Serviço vazio cai
  no bucket "Geral" em vez de ser rejeitado.
- O formato e as colunas reais da planilha (incluindo qual coluna identifica o local, o serviço e
  o nome da grade) estão documentados na skill `importacao-planilha`, não repetidos aqui por
  serem detalhe de implementação, não de comportamento observável pelo usuário.
- A distinção entre "cancelado" e "ausente" já é reconhecida pelo fluxo de atendimento
  (`specs/001-fila-atendimento-presencial`) — este spec só garante que o cancelado nunca chega a
  entrar nesse fluxo.
- **Reagendamento disfarçado de cancelamento**: por enquanto, importado sem distinção de um
  cancelamento de verdade — explicitamente **não** deve ser tratado como equivalente pra métricas
  futuras de não comparecimento/cancelamento; a forma de separar os dois casos fica pra uma
  discussão futura, fora deste spec.
- Reimportação/atualização de um agendamento já existente (além de rejeitar duplicata) não faz
  parte deste spec — hoje a linha duplicada é só rejeitada, não atualizada.
- A coluna "Atendido por" da planilha é reconhecida mas não mapeada — fora de escopo até que
  apareça preenchida em exports reais.
