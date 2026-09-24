---
name: papeis-e-telas
description: Use ao criar rotas de frontend, componentes de tela, ou regras de autorização de endpoint. Define o que cada papel pode ver e fazer — nenhum papel deve enxergar dado ou ação fora do que está descrito aqui.
---

# Papéis e telas

Cinco papéis (`admin`, `gestor`, `atendente`, `recepcionista`, e o conceito sem-login
`painel_chamada`), telas bem diferentes entre si. Nenhuma se sobrepõe em permissão.
**Atualizado em 22/09 (rodada do Admin)** — Admin entrou como acesso master da plataforma, e
o Gestor deixou de ser vinculado a uma UNIDADE (o que só funcionava porque o piloto tinha uma
unidade só) pra ser vinculado à SECRETARIA inteira. Ver skill `modelo-dados` pro schema
completo (hierarquia `Admin → Prefeitura → Secretaria → Usuários`) e
`internal/auth/permissoes.go` pra onde a lógica de "quem pode o quê" vive centralizada.

## Admin

Acesso master da plataforma inteira — **sem vínculo nenhum** de prefeitura/secretaria/
unidade (`usuario.SecretariaID`/`UnidadeID`/`GradeID` todos nulos). Tela própria em `/admin`,
mesma estrutura de sidebar do Gestor (reaproveita as classes CSS `gestor-shell`/
`gestor-sidebar`, que já eram genéricas apesar do nome):

- **Dashboard** (`/admin`): contadores simples da plataforma inteira (total de prefeituras,
  secretarias, gestores, atendentes, recepcionistas) — deliberadamente sem gráfico nesta
  rodada, a prioridade era a estrutura de acesso, não dashboard de negócio pro admin.
- **Prefeituras** (`/admin/prefeituras`): lista + criar (só nome). Engrenagem por linha abre
  `/admin/prefeituras/{id}` — configurações: nome + as duas cores de identidade (destaque e
  clara, ver skill `identidade-visual`) + lista de secretarias daquela prefeitura, com
  criação rápida de secretaria nova ali mesmo.
- **Usuários internos** (`/admin/usuarios`, consolidada 22/09 — pedido explícito: "cria
  gestores e outros funcionarios na mesma tela"): lista TODO gestor/atendente/recepcionista
  da plataforma inteira numa tabela só. "Criar usuário" abre um formulário **condicional
  pelo Cargo escolhido**: Gestor pede só Prefeitura → Secretaria (vinculado à secretaria
  inteira); Atendente/Recepcionista pedem Prefeitura → Secretaria → **Grade** — a grade já diz
  a qual unidade pertence, então não pergunta Unidade como campo separado (ver skill
  `modelo-dados`); recepcionista não grava a grade escolhida no banco (só usa pra descobrir a
  unidade), atendente grava normalmente.

Admin também pode entrar em qualquer endpoint de unidade/secretaria/grade que gestor
acessaria (bypass central em `auth.EhAdmin`, ver `internal/auth/permissoes.go`) — por
exemplo, pra depurar a fila de uma unidade sem precisar logar como o gestor dela.

## Gestor

Vinculado a uma **secretaria inteira** (não mais uma unidade só, desde 22/09) — enxerga e
gerencia todas as unidades/grades daquela secretaria. Tela própria com **sidebar lateral
fixa** (não cabeçalho de topo como as outras telas — ver skill `identidade-visual`), 5
seções (ordem da sidebar: Dashboard, Usuários internos, Grade de horário, Painéis,
Importar):

### Dashboard (`/gestor`)

- 3 blocos de total do dia: **Total hoje**, **Total atendido**, **Total ausente** (dados da
  SECRETARIA inteira, todas as unidades e grades dela).
- **Atendido por atendente**: lista de progressão com porcentagem (quem atendeu mais hoje,
  em proporção do total).
- **Gráfico de linha "Atendido vs. ausente"**, últimos 7 dias (SVG feito à mão, sem
  biblioteca de gráfico — ver `componentes/GraficoLinha.tsx`).
- **Atendimentos por grade hoje**: proporção atendido-vs-ausente por grade (não só volume) —
  só faz sentido depois que uma unidade pode ter mais de uma grade (ver seção seguinte).

### Grade de horário (`/gestor/grades`)

Cada `grade` é a fila de **um serviço específico** dentro de uma unidade (ex.: "CadÚnico" e
"Bolsa Família" na mesma unidade seriam duas grades distintas — ver skill `modelo-dados`).
Lista TODAS as grades de TODAS as unidades da secretaria (`GET /secretarias/{id}/grades`,
join com `unidades`) — não só uma unidade. Nome + serviço de cada grade, e por linha:
- **Copiar link do painel** — cada grade tem seu próprio painel público
  (`/painel/{gradeId}`), diferente de antes (um painel por unidade).
- **Engrenagem** → abre `/gestor/grades/{gradeId}`, a tela de configurações daquela grade
  especificamente: identificação (nome de exibição, ativar/desativar), guichês/salas (CRUD,
  igual já existia), SLA da recepção, SLA do atendimento, configurações do painel. Tudo que
  antes vivia em "Gerenciar CRAS" por unidade agora vive aqui, por grade.

### Painéis (`/gestor/paineis`)

Central de painéis (23/09): como cada GRADE tem seu próprio painel de TV público
(`/painel/{gradeId}`), uma secretaria com várias unidades pode ter dezenas de painéis. Lista
todos, com busca, e abre qualquer um "em foco" dentro da própria tela via `<iframe>` (zero
duplicação da lógica de `painel-tv`) com um botão de tela cheia de verdade (Fullscreen API no
iframe). **Agrupado por unidade** (auditoria de multi-grade, 23/09 — mesma regra de negócio
da tela do atendente acima): grades do mesmo local físico aparecem juntas sob um cabeçalho
com o nome da unidade, nunca como cartões soltos e independentes. Exemplo:

```
Unidade A
  [Grade 1]  [Grade 2]  [Grade 3]
Unidade B
  [Grade 4]
```

Também tem uma seção "Acesso pela TV do local" — um link público por UNIDADE
(`/tv/{unidadeId}`, sem login nenhum, endpoint `GET /unidades/{unidadeId}/paineis-publico`),
pra deixar fixo na TV física do CRAS; essa tela pública já é inerentemente "um grupo só" (é
sempre uma unidade específica), então não precisa de agrupamento adicional. A mesma regra de
agrupamento por unidade se aplica também à aba "Painéis" da configuração de cada prefeitura
no Admin (`PrefeituraPaineis.tsx`) — mesmo componente visual, dado agregado por prefeitura
inteira em vez de uma secretaria só.

### Usuários internos (`/gestor/usuarios`)

Lista de todo mundo que trabalha na SECRETARIA inteira (`GET /secretarias/{id}/usuarios` —
o próprio gestor, mais todo recepcionista/atendente de qualquer unidade dela, via join) —
nome, email, cargo (papel), uma **etiqueta por grade** em que está alocado (23/09 — um
atendente/recepcionista pode ter zero, uma ou várias; deixou de ser um único valor) e
status: **pendente** (nunca fez login) ou **"acesso em {data}"** (último login registrado).
Engrenagem por linha abre um modal de edição: nome, email, cargo (select — gestor pode trocar
o papel de outra pessoa), nova senha (opcional — só troca se preenchida) e as grades
alocadas (`componentes/SelecaoGrades.tsx`, checkboxes — multi-seleção, não mais um `<select>`
único). Atendente pode ter grades de unidades diferentes; recepcionista só pode ser alocada a
grades da própria unidade fixa (validado no backend, `PatchUsuario` — 400 `corpo_invalido` se
tentar uma grade de outra unidade).

**Gestor pode criar atendente/recepcionista direto** (22/09, botão "Criar funcionário" que
estava faltando) — `POST /unidades/{unidadeId}/funcionarios` aceita gestor além de admin,
checando que a unidade pertence à própria secretaria (`usuarioPodeAcessarUnidade`). Como o
gestor só tem UMA secretaria, o formulário **não pergunta Prefeitura nem Secretaria** — só
Cargo + Grade (implica a unidade) + nome/email/senha. Criar um GESTOR novo continua exclusivo
do Admin (ver seção acima) — o gestor não cria outro gestor.

**Gestor não pode mudar o próprio cargo** (22/09, pedido explícito — evita ele se rebaixar/
travar o próprio acesso sem outro gestor por perto pra reverter): o select "Cargo" fica
desabilitado quando `usuario.id === alvo.id` no modal de edição, com uma nota explicando por
quê; nome/email/senha continuam editáveis normalmente. Validado também no backend
(`PatchUsuario` rejeita com 403 `nao_pode_mudar_proprio_cargo`) — não é só uma trava de UI.

### Importar (`/gestor/importar`)

Fluxo em **duas etapas** (pedido explícito do dono do produto, 22/09), não mais importação
direta:
1. Arrastar ou clicar pra escolher o arquivo dispara uma **análise** (endpoint
   `.../lotes/preview`) — mostra quantos agendamentos válidos, cancelados e com erro, uma
   lista "agendamentos por dia", quais unidades/grades novas seriam criadas, e a tabela de
   erros — **sem gravar nada no banco ainda**.
2. O gestor confirma ("Confirmar N agendamentos") — só então o arquivo é reenviado pro
   endpoint que grava de verdade (o mesmo de sempre, `.../lotes`).
O sistema **separa automaticamente por unidade E por grade** (serviço) — o filtro é
responsabilidade do nosso sistema, não de quem gerou a planilha (mesmo princípio já
registrado na skill `api-conecta-cidades` sobre a API externa).

### Não faz

Nenhuma ação sobre a fila em si (não chama, não confirma chegada, não marca
atendido/ausente) — é leitura/gerencial + configuração + importação.

## Recepcionista

Trabalha a **unidade inteira** (todas as grades daquele local físico), não uma grade
específica — a pessoa chega fisicamente num local, não numa fila de serviço. Tela com busca
(nome/protocolo/CPF) e 3 abas:

- **Recepção**: quem está `aguardando_chegada` hoje. Quando o horário previsto chega, a
  linha fica marcada visualmente (contagem regressiva até o `sla_chegada_minutos` **da
  grade** daquele agendamento, não mais da unidade — ver `modelo-dados`).
- **Sala de espera**: quem já confirmou chegada (`sala_espera`). Cada linha tem um ícone
  "puxar de volta" (desfaz a confirmação, volta pra `aguardando_chegada`) — só pra
  `tipo = 'agendado'`, encaixe não tem essa etapa.
- **Faltaram** (22/09): quem virou `ausente` HOJE — informativo, sem ação, pra recepção já
  saber que a pessoa está atrasada se ela aparecer depois de já ter sido marcada ausente
  automaticamente.

**Faz**:
- Confirma chegada de um agendado → grava `chegada_em`, calcula e grava `prioridade`
  (`no_horario`/`atrasado`) comparando contra o SLA **da grade** do agendamento, muda
  `status` pra `sala_espera`.
- Cadastra encaixe (walk-in) → precisa escolher **qual grade** (serviço) é o encaixe, se a
  unidade tiver mais de uma (select some se só existir uma) — cria `agendamentos` com
  `tipo = 'encaixe'`, já em `sala_espera`, sem `horario_previsto`/`prioridade`.

**Não vê**: guichês nem aciona chamada — isso é do atendente.

## Atendente

**Reescrito 23/09 (auditoria de multi-grade)** — a versão anterior deste texto descrevia uma
única grade fixa por atendente (`usuarios.grade_id`, coluna que nem existe mais desde a
rodada 30) e um gate obrigatório de seleção de grade ao entrar. Isso foi revertido: um
atendente pode estar alocado em **várias grades ao mesmo tempo** (`usuario_grades`,
N:N — ver skill `modelo-dados`), inclusive de **unidades físicas diferentes** (ex: um
atendente com grades numa unidade A e numa unidade B). A regra de negócio confirmada pelo
dono do produto é explícita: "o atendente NÃO escolhe uma grade pra trabalhar — ele vê e
trabalha em TODAS as que estiver alocado, consolidadas". Não existe mais nenhuma tela/modal
de seleção de grade — só de **guichê**, e mesmo assim por grade, não bloqueando a tela
inteira (ver abaixo).

Todos os dados operacionais (chamados, sala de espera, atendidos/ausentes hoje) vêm de
endpoints consolidados que derivam a lista de grades do próprio usuário autenticado no
backend (`GET /api/v1/atendente/grades*`, `internal/handlers/fila.go`, função
`minhasGrades`) — o frontend nunca envia a lista de IDs, evitando N requisições (uma por
grade) e qualquer risco de o cliente pedir dados de uma grade que não é dele.

### Guichê é por grade, escolhido sob demanda (não é mais um gate)

Cada grade que o atendente tem resolve o próprio guichê de forma independente — ao entrar,
o sistema tenta reocupar automaticamente o guichê lembrado (`localStorage`, chave por grade)
de cada uma; se conseguir, aquela grade já aparece pronta pra trabalhar. Só a(s) grade(s)
sem guichê resolvido mostram uma pílula "Selecionar guichê" (clicável, abre um modal
pequeno só daquela grade) — as outras já resolvidas continuam operando normalmente, nada
trava esperando essa escolha. **Ocupação em tempo real (22/09)**: um guichê já escolhido por
outro atendente aparece desabilitado no modal, com "Ocupado por {nome}" — ver skill
`regras-negocio-fila` pro mecanismo de heartbeat por trás disso (agora um heartbeat por
grade/guichê ocupado, não mais um só).

### Realocação ao vivo, sem logout/reload

A lista de grades do atendente é revalidada periodicamente (a cada 8s, `GET /sessao`)
enquanto a tela está aberta — se um gestor/admin adicionar ou remover uma grade dele nesse
meio tempo, a mudança aparece sozinha: uma grade nova ganha sua própria seção (com guichê a
resolver); uma grade removida some, e se havia um guichê ocupado nela, é liberado no backend
automaticamente. Isso fecha um requisito explícito da auditoria de 23/09: "não quero
situações em que a alteração aparece apenas depois de logout/login ou apenas depois de
atualizar manualmente a página".

Tela com 3 blocos lado a lado, **sem cartão em volta do título** (ver skill
`identidade-visual`):

### "Pendentes" (chamando agora)

Uma tabela só, **consolidada entre todas as grades do atendente**, com uma coluna extra
mostrando de qual grade é cada linha. A regra de posse/cooldown/órfão continua exatamente a
mesma de sempre, mas **escopada por grade** (confirmado no código:
`Repo.TemChamadaPendenteNaoRechamada`/`ExistemChamadosOrfaos` filtram por `grade_id`) — uma
chamada pendente ou um órfão numa grade nunca bloqueia as outras grades do mesmo atendente,
mesmo as da mesma unidade física. Só o **dono** (quem chamou, ou quem assumiu — ver abaixo)
ganha os botões de ação:
- **Rechamar**: só depois de um cooldown configurável (`cooldown_rechamada_minutos` da
  grade) desde a última chamada — enquanto isso, ícone vira uma pílula com relógio +
  contagem regressiva (`.botao-cooldown`).
- **Marcar presente**: grava `atendido_em`, `status` vira `atendido`.
Um chamado vira **órfão** quando o dono já rechamou pelo menos uma vez E já seguiu em frente
pra outra pessoa **naquela mesma grade** — qualquer atendente ocioso QUE TAMBÉM tenha aquela
grade alocada pode **assumir** esse chamado (precisa ter um guichê já resolvido nela; se não
tiver, o modal daquela grade abre pra escolher antes de assumir). Enquanto existir qualquer
órfão numa grade, ninguém consegue puxar gente nova da sala de espera DAQUELA grade.

### "Sala de espera"

**Agrupada por unidade → grade** (23/09, pedido explícito do dono do produto: "grades do
mesmo local devem aparecer agrupadas visualmente, não como filas independentes" — ver o
exemplo de árvore na seção "Painéis" do Gestor, acima — a mesma regra vale aqui). O cabeçalho da unidade
só aparece quando o atendente tem grades de mais de uma unidade; com uma só, seria ruído
sem informação nova. Dentro de cada grupo de grade:
- **Aguardando**: quem está em `sala_espera` NAQUELA grade, ordenado pela regra de
  prioridade (no-horário primeiro; atrasado e encaixe competem juntos depois, por ordem de
  chegada). Botão **Chamar próximo** e o indicador de guichê ficam no cabeçalho do próprio
  grupo da grade — desabilitado se o atendente tem uma chamada pendente não-rechamada NAQUELA
  grade, ou se existe qualquer órfão NELA (nunca por causa de outra grade).
- **Ausentes**: aba separada, ainda compartilhada entre atendentes mas mostrada como lista
  única com uma coluna "Grade" (não agrupada como a de aguardando, por ser só consulta —
  quem faltou depois de já ter sido chamado, hoje, em qualquer grade do atendente).

### "Meus atendimentos hoje"

**Pessoal** (só quem o atendente logado atendeu, em qualquer uma das suas grades), com busca
por nome/protocolo e uma coluna "Grade". Consulta pura, sem ação.

### Janela de detalhe do cidadão

Linha do tempo de um agendamento: horário agendado, chegada, chamada(s) (pode ter mais de
uma, por rechamada/assumir), atendimento. Acessível via `GET /agendamentos/:id/historico`.

## Painel de TV (público, sem login)

Escopado por **grade**, não mais por unidade (`/painel/{gradeId}`) — cada grade tem seu
próprio painel/link, já que cada uma é uma fila independente com seu próprio guichês/SLA. Ver
skill `painel-tv` pro comportamento visual completo (modo ativo/descanso, voz, repetição).

## Regra de autorização (backend) — centralizada em `internal/auth/permissoes.go`

Pedido explícito do dono do produto (rodada do Admin, 22/09): "não quero permissões
espalhadas em vários componentes através de condições difíceis de manter". Toda decisão de
"esse usuário pode acessar esse recurso?" passa por uma função pura de
`internal/auth/permissoes.go` (`EhAdmin`, `PodeAcessarSecretaria`, `PodeAcessarUnidade`,
`PodeAcessarGrade`) — nenhum handler compara `usuario.unidadeId`/`secretariaId` diretamente.
Os helpers `exigirSecretaria`/`exigirUnidade`/`exigirGrade` (`internal/handlers/fila.go`)
resolvem o dado do path e chamam essas funções:

- **Admin**: bypassa toda checagem de tenant — pode acessar qualquer secretaria/unidade/
  grade da plataforma.
- **Secretaria** (`/secretarias/{secretariaId}/...`) — dashboard, grades, usuários do gestor:
  só admin ou o gestor `usuario.secretariaId == path.secretariaId`.
- **Unidade** (`/unidades/{unidadeId}/...`) — recepção, encaixe, grades daquela unidade
  específica (pro formulário de encaixe): admin sempre; gestor se a unidade pertence à
  própria secretaria; recepcionista/atendente só a própria `usuario.unidadeId`.
- **Grade** (`/grades/{gradeId}/...`) — fila do atendente, configuração da grade: admin
  sempre; gestor qualquer grade de qualquer unidade da própria secretaria; atendente só se a
  grade estiver na sua lista de alocações (`usuario_grades`, N:N desde a rodada 30 — checagem
  via `slices.Contains(gradesDoUsuario, gradeID)` em `PodeAcessarGrade`, não mais um campo
  único). Os endpoints consolidados `/atendente/grades*` (23/09) não recebem `{gradeId}` na
  URL — derivam a lista direto da sessão, então nem precisam dessa checagem por request.

Nenhuma tela de frontend deve ser a única barreira — sempre validar no backend também, já
que é fácil chamar a API direto do navegador.

## Painel de Chamada (`painel_chamada`) — não é um login de verdade

O quinto "papel" existe só como marcador conceitual em `domain.PapelPainelChamada`, pro
esquema central de permissões ter um nome de primeira classe pra esse contexto — nenhuma
linha de `usuarios` tem esse papel de verdade. O painel de TV continua público, sem cookie de
sessão, exatamente como sempre foi (`GET /painel/{gradeId}`, sem `auth.Middleware`). Isso é
proposital: existe pro dia em que o painel precisar de autenticação própria (ex. um
dispositivo/kiosk com credencial) sem ter que inventar o conceito do zero — não implementar
login pra ele agora sem pedido explícito.
