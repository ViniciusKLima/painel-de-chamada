---
name: regras-negocio-fila
description: Use sempre que for implementar ou alterar o fluxo de chegada, sala de espera, chamada, ou a lógica de priorização da fila. Contém as regras de negócio definidas pelo dono do produto — não inferir regra nova sem confirmar.
---

# Fluxo completo (MVP — unidade piloto do CRAS)

**Nota (22/09)**: todo SLA/cooldown mencionado neste documento (`sla_chegada_minutos`,
`sla_atendimento_minutos`, `cooldown_rechamada_minutos`) é configurado por **grade**, não
mais por unidade — uma unidade pode ter várias grades (uma por serviço), cada uma com seu
próprio SLA e sua própria fila de "chamando agora"/órfãos. Ver skill `modelo-dados` pro
raciocínio completo. Onde este documento diz "a unidade" num contexto de SLA/fila de
chamada, leia "a grade daquele agendamento". A recepção continua trabalhando com a unidade
inteira (todas as grades) — isso não mudou.

O fluxo tem 3 papéis distintos e cada um enxerga uma fatia diferente dos dados. Isso é proposital: **o atendente nunca vê a lista completa do dia, só a sala de espera** — evita chamar alguém que ainda não chegou.

```
Recepção                              Atendente (aba Sala de espera)     Atendente (aba Atendimento)
--------                              -----------------------------     ---------------------------
Lista do dia (import CSV, gestor)
  ↓
Horário previsto chega
  ↓ (linha marcada, contagem de sla_chegada_minutos começa)
Cidadão chega → recepção confirma
  ↓ (grava chegada_em + prioridade)
Move para "sala de espera" -------->  Vê sala de espera da unidade
                                       (ordenada por prioridade)
                                         ↓
                                       Chama o próximo -----------------> Aguarda a pessoa no guichê
                                       (cria `chamadas`, painel de TV      (até sla_atendimento_minutos)
                                        exibe, status = chamado)            ↓
                                                                          Pessoa chega → marca atendido
                                                                            ↓
                                                                          Vai pro histórico do atendente
```

Se `sla_chegada_minutos` (chegada) estourar sem confirmação → `ausente` automático.
Se `sla_atendimento_minutos` (atendimento) estourar sem a pessoa aparecer no guichê depois de
chamada → `ausente` automático também, **confirmado (19/09)**: mesmo padrão do timeout de
chegada, sem exigir ação manual do atendente.

**Todo limite de tempo deste documento é exclusivo** (confirmado 19/09): ao bater o minuto exato
do limite, a janela já conta como estourada — não existe tolerância adicional no instante do
limite em nenhuma das janelas (chegada, atendimento, classificação de prioridade).

## Estados do agendamento

1. `aguardando_chegada` — importado da planilha, cidadão ainda não chegou. Só a recepção vê isso.
2. `sala_espera` — recepção confirmou a chegada. A partir daqui o atendente passa a enxergar o registro.
3. `chamado` — atendente chamou, aguardando a pessoa se apresentar no guichê.
4. `atendido` — concluído.
5. `ausente` — não compareceu — na chegada (estourou `sla_chegada_minutos` sem confirmação) ou no
   atendimento (chamado, estourou `sla_atendimento_minutos` sem aparecer) — os dois casos viram
   `ausente` automaticamente, sem ação manual.

**Regra dura**: o atendente só pode chamar quem está em `sala_espera`. Nunca deve ser possível chamar diretamente de `aguardando_chegada` — se isso for tecnicamente possível na API, é bug.

## Agendado vs. encaixe

Nem toda pessoa na fila tem horário marcado.

- **Agendado**: tem `horario_previsto`. Veio da planilha importada pela recepção.
- **Encaixe**: sem horário — recepção pode cadastrar na hora (walk-in), ou pode ser alguém que chegou sem estar na planilha do dia.

## Janela de chegada e ausência automática (recepção)

Cada unidade tem um `sla_chegada_minutos` configurável (padrão 60min, ver skill `modelo-dados`).

- Quando o relógio bate o `horario_previsto` de um agendado, a linha dele fica **marcada** na
  tela da recepção e uma contagem regressiva de `sla_chegada_minutos` começa.
- Se a contagem **zerar sem a recepção confirmar a chegada**, o agendamento vira `ausente`
  **automaticamente** — precisa de um mecanismo que rode isso sem depender de alguém abrir a
  tela (job periódico, ou verificação no momento da leitura — decisão de implementação, não de
  negócio; a regra é que aconteça sozinho, sem ação humana).
- Isso só se aplica a `tipo = 'agendado'` (encaixe não tem `horario_previsto`, não tem o que
  estourar).

## Prioridade dentro da janela de chegada (no horário vs. atrasado)

Cada unidade também tem um `sla_atendimento_minutos` (padrão 20min — é a duração de um
atendimento, e por isso também vira a régua de tolerância aqui).

Quando a recepção confirma a chegada de um agendado, compara `chegada_em` contra
`horario_previsto + sla_atendimento_minutos`:

- **Chegou dentro da janela** (até `sla_atendimento_minutos` depois do horário previsto):
  `prioridade = 'no_horario'`. Essa pessoa mantém a prioridade normal — pode ser chamada dentro
  do próprio horário agendado, como se tivesse chegado pontual.
- **Chegou depois da janela, mas antes do `sla_chegada_minutos` estourar** (senão já virou
  ausente automático, ver acima): `prioridade = 'atrasado'`. Entra na sala de espera, mas **atrás
  de quem chegou no horário** — o sistema prioriza quem foi pontual.

Gravar esse valor **uma vez**, no momento da confirmação de chegada — não recalcular depois, pra
não mudar de resultado se o SLA da unidade mudar posteriormente (ver skill `modelo-dados`).

## Regra de prioridade na sala de espera

**Confirmado (19/09), inclusive o motivo**: um agendado que chega atrasado já recebeu a
tolerância da janela de chegada (que "nem deveria precisar dar") — não faz sentido ele também
furar a frente de um encaixe que chegou antes dele. Na prática, uma vez atrasado, a pessoa
compete em pé de igualdade com quem é encaixe puro.

Ordem de exibição/chamada na sala de espera, do primeiro a ser chamado pro último:

1. Agendados com `prioridade = 'no_horario'`, ordenados por `horario_previsto` mais antigo primeiro.
2. **Todo o restante** — agendados com `prioridade = 'atrasado'` **e** encaixes, **misturados
   numa fila única**, ordenados por `chegada_em` (quem chegou primeiro é chamado primeiro,
   independente de ter sido originalmente agendado ou não).

Isso substitui as duas versões anteriores deste documento (a v1 não tinha tier `atrasado`
nenhum; a v2 colocava `atrasado` inteiro antes de todo encaixe). A versão atual é a definitiva.

## Concorrência ao chamar

Igual ao que já foi validado na versão anterior (Node): dois atendentes chamando ao mesmo tempo
nunca podem pegar o mesmo agendamento. Implementar com `SELECT ... FOR UPDATE SKIP LOCKED` (ou
equivalente) dentro de uma transaction ao buscar o próximo da sala de espera — não confiar em
checar-e-atualizar em dois passos separados.

## Ocupação de guichê em tempo real (22/09)

Pedido do dono do produto: se são 3 guichês/salas numa grade e um atendente escolhe o 2, os
outros atendentes que forem escolher guichê depois só devem ver o 1 e o 3 livres — o 2
precisa aparecer ocupado. Antes disso, a escolha de guichê só era lembrada em `localStorage`
(local a cada navegador, sem visibilidade entre atendentes) — dois atendentes diferentes
podiam escolher o mesmo guichê sem que nenhum soubesse do outro.

**Mecanismo — heartbeat, não uma trava permanente**: escolher um guichê no modal chama
`POST /grades/{gradeId}/guiches/{guicheId}/ocupar`, que grava `ocupado_por_usuario_id` +
`ocupado_em = now()`. Enquanto o atendente continua ali, o frontend **reafirma a ocupação a
cada 5s** (mesmo endpoint, idempotente pro próprio dono). Um guichê só aparece "ocupado" pros
outros enquanto o heartbeat estiver **fresco** (`ocupado_em` dentro dos últimos 15s, ver
`janelaOcupacaoGuiche` em `internal/repository/grades.go`) — depois disso, mesmo sem nenhuma
ação explícita, ele volta a aparecer livre sozinho. Isso cobre o caso de aba fechada/crash
sem logout, sem precisar de nenhuma limpeza manual ou job de background.

**Liberação explícita**: clicar "Sair" chama `POST .../liberar` antes de encerrar a sessão,
soltando o guichê na hora (não espera os até 15s da janela). Trocar de guichê (botão no
cabeçalho) não libera explicitamente o antigo — só para de mandar heartbeat pra ele, que
expira sozinho pela mesma janela.

**Conflito**: se dois atendentes tentam escolher o mesmo guichê livre ao mesmo tempo, só um
`ocupar` sucede (`UPDATE ... WHERE ocupado_por_usuario_id IS NULL OR ...` — atômico, sem
race condition de dois passos) — o outro recebe 409 `guiche_ocupado` e a lista é atualizada
na hora pra refletir o estado real.

**Escopo desta regra**: é sobre a ESCOLHA do guichê no modal (evitar dois atendentes
escolhendo o mesmo ao mesmo tempo), não uma trava adicional sobre `chamar-próximo` — esse
endpoint continua aceitando qualquer `guicheId` da grade, sem verificar se é o guichê
"ocupado" por quem está chamando. Documentando aqui pra não presumir uma trava mais rígida
que não foi pedida nem implementada.

## Rechamada, posse da chamada e chamados órfãos (reformulado 20/09)

**Reverificado de ponta a ponta em 22/09** depois de um relato do dono do produto que parecia
indicar um bug ("um atendente com chamada pendente bloqueou outro atendente de chamar
alguém"). Reproduzi o cenário exato com 4 atendentes reais e dados de teste do zero: (1) dois
atendentes com uma chamada pendente cada, NENHUMA ainda recolhida — um terceiro atendente
ocioso conseguiu chamar normalmente (201), confirmando que uma chamada pendente **só bloqueia
o próprio dono**, não trava a grade inteira; (2) depois que um dono rechamou e seguiu em
frente (chamado virou órfão de verdade), um quarto atendente **sem nenhuma pendência própria**
foi corretamente bloqueado (409 `existem_chamados_pendentes`) até assumir o órfão. Conclusão:
o comportamento já estava correto — o relato provavelmente veio de um chamado órfão real
deixado por teste anterior na mesma grade (não visível/óbvio na hora), não de um bug de
código. Documentando aqui porque a diferença entre "pendente" (só do dono) e "órfão" (trava
todo mundo) é sutil e fácil de confundir visualmente na tela.

**Não existe mais aba "Atendimento" separada.** A tela do atendente é um grid único de 3
blocos: "Pendentes"/"Chamando agora" (compartilhado entre todos os atendentes da **grade**
em que o atendente está alocado — 22/09, era "da unidade" antes da grade existir), "Sala de
espera" (compartilhada, com sub-abas "Aguardando"/"Ausentes") e "Meus atendimentos hoje"
(consulta pessoal, sem ação, com busca). Ver skill `papeis-e-telas`.

**"Chamando agora" é compartilhado, mas a ação tem dono**: depois que um atendente chama
alguém (status vira `chamado`, aparece no painel de TV, cria uma linha em `chamadas`), essa
pessoa aparece pra **todos** os atendentes daquela grade, mostrando quem chamou e por qual
guichê — mas só quem chamou por último (o "dono") pode **rechamar** ou **marcar presente**.
Pra qualquer outro atendente, a linha é só leitura (a não ser que já seja um "chamado
órfão" — ver adiante).

**Cooldown de rechamada**: cada GRADE (não mais unidade, desde que o SLA migrou pra lá) tem um
`cooldown_rechamada_minutos` (padrão 1min desde a oitava rodada de 23/09, configurável).
Rechamar só é permitido depois desse tempo desde a **última** chamada — antes disso, o botão
fica desabilitado mostrando quanto falta. O motivo: dar um tempo mínimo real pra pessoa ouvir/
reagir à chamada antes de repetir.

**Escopo por grade, confirmado com múltiplas grades por atendente (auditoria 23/09)**: toda
essa mecânica — pendente, cooldown, órfão, bloqueio de "chamar próximo" — é filtrada por
`grade_id` no SQL (`TemChamadaPendenteNaoRechamada`, `ExistemChamadosOrfaos`), nunca só por
`usuario_id`. Isso foi verificado de propósito depois que um atendente passou a poder estar
alocado em várias grades ao mesmo tempo (inclusive de unidades físicas diferentes — ver skill
`papeis-e-telas`): uma chamada pendente ou um órfão numa grade **nunca** bloqueia as outras
grades do mesmo atendente. Reproduzido via curl (23/09): atendente com chamada pendente na
grade A conseguiu chamar normalmente na grade B na sequência, sem esperar o cooldown de A.

**Regra corrigida (23/09): o gate pra chamar outra pessoa é o COOLDOWN em si, não "já
rechamou uma vez".** Até 22/09 a checagem (`TemChamadaPendenteNaoRechamada`) só olhava se o
dono já tinha rechamado pelo menos uma vez (`cnt.total = 1` = ainda bloqueado, `>= 2` =
livre) — na prática isso liberava o atendente pra chamar outra pessoa IMEDIATAMENTE depois
de rechamar, mesmo com um novo cooldown (contando até a rechamada seguinte) ainda visível
contando na tela. Relatado como errado pelo dono do produto: "se eu estou chamando alguém
nesse momento e contando os 3min, eu não posso chamar outra pessoa até o prazo terminar" —
ou seja, **enquanto o cooldown desde a última chamada (seja a 1ª ou a 3ª) não tiver passado,
o atendente não pode chamar mais ninguém**, não importa quantas vezes já rechamou. Corrigido
comparando `now() < ultima_chamada + cooldown` em vez de contar chamadas. Só depois que esse
cooldown passa é que o atendente escolhe entre: rechamar (abre um cooldown novo, o ciclo se
repete) ou chamar outra pessoa (abandona esta, que vira **chamado órfão** — mecânica
inalterada, ver abaixo). Isso significa que, na prática, um chamado só pode virar órfão
depois de pelo menos um cooldown completo ter passado sem o dono rechamar OU depois de
rechamar e esperar um cooldown completo de novo — nunca "logo depois de rechamar uma vez".

**Chamado órfão**: um agendamento em `chamado` onde o dono já rechamou pelo menos uma vez E
já seguiu em frente (tem uma chamada mais recente em OUTRO agendamento). Regra de
prioridade: **enquanto existir qualquer chamado órfão NAQUELA GRADE, nenhum atendente ocioso
que também tenha aquela grade alocada pode puxar alguém novo dela** — precisa assumir um
órfão primeiro. Isso evita que uma pessoa já chamada fique esquecida enquanto todo mundo só
chama gente nova; nunca afeta as outras grades do mesmo atendente (ver nota de escopo acima).

**Assumir um órfão**: qualquer atendente que tenha aquela grade alocada pode assumir um
chamado órfão pro seu próprio guichê NAQUELA grade — isso transfere a posse (nova linha em `chamadas` com o novo `usuario_id`/`guiche_id`,
`agendamentos.guiche_id` atualizado) e o painel de TV anuncia de novo, agora com o guichê
novo. A partir daí, só quem assumiu pode rechamar/marcar presente essa pessoa — e ela também
precisa passar pelo mesmo ciclo (cooldown → rechamar → aí sim liberar o novo dono pra chamar
outra pessoa, se ainda não tiver aparecido).

**O prazo de ausência automática é um orçamento fixo, não reseta:** o `sla_atendimento_minutos`
(padrão 20min) conta a partir da **primeira** chamada de sempre, nunca da mais recente. Nem
rechamar nem uma transferência via "assumir" dão tempo extra — só usam o tempo que já
existia. Se estourar sem a pessoa aparecer, vira `ausente` automaticamente, do mesmo jeito
que já acontecia (mesmo padrão do timeout de chegada), independente de quantas vezes foi
rechamado ou transferido nesse meio tempo.

**Ausência manual (23/09)**: além do sweeper automático (acima), o dono da chamada tem um
botão de ausência (`POST /agendamentos/{id}/ausencia`, ícone `UserX` vermelho ao lado de
"marcar presente" na tabela "Chamando agora") pra marcar ausente na hora, sem esperar o SLA
inteiro — pedido do dono do produto pro caso de alguém avisar (ou o próprio cidadão avisar)
que já foi embora. Mesma regra de posse dos outros dois botões (só dono, status precisa ser
`chamado`), sem cooldown nenhum associado — é uma decisão do atendente, não uma repetição de
chamada.

## Janela/histórico do cidadão

Cada agendamento deve ter uma visão de detalhe com a linha do tempo completa: horário agendado
(`horario_previsto`), hora que chegou (`chegada_em`), hora(s) que foi chamado (todas as linhas de
`chamadas` daquele `agendamento_id`, ordenadas por `chamado_em` — a primeira é a chamada
original, as seguintes são rechamada), e hora que foi atendido (`atendido_em`). Ver skill
`papeis-e-telas` pra onde essa visão aparece.

## Dashboard do gestor

Ver skill `papeis-e-telas` — o dashboard ficou bem mais detalhado que a versão anterior deste
documento (filtro por unidade, atendidos por atendente, gráficos).

## Escopo do MVP

Uma única unidade (CRAS piloto), sem multi-secretaria ainda ativa na interface — mas o modelo de dados já deve manter a hierarquia (ver skill `modelo-dados`) pra não exigir migração quando expandir.
