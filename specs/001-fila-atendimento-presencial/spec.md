# Feature Specification: Fluxo de atendimento presencial (recepção → sala de espera → atendente)

**Feature Branch**: `001-fila-atendimento-presencial`

**Created**: 2026-09-19

**Last updated**: 2026-09-24

**Status**: Implementado (piloto)

**Input**: User description original (19/09): "Fluxo operacional do painel de chamada: telas de Gestor (dashboard com total do dia filtrável por unidade/CRAS, contadores de atendidos/ausentes, atendidos por atendente, gráficos; gerenciar SLA por unidade — sla_chegada_minutos e sla_atendimento_minutos — e quantidade de guichês; importar planilha do dia com separação automática por unidade), Recepção (painel do dia organizado por agendados, confirmação de chegada que move pra sala de espera, cadastro de encaixe, contagem regressiva visual quando o horário previsto chega, ausência automática se a janela de chegada estourar) e Atendente (duas abas: Sala de Espera compartilhada pela unidade ordenada por prioridade no_horario > atrasado > encaixe, e Atendimento pessoal do atendente com rechamada e marcação de atendido dentro da janela de SLA de atendimento)."

**Mudança de modelo desde a versão original (23/09, ver skill `modelo-dados`)**: SLA e guichês
deixaram de ser configuração da UNIDADE e passaram a ser configuração da **grade** — uma unidade
física (ex. um CRAS) pode oferecer mais de um serviço ao mesmo tempo, cada um com sua própria
fila, SLA e guichês independentes. A recepção continua trabalhando a unidade inteira (confirma
chegada de qualquer serviço daquele local); o atendente trabalha por grade, e pode estar alocado
em mais de uma ao mesmo tempo — ver User Story 8.

**Antigo "fora de escopo, de propósito"**: a versão original desta seção excluía "gestão de
atendentes" e "o desenho do painel de TV público". O painel de TV continua fora deste spec (ver
`specs/003-painel-tv`). Gestão de atendentes deixou de ser exclusão — foi implementada (criar/
editar/alocar/desativar/reativar atendente e recepcionista) e está coberta pela User Story 9.
A hierarquia Admin → Prefeitura → Secretaria (multi-tenant da plataforma) também não é coberta
por este spec — ver skill `papeis-e-telas`.

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Recepção confirma chegada e organiza a sala de espera (Priority: P1)

Uma pessoa com agendamento chega na unidade. A recepção confirma a chegada dela no sistema, que a envia pra sala de espera com a prioridade correta (quem chegou dentro da janela de tolerância entra com prioridade normal; quem chegou depois entra com prioridade reduzida). A tela de recepção é dividida em três abas — **Recepção** (aguardando chegada), **Sala de espera** (já confirmados, com opção de desfazer) e **Faltaram** (marcados ausentes automaticamente hoje, só consulta) — com busca por nome/protocolo/CPF.

**Why this priority**: é o ponto de entrada de todo o fluxo — sem isso, ninguém chega à sala de espera e o atendente não tem quem chamar. É a fundação de todas as outras histórias.

**Independent Test**: pode ser testado sozinho verificando que, após a recepção confirmar duas chegadas (uma dentro e uma fora da janela de tolerância), a sala de espera resultante mostra as duas pessoas na ordem correta.

**Acceptance Scenarios**:

1. **Given** um agendamento com horário previsto às 10:00 e janela de tolerância de 20 minutos, **When** a recepção confirma a chegada às 10:10, **Then** o agendamento entra na sala de espera com prioridade normal.
2. **Given** o mesmo agendamento, **When** a recepção confirma a chegada às 10:35, **Then** o agendamento entra na sala de espera com prioridade reduzida (atrás de quem chegou dentro da janela).
3. **Given** uma pessoa sem agendamento prévio, **When** a recepção cadastra um encaixe (escolhendo a grade/serviço, se a unidade tiver mais de uma), **Then** essa pessoa entra na sala de espera sem horário previsto, ordenada por ordem de chegada.
4. **Given** uma chegada confirmada por engano, **When** a recepção aciona "puxar de volta" na aba Sala de espera, **Then** o agendamento volta pra aba Recepção como se a chegada nunca tivesse sido confirmada (chegada/prioridade zeradas) — só funciona enquanto o status ainda for `sala_espera` (se já foi chamado, não reverte mais) e nunca pra um encaixe (que nunca teve etapa de "aguardando chegada").
5. **Given** um agendamento marcado ausente automaticamente hoje, **When** a recepção abre a aba Faltaram, **Then** ele aparece ali, ordenado pelo mais recente primeiro, só como consulta (sem ação de reabrir).

---

### User Story 2 - Atendente chama o próximo da sua grade (Priority: P2)

O atendente vê a sala de espera da(s) grade(s) em que está alocado, já ordenada pela prioridade correta, e chama a próxima pessoa no guichê escolhido.

**Why this priority**: é a ação central do produto — chamar alguém é o motivo do painel existir. Depende da História 1 já ter colocado alguém na sala de espera.

**Independent Test**: com pelo menos duas pessoas na sala de espera em ordens de prioridade diferentes, o atendente aciona "chamar próximo" e o sistema seleciona a pessoa de maior prioridade, não a de menor.

**Acceptance Scenarios**:

1. **Given** a sala de espera com uma pessoa de prioridade normal e uma de prioridade reduzida, **When** o atendente chama o próximo, **Then** a pessoa de prioridade normal é chamada primeiro.
2. **Given** dois atendentes da mesma grade chamando ao mesmo tempo, **When** ambos acionam "chamar próximo" simultaneamente, **Then** cada um recebe uma pessoa diferente — nenhuma pessoa é chamada duas vezes (`FOR UPDATE SKIP LOCKED`).
3. **Given** uma pessoa chamada, **When** a chamada é confirmada, **Then** ela aparece no bloco compartilhado "Chamando agora" (visível a todos os atendentes daquela grade, com o nome de quem chamou e o guichê) — só quem chamou (o "dono") pode agir nela.
4. **Given** um atendente que ainda não escolheu um guichê numa grade, **When** ele tenta chamar o próximo, **Then** o sistema pede a escolha do guichê primeiro (modal por grade, não uma tela cheia bloqueando as outras grades já resolvidas) — um guichê já ocupado por outro atendente (heartbeat de até 15s) aparece desabilitado com o nome de quem está nele.

---

### User Story 3 - Atendente gerencia o atendimento em andamento, com posse e cooldown (Priority: P3)

Depois de chamar alguém, o atendente espera a pessoa chegar ao guichê. Se necessário, rechama — só depois de um tempo mínimo configurável desde a última chamada (cooldown). Quando a pessoa chega, marca como atendida; se alguém avisar que ela já foi embora, pode marcar ausência manualmente, sem esperar o SLA inteiro.

**Why this priority**: fecha o ciclo de vida do atendimento; sem isso, o atendente não tem como concluir ou repetir uma chamada, e o histórico de atendidos nunca é preenchido.

**Independent Test**: chamar uma pessoa, rechamar depois do cooldown, marcar como atendida, e confirmar que ela aparece no histórico de atendidos do atendente com o horário correto.

**Acceptance Scenarios**:

1. **Given** uma pessoa chamada, **When** o atendente tenta rechamar antes do cooldown configurado da grade (padrão 1 minuto) ter passado, **Then** o sistema recusa (409) e mostra quanto falta.
2. **Given** o cooldown já passado, **When** o atendente rechama, **Then** um novo registro de chamada é criado e o painel volta a exibir essa pessoa como a chamada mais recente.
3. **Given** o cooldown já passado e o atendente decide chamar outra pessoa em vez de rechamar, **When** ele chama alguém novo, **Then** a pessoa anterior vira um **chamado órfão** (visível a todos, com indicador visual) — nenhum atendente ocioso daquela grade (incluindo o próprio dono) consegue puxar gente nova da sala de espera daquela grade até alguém **assumir** o órfão.
4. **Given** um chamado órfão, **When** outro atendente da mesma grade o assume pro próprio guichê, **Then** a posse passa pra ele (novo registro de chamada, `guiche_id` atualizado) e só ele pode agir nessa pessoa a partir daí.
5. **Given** uma pessoa chamada que chegou ao guichê, **When** o atendente marca como atendida, **Then** o status muda pra atendido, o horário é registrado, e ela aparece no histórico de atendidos daquele atendente especificamente (não no de outros atendentes da mesma grade).
6. **Given** uma pessoa chamada que avisou (ou foi avisado) que já foi embora, **When** o dono marca ausência manualmente, **Then** o status muda pra ausente na hora, sem esperar o SLA de atendimento estourar.
7. **Given** uma chamada pendente ou um chamado órfão numa grade, **When** o mesmo atendente tem uma OUTRA grade alocada, **Then** ele consegue chamar/agir normalmente na outra grade — o bloqueio de posse/cooldown/órfão é sempre por grade, nunca pelo atendente como um todo.

---

### User Story 4 - Ausência automática por estouro de janela, em qualquer estágio (Priority: P4)

Uma pessoa com horário agendado não comparece à recepção dentro da janela de tolerância de chegada, **ou** uma pessoa já chamada não aparece no guichê dentro da janela de atendimento (contada da PRIMEIRA chamada, não da última — rechamar ou "assumir" não dão tempo extra). Em ambos os casos, o sistema marca automaticamente como ausente, sem exigir uma ação manual.

**Why this priority**: protege o fluxo de acumular atendimentos "esquecidos" indefinidamente — seja em aguardando-chegada, seja em chamado — mantendo o painel da recepção, a sala de espera e os indicadores do gestor confiáveis.

**Independent Test**: criar um agendamento com horário previsto no passado e nenhuma confirmação de chegada, esperar a janela configurada passar, e verificar que o status muda pra ausente sem qualquer ação humana; separadamente, chamar um atendimento e não marcá-lo como atendido dentro da janela de atendimento, e verificar o mesmo resultado.

**Acceptance Scenarios**:

1. **Given** um agendamento com horário previsto às 09:00 e janela de tolerância de chegada de 60 minutos, **When** o relógio chega a 10:00 sem confirmação de chegada, **Then** o agendamento muda automaticamente pra ausente.
2. **Given** um agendamento do tipo encaixe (sem horário previsto), **When** o tempo passa, **Then** ele nunca é marcado ausente automaticamente por essa regra (não se aplica a encaixe).
3. **Given** um atendimento chamado às 14:00 e janela de atendimento de 10 minutos (padrão atual), **When** o relógio chega a 14:10 sem a pessoa ter sido marcada como atendida (nem manualmente ausente), **Then** o atendimento muda automaticamente pra ausente, liberando o atendente pra chamar o próximo — mesmo que tenha havido uma rechamada ou uma troca de dono ("assumir") no meio do caminho, o prazo não é resetado.

---

### User Story 5 - Gestor acompanha o dia pelo dashboard (Priority: P5)

O gestor abre o dashboard e vê o panorama do dia da secretaria inteira (todas as unidades e grades dela): total de agendamentos, quantos foram atendidos, quantos ficaram ausentes, quebra por atendente, gráfico de linha atendido-vs-ausente dos últimos 7 dias, e proporção atendido-vs-ausente por grade.

**Why this priority**: é a visão gerencial que justifica o investimento no sistema, mas não bloqueia o funcionamento operacional do dia a dia (as Histórias 1–4 funcionam sem essa tela existir).

**Independent Test**: com um dia de dados já processados pelas histórias anteriores, abrir o dashboard e conferir que os contadores batem com os registros reais.

**Acceptance Scenarios**:

1. **Given** um dia com atendimentos concluídos e ausências em mais de uma grade da secretaria, **When** o gestor abre o dashboard, **Then** os contadores totais somam todas as unidades/grades dela.
2. **Given** o mesmo dia, **When** o gestor consulta a quebra por atendente, **Then** cada atendente aparece com a contagem de pessoas que ele mesmo atendeu, em qualquer grade que trabalhe.
3. **Given** os últimos 7 dias com movimento variável, **When** o gestor olha o gráfico de linha, **Then** dias sem nenhum atendimento/ausência aparecem com zero (não ficam ausentes do eixo).

---

### User Story 6 - Gestor importa a planilha do dia (Priority: P6)

O gestor sobe a planilha de agendamentos do dia (fluxo em duas etapas: analisa sem gravar → confirma), e o sistema separa automaticamente os registros pra cada unidade **e grade** correspondente.

**Why this priority**: é a origem dos dados que alimentam todo o fluxo, mas o MVP também aceita cadastro manual/encaixe como caminho alternativo — não é estritamente bloqueante pras Histórias 1–4 num piloto pequeno.

**Independent Test**: subir uma planilha com registros de mais de uma unidade e confirmar que cada agendamento aparece na fila da unidade e grade corretas, sem intervenção manual de separação.

**Acceptance Scenarios**:

1. **Given** uma planilha com agendamentos de duas unidades diferentes e mais de um serviço, **When** o gestor confirma a importação (depois de ver o preview), **Then** cada agendamento é atribuído automaticamente à sua unidade e grade corretas — grade nova é criada automaticamente se ainda não existir.
2. **Given** uma planilha com uma linha inválida (dado obrigatório faltando) ou um protocolo já existente na secretaria, **When** o gestor importa, **Then** o sistema reporta o erro daquela linha especificamente, sem descartar as linhas válidas do restante do arquivo.
3. Ver `specs/002-importacao-multi-local` pro detalhe completo do roteamento automático.

---

### User Story 7 - Gestor configura SLA e guichês da grade (Priority: P7)

O gestor ajusta, por **grade** (não mais por unidade), o tempo de tolerância de chegada, o tempo padrão de atendimento, o cooldown de rechamada, as durações de exibição do painel, a permissão de encaixe pela recepção, e a quantidade de guichês/salas disponíveis — inclusive copiando as predefinições de outra grade já configurada.

**Why this priority**: personalização operacional — tem um padrão razoável de fábrica, então não bloqueia o uso inicial do piloto.

**Independent Test**: alterar o tempo de tolerância de uma grade e confirmar que um novo agendamento criado depois da mudança respeita o novo valor, enquanto agendamentos já processados antes da mudança mantêm o resultado anterior.

**Acceptance Scenarios**:

1. **Given** uma grade com tempo de tolerância padrão, **When** o gestor altera esse valor, **Then** o novo valor se aplica imediatamente (inclusive a chamados já em andamento, que são sempre recalculados na hora, não congelados no momento da chamada).
2. **Given** uma grade com dois guichês, **When** o gestor adiciona um terceiro, **Then** o novo guichê fica disponível pra atendentes daquela grade.
3. **Given** uma grade ou guichê excluído por engano, **When** o gestor abre a aba "Excluídas"/lista de guichês e reativa, **Then** o registro volta a operar normalmente (uma grade reativada não recupera automaticamente quem estava alocado nela antes — precisa realocar).
4. **Given** uma grade com um único guichê, **When** o gestor tenta excluí-lo, **Then** o sistema recusa (409) — toda grade precisa de pelo menos um guichê ou sala.

---

### User Story 8 - Atendente/recepcionista alocado em mais de uma grade, sem selecionar nenhuma (Priority: P8)

Um atendente pode estar alocado em várias grades ao mesmo tempo, inclusive de unidades físicas diferentes (ex.: uma grade na Unidade A de manhã e outra na Unidade B à tarde). Ele nunca escolhe "em qual grade vai trabalhar" — vê e opera todas, consolidadas, o tempo todo; só escolhe um guichê por grade, sob demanda. Uma recepcionista pode ter zero, uma ou mais grades marcadas — sem nenhuma marcada, ela continua vendo/cadastrando encaixe em qualquer grade da própria unidade (as grades marcadas viram um filtro, nunca uma restrição a menos que preenchidas); ela só pode ser alocada a grades da própria unidade fixa (nunca de outra).

**Why this priority**: reflete a realidade operacional real do piloto (confirmada com o dono do produto) — sem isso, um atendente com duas frentes de trabalho precisaria de duas contas, ou perderia visibilidade de uma das duas.

**Independent Test**: alocar um atendente em duas grades de unidades diferentes; confirmar que ele vê as duas simultaneamente, agrupadas por unidade na tela, sem nenhuma tela de seleção antes.

**Acceptance Scenarios**:

1. **Given** um atendente recém-alocado em duas grades, **When** ele faz login, **Then** vê as duas seções de sala de espera lado a lado (agrupadas por unidade, se forem de unidades diferentes), sem escolher nenhuma antes.
2. **Given** um atendente já com a tela aberta, **When** um gestor/admin adiciona ou remove uma grade dele, **Then** a mudança aparece sozinha em poucos segundos (revalidação periódica da sessão) — nunca só depois de logout/login ou F5 manual.
3. **Given** uma chamada pendente numa das grades do atendente, **When** ele opera a outra grade, **Then** não há nenhum bloqueio cruzado (ver User Story 3, cenário 7).

---

### User Story 9 - Gestor/admin gerencia usuários internos: criar, editar, excluir e reativar (Priority: P9)

O gestor cria/edita atendentes e recepcionistas da própria secretaria (nome, email, senha, cargo, grades alocadas); o admin faz o mesmo pra qualquer secretaria da plataforma, além de criar gestores. Excluir um usuário é reversível — ele desaparece da operação mas continua visível numa aba "Inativos" com um botão de reativar.

**Why this priority**: sem isso, o cadastro de quem opera o sistema dependeria de acesso direto ao banco — inviável pra um piloto real com equipe própria.

**Independent Test**: criar um atendente novo, confirmar que ele consegue logar e ver as grades certas; excluí-lo, confirmar que ele não consegue mais logar mas continua listado como inativo; reativá-lo e confirmar que volta a conseguir logar com as mesmas grades de antes.

**Acceptance Scenarios**:

1. **Given** um gestor autenticado, **When** ele cria um atendente com uma ou mais grades da própria secretaria, **Then** o novo usuário consegue logar e já vê essas grades.
2. **Given** um usuário excluído, **When** alguém tenta logar com essa conta, **Then** o acesso é recusado (403 `usuario_desativado`).
3. **Given** um usuário excluído, **When** o gestor/admin abre a aba "Inativos" e reativa, **Then** o usuário volta a conseguir logar, com as mesmas grades alocadas de antes (a exclusão não mexe nas alocações).
4. **Given** um gestor autenticado, **When** ele tenta mudar o próprio cargo, **Then** o sistema recusa (403) — só pode mudar nome/email/senha da própria conta.

---

### Edge Cases

- Todo limite de tempo deste spec (janela de chegada, janela de atendimento, cooldown de rechamada, classificação de prioridade) é **exclusivo**: ao bater o minuto exato do limite, a janela já é considerada estourada, não há tolerância adicional no instante do limite.
- Uma pessoa é chamada mas não aparece dentro da janela de atendimento configurada da grade → vira **ausente automaticamente**, mesmo padrão da ausência por falta de chegada (não exige ação manual do atendente) — o prazo é sempre contado da PRIMEIRA chamada, nunca resetado por rechamada ou troca de dono.
- Na sala de espera, um agendado "atrasado" e um encaixe competem pela mesma posição: quem chegou primeiro é chamado primeiro, independente de um ter sido originalmente agendado e o outro não (decisão do dono do produto: quem se atrasa já recebeu a tolerância da janela de chegada; não faz sentido furar a frente de um encaixe que chegou antes).
- Um guichê que o gestor tenta excluir sendo o último da grade é recusado (409 `ultimo_guiche`) — checagem e exclusão acontecem na mesma query, sem condição de corrida.
- Um guichê ocupado por um atendente que perde a alocação daquela grade (realocado por um admin/gestor) não precisa de limpeza explícita — a ocupação é lida com uma janela de frescor de 15 segundos e se autocorrige sozinha.
- Uma grade excluída tem suas alocações de usuário (`usuario_grades`) limpas na hora da exclusão — reativar a grade não as recupera automaticamente.
- Uma recepcionista nunca pode ser alocada a uma grade de uma unidade diferente da sua (validado no servidor, não só na tela) — o vínculo ficaria "morto" (nunca usado, já que a lista de grades dela é só um filtro dentro da própria unidade).

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: O sistema DEVE permitir que a recepção veja a lista de agendamentos do dia da sua unidade, organizada em abas (Recepção/aguardando chegada, Sala de espera/já confirmados, Faltaram/ausentes automáticos de hoje), com busca por nome/protocolo/CPF.
- **FR-002**: O sistema DEVE destacar visualmente um agendamento cujo horário previsto já chegou e ainda não teve a chegada confirmada.
- **FR-003**: O sistema DEVE permitir que a recepção confirme a chegada de um agendado, registrando o horário da confirmação e movendo o registro pra sala de espera; e DEVE permitir desfazer essa confirmação enquanto o status ainda for `sala_espera` (nunca depois de chamado, nunca pra um encaixe).
- **FR-004**: O sistema DEVE classificar automaticamente cada chegada confirmada em "prioridade normal" ou "prioridade reduzida", comparando o horário de chegada contra o horário previsto mais a tolerância configurada **da grade** daquele agendamento.
- **FR-005**: O sistema DEVE preservar essa classificação de prioridade mesmo que a tolerância da grade seja alterada depois.
- **FR-006**: O sistema DEVE permitir que a recepção cadastre um atendimento sem agendamento prévio (encaixe), escolhendo a grade/serviço quando a unidade tiver mais de uma (respeitando a permissão de encaixe por grade — FR-023), entrando direto na sala de espera.
- **FR-007**: O sistema DEVE marcar automaticamente como ausente qualquer agendamento cuja janela de tolerância de chegada se esgote sem confirmação — sem exigir ação manual de nenhum papel.
- **FR-008**: O sistema NÃO DEVE aplicar a ausência automática por falta de chegada a atendimentos do tipo encaixe (que não têm horário previsto).
- **FR-009**: O sistema DEVE exibir a sala de espera de uma grade compartilhada entre todos os atendentes alocados naquela grade, nunca exibindo quem ainda não confirmou chegada.
- **FR-010**: O sistema DEVE ordenar a sala de espera em dois grupos: (1) agendados de prioridade normal primeiro, por ordem de horário previsto; (2) todo o restante — agendados de prioridade reduzida e encaixes, **misturados numa única fila**, ordenados por horário de chegada.
- **FR-010a**: Todo limite de tempo usado nas regras deste spec DEVE ser tratado como exclusivo — ao atingir o minuto exato do limite, a janela já conta como estourada.
- **FR-011**: O sistema DEVE permitir que um atendente chame o próximo da sala de espera de uma grade em que está alocado, garantindo que, sob chamadas simultâneas de atendentes diferentes, cada agendamento seja entregue a exatamente um atendente.
- **FR-012**: O sistema DEVE, após uma chamada, exibir o atendimento num bloco compartilhado "Chamando agora" visível a todos os atendentes daquela grade, indicando quem é o dono (só ele pode agir).
- **FR-013**: O sistema DEVE permitir que o dono rechame a mesma pessoa, mas só depois de um cooldown configurável (por grade) desde a última chamada — antes disso, a ação é recusada com o tempo restante.
- **FR-013a**: O sistema DEVE permitir que o dono chame outra pessoa (abandonando a atual) só depois do mesmo cooldown ter passado; a pessoa abandonada vira um **chamado órfão**, visível a todos, bloqueando qualquer atendente ocioso daquela grade (inclusive o próprio antigo dono) de puxar gente nova da sala de espera daquela grade até alguém assumir o órfão.
- **FR-013b**: O sistema DEVE permitir que qualquer atendente alocado na grade assuma um chamado órfão pro próprio guichê, transferindo a posse (nova chamada registrada, dono e guichê atualizados).
- **FR-014**: O sistema DEVE registrar cada chamada, rechamada e transferência de posse individualmente, incluindo quem a realizou, o guichê e quando — formando o histórico de chamadas de um atendimento.
- **FR-015**: O sistema DEVE permitir que o dono marque um atendimento em andamento como concluído, registrando o horário da conclusão.
- **FR-015a**: O sistema DEVE marcar automaticamente como ausente um atendimento chamado cuja janela de atendimento (contada da PRIMEIRA chamada, nunca resetada por rechamada/transferência) se esgote sem a pessoa ser marcada como atendida — sem exigir ação manual.
- **FR-015b**: O sistema DEVE permitir que o dono marque manualmente um atendimento chamado como ausente, sem esperar a janela de atendimento estourar.
- **FR-016**: O sistema DEVE manter um histórico de atendidos por atendente, mostrando só os atendimentos concluídos por aquele atendente específico, em qualquer grade que ele trabalhe.
- **FR-017**: O sistema DEVE oferecer uma visão de linha do tempo de um atendimento específico, mostrando horário agendado, horário de chegada, todos os horários de chamada/rechamada/transferência, e horário de conclusão, quando disponíveis.
- **FR-018**: O sistema DEVE permitir que o gestor visualize, pra secretaria inteira e para o dia corrente: total de agendamentos, total atendido, total ausente, quebra de atendidos por atendente, série diária dos últimos 7 dias e proporção atendido-vs-ausente por grade.
- **FR-019**: O sistema DEVE permitir que o gestor analise (sem gravar) e depois confirme a importação de uma planilha de agendamentos do dia, atribuindo automaticamente cada linha à sua unidade e grade correspondentes (criando a grade se ainda não existir), sem exigir separação manual.
- **FR-020**: O sistema DEVE reportar, na importação da planilha, erros específicos por linha (incluindo protocolo duplicado na secretaria), sem descartar as linhas válidas do restante do arquivo.
- **FR-021**: O sistema DEVE permitir que o gestor configure, por grade, o tempo de tolerância de chegada, o tempo padrão de atendimento, o cooldown de rechamada, as durações de exibição do painel e a permissão de encaixe pela recepção — inclusive copiando de outra grade já configurada.
- **FR-022**: O sistema DEVE permitir que o gestor gerencie a quantidade de guichês/salas disponíveis numa grade, com pelo menos um sempre ativo e não-excluído.
- **FR-023**: O sistema DEVE permitir que o gestor restrinja, por grade, se a recepção pode cadastrar encaixe nela (gestor/admin sempre podem, independente dessa configuração).
- **FR-024**: O sistema DEVE permitir que um atendente ou recepcionista esteja alocado em zero (só recepcionista), uma ou várias grades ao mesmo tempo, inclusive de unidades físicas diferentes (só atendente — recepcionista fica restrita à própria unidade), sem exigir seleção de qual grade "usar" — vê e opera todas simultaneamente.
- **FR-025**: O sistema DEVE propagar uma mudança de alocação de grade (feita por um gestor/admin) pra sessão de um atendente já logado em poucos segundos, sem exigir logout/login ou recarregamento manual da página.
- **FR-026**: O sistema DEVE permitir criar, editar (nome/email/senha/cargo/grades) e excluir (soft-delete) um usuário interno (gestor cria só na própria secretaria; admin, em qualquer uma), e DEVE permitir reativar um usuário excluído, restaurando o acesso com as mesmas alocações de antes.
- **FR-027**: O sistema DEVE permitir excluir (soft-delete) uma grade ou um guichê e reativar depois — uma grade reativada não restaura automaticamente as alocações de usuário que tinha antes de ser excluída.
- **FR-028**: O sistema NÃO DEVE permitir que um gestor altere o próprio cargo.

### Key Entities

- **Agendamento**: um atendimento previsto ou não (encaixe) pra uma pessoa, numa unidade e numa **grade**. Guarda identificação da pessoa (nome, CPF/telefone quando disponíveis, sem máscara — decisão de produto), horário previsto (quando aplicável), horário de chegada, classificação de prioridade, status ao longo do ciclo de vida, e horários de conclusão/ausência.
- **Chamada**: um evento de chamada, rechamada ou transferência de posse de um agendamento por um atendente específico, num guichê específico, com horário — a sequência dessas chamadas forma o histórico de um atendimento e determina quem é o "dono" atual (a mais recente).
- **Unidade**: um local físico de atendimento (ex: um CRAS) — só identidade/local; não guarda mais SLA/guichês diretamente.
- **Grade**: a fila de um serviço específico dentro de uma unidade (ex. "CadÚnico" e "Bolsa Família" na mesma unidade seriam duas grades distintas) — dona do SLA de chegada/atendimento, cooldown de rechamada, durações de exibição do painel, permissão de encaixe pela recepção, e do conjunto de guichês. Pode ser desativada (reversível) ou excluída (soft-delete, reversível via reativação, mas perde as alocações de usuário).
- **Guichê**: um ponto de atendimento físico ou lógico (guichê ou sala) dentro de uma grade, com ocupação em tempo real (heartbeat) por um atendente.
- **Usuário interno**: gestor, atendente ou recepcionista — atendente e recepcionista têm uma relação N:N com grades (`usuario_grades`); recepcionista também tem uma unidade fixa. Pode ser excluído (soft-delete via `ativo=false`) e reativado.
- **Lote de importação**: o registro de uma planilha de agendamentos importada, com o resultado (linhas processadas, canceladas, erros).

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Uma pessoa que chega dentro da janela de tolerância nunca perde a prioridade pra quem chegou atrasado, em nenhuma situação observável na sala de espera.
- **SC-002**: Nenhum agendamento é chamado por mais de um atendente ao mesmo tempo, mesmo sob uso concorrente real da tela de sala de espera.
- **SC-003**: Um agendamento sem chegada confirmada, ou uma chamada sem conclusão, nunca fica pendente além da janela configurada — a transição pra ausente acontece sem intervenção humana em 100% dos casos.
- **SC-004**: O gestor consegue obter o panorama do dia da secretaria inteira (total, atendidos, ausentes, por atendente, por grade) em uma única tela, sem precisar cruzar dados manualmente.
- **SC-005**: Uma planilha com agendamentos de múltiplas unidades e serviços é importada sem que o gestor precise separar manualmente nenhuma linha por unidade ou grade.
- **SC-006**: Qualquer atendimento do dia pode ter sua linha do tempo completa (agendado → chegada → chamada(s)/transferências → conclusão) reconstruída a partir do sistema, sem depender de anotação externa.
- **SC-007**: Um atendente alocado em várias grades nunca precisa escolher qual usar pra ver ou trabalhar em qualquer uma delas — todas aparecem consolidadas desde o primeiro acesso.
- **SC-008**: Uma exclusão (usuário, grade ou guichê) é sempre reversível pela própria interface, sem depender de acesso direto ao banco de dados.

## Assumptions

- O piloto roda com uma secretaria (SASC) e a unidade "Cadastro Único Sede Prazeres" como principal, mas o modelo já suporta múltiplas unidades e grades por secretaria, e múltiplas secretarias por prefeitura.
- Os valores padrão de tolerância de chegada (60 minutos), tempo de atendimento (10 minutos) e cooldown de rechamada (1 minuto) são os usados até o gestor configurar diferente por grade — aplicam-se só a grades novas, não retroativamente.
- A hierarquia Admin → Prefeitura → Secretaria (multi-tenant da plataforma) está fora deste spec — ver skill `papeis-e-telas`.
- O desenho do painel de TV (o que exatamente é exibido publicamente) está fora deste spec — ver `specs/003-painel-tv`.
- A autenticação de quem opera cada tela (gestor/recepcionista/atendente/admin) é login simples (email+senha) — ver skill `auth-login-mvp`. O SSO real da plataforma está adiado pro pós-MVP (skill `auth-conecta-cidades`).
- A importação de planilha em si (regras de roteamento por unidade/grade) tem o detalhe completo em `specs/002-importacao-multi-local` e deve permanecer consistente com este spec.
