---
name: modelo-dados
description: Use sempre que for criar migrations, queries SQL, ou structs Go que representam entidades do domínio (agendamento, usuário, guichê, fila). Fonte da verdade do schema — não inventar coluna ou tabela sem checar aqui primeiro.
---

# Modelo de dados (Postgres)

Multi-tenant hierárquico: Prefeitura → Secretaria → Unidade → Usuário/Papel → Agendamento.
MVP roda com uma única unidade (CRAS piloto), mas o schema já nasce com a hierarquia completa.

## Tabelas

```sql
prefeituras (id uuid pk, nome text, slug text unique, issuer_jwt text unique)

secretarias (id uuid pk, prefeitura_id fk, nome text, sigla text)

unidades (
  id uuid pk, secretaria_id fk, nome text, -- ex: "CRAS Piloto"
  sla_chegada_minutos int not null default 60,     -- janela pra confirmar chegada antes de virar ausente automático
  sla_atendimento_minutos int not null default 20, -- duração padrão do atendimento; também usada como tolerância de chegada "no horário" e como janela de recall após chamado (ver skill `regras-negocio-fila`)
  duracao_chamada_painel_segundos int not null default 30,      -- quanto tempo uma chamada fica em destaque no painel de TV (ver skill `painel-tv`)
  duracao_chamada_painel_fila_segundos int not null default 15  -- duração reduzida usada quando há backlog de chamadas aguardando exibição
)
-- Pode ser criada por cadastro manual do gestor OU automaticamente durante a importação de
-- planilha, a partir do nome em "Local no mapa" quando não existir ainda (confirmado 19/09, ver
-- skill `importacao-planilha`) — nesse caso nasce com os valores padrão de SLA, ajustáveis depois.
-- Configurável por unidade pelo gestor ("SLA das chamadas no painel", ver skill `papeis-e-telas`).
-- Guardar esses dois valores por unidade em vez de fixos no código é o que permite o gestor
-- ajustar sem deploy. Ver skill `regras-negocio-fila` pra onde cada um é usado.

usuarios (
  id uuid pk,
  unidade_id fk, -- gestor pode realocar (mudar a pessoa de unidade) — mecanismo já suportado por essa FK, resolvido durante o MVP junto com o gestor (ver `papeis-e-telas`)
  nome text,
  email text,
  senha_hash text nullable, -- login do MVP (email+senha, ver skill `auth-login-mvp`) — hash (bcrypt), nunca texto puro
  papel text check (papel in ('gestor', 'atendente', 'recepcionista')), -- gestor pode trocar o papel de uma pessoa (ex: atendente -> recepcionista), resolvido durante o MVP junto com o gestor
  external_sub text unique, -- "sub" do JWT real — fica null durante todo o MVP (SSO real adiado, ver skill `auth-conecta-cidades`)
  ativo boolean default true -- gestor pode restringir/desativar acesso
)

guiches (id uuid pk, unidade_id fk, nome text, ativo boolean default true)
-- "Quantidade de guichês/salas" configurável pelo gestor = CRUD nessa tabela, não precisa de
-- campo de contagem separado.

lotes_importacao (
  id uuid pk, secretaria_id fk, usuario_id fk,
  -- POR SECRETARIA, não por unidade (corrigido 19/09) — um único upload de planilha do gestor
  -- gera agendamentos em VÁRIAS unidades ao mesmo tempo (ver skill `importacao-planilha`), então
  -- o lote não pode pertencer a uma unidade só.
  arquivo_nome text, data_upload timestamptz default now(),
  status text check (status in ('processando','concluido','erro')),
  total_linhas int, total_erros int
)

agendamentos (
  id uuid pk,
  unidade_id fk, -- resolvido a partir da coluna "Local no mapa" da planilha, ver `importacao-planilha`
  lote_id fk nullable,          -- null quando for encaixe cadastrado na hora
  protocolo text,               -- identificação do cidadão, SEM CPF — ver ressalva de CPF abaixo
  nome_cidadao text,
  cpf text nullable,            -- CPF do cidadão — guardado, SEM máscara (decisão 19/09: atendentes já têm acesso a esse dado na outra plataforma, mascarar aqui não protege nada de fato)
  telefone text nullable,       -- mesma decisão do CPF, sem máscara
  servico text nullable,        -- "Serviço" da planilha (ex: CadÚnico) — tipo de atendimento, não onde acontece
  grade_horario text nullable,  -- "Grade de horários" da planilha — ver `importacao-planilha`: relação exata com `unidade_id`/Local ainda não confirmada com múltiplos locais reais
  tipo text check (tipo in ('agendado', 'encaixe')),
  horario_previsto timestamptz nullable, -- null quando tipo = 'encaixe'
  chegada_em timestamptz nullable,       -- preenchido quando recepção confirma chegada
  prioridade text check (prioridade in ('no_horario', 'atrasado')) nullable,
    -- calculado UMA VEZ no momento da confirmação de chegada (comparando chegada_em contra
    -- horario_previsto + unidades.sla_atendimento_minutos) e gravado — não recalculado depois,
    -- pra não mudar de resultado se o SLA da unidade for reconfigurado mais tarde.
    -- Null pra tipo = 'encaixe' (não se aplica) ou enquanto ainda não chegou.
  guiche_id fk nullable,                 -- preenchido quando é chamado
  status text check (status in (
    'aguardando_chegada', 'sala_espera', 'chamado', 'atendido', 'ausente', 'cancelado'
  )) default 'aguardando_chegada',
  -- 'cancelado' reincorporado 19/09: decisão já tomada na fase Node (15/09) — cancelamento feito
  -- pelo cidadão ANTES de chegar (via planilha) é diferente de 'ausente' (chamado e não apareceu).
  atendido_em timestamptz nullable,  -- quando o atendente marca como atendido
  ausente_em timestamptz nullable,   -- quando vira ausente — manual (recepção/atendente) ou automático (job de SLA), ver `regras-negocio-fila`
  cancelado_em timestamptz nullable,
  motivo_cancelamento text nullable, -- vem preenchido às vezes com reagendamento disfarçado de cancelamento, ver `importacao-planilha`
  criado_em timestamptz default now()
)

chamadas (
  id uuid pk,
  agendamento_id fk,
  guiche_id fk,
  usuario_id fk, -- atendente que chamou — é o log de "quem chamou", 1 linha por chamada/rechamada
  chamado_em timestamptz default now()
)
-- A "janela do cidadão" (agendado / chegou / chamou / chamou de novo / atendeu, ver
-- `papeis-e-telas`) é montada juntando agendamentos.horario_previsto, .chegada_em, .atendido_em
-- com todas as linhas de chamadas daquele agendamento_id ordenadas por chamado_em.

mapeamento_externo (
  id uuid pk,
  tipo text check (tipo in ('department', 'unit')),
  external_id text,
  secretaria_id fk nullable,
  unidade_id fk nullable,
  unique (tipo, external_id)
)
```

## Índices importantes

```sql
create index on agendamentos (unidade_id, status);
create index on agendamentos (unidade_id, tipo, horario_previsto);
create index on chamadas (chamado_em desc); -- consulta "últimos 10 chamados"
```

## Query de referência: fila de sala de espera ordenada por prioridade

```sql
select *
from agendamentos
where unidade_id = $1 and status = 'sala_espera'
order by
  case when prioridade = 'no_horario' then 0 else 1 end,  -- no_horario primeiro; atrasado e encaixe competem juntos depois
  case when prioridade = 'no_horario' then horario_previsto else chegada_em end asc
limit 20;
```

Ver skill `regras-negocio-fila` para a justificativa dessa ordenação — **confirmado 19/09**:
`atrasado` e `encaixe` competem juntos por ordem de chegada (`chegada_em`), não existe mais um
tier `atrasado` isolado antes de encaixe.

## Diferença em relação à versão anterior do projeto (Node/Prisma)

Esta versão substitui o desenho anterior que tinha só 3 papéis (admin/gestor/atendente) sem separar recepção. Mudanças:
- Papel `admin` removido do MVP (volta quando multi-secretaria for ativado de verdade).
- Papel `recepcionista` adicionado — responsável pela confirmação de chegada.
- `agendamentos.status` ganhou os estados `aguardando_chegada` e `sala_espera` no lugar de só `aguardando`.
- Adicionado `tipo` (agendado/encaixe) e `chegada_em`.
- ~~Tabela `lotes_importacao` passou de `secretarias` para `unidades`~~ — **revertido em 19/09**: um lote pode (e vai) cobrir várias unidades de uma vez, ver "Mudanças (19/09, segunda rodada)" abaixo.

## Mudanças (19/09) — levantamento do fluxo operacional completo

- `unidades` ganhou `sla_chegada_minutos` e `sla_atendimento_minutos` (configuráveis pelo gestor).
- `agendamentos` ganhou `prioridade`, `atendido_em`, `ausente_em`.
- Nada mudou em `chamadas` — o desenho já suportava o "log de quem chamou" e o histórico de
  rechamada sem alteração.
- **Convenção de limite exclusivo** (confirmado 19/09): qualquer comparação de janela de tempo
  (chegada, atendimento, classificação de prioridade) deve usar `<` estrito, não `<=` — ao bater
  o minuto exato do limite, a janela já conta como estourada.

## Mudanças (19/09, segunda rodada) — planilha real com colunas novas

Uma nova planilha real trouxe 3 colunas que não existiam nas anteriores: `Grade de horários`,
`Local no mapa`, `Atendido por`. Ver skill `importacao-planilha` pro detalhe completo. Mudanças
de schema decorrentes:

- **`lotes_importacao` corrigido de `unidade_id` pra `secretaria_id`** — um upload só do gestor
  gera agendamentos em várias unidades ao mesmo tempo (confirmado pelo usuário: "depois terão
  vários atendimentos de locais diferentes e isso deverá ser filtrado"), então o lote não podia
  pertencer a uma unidade só. Esse erro já vinha desde a v1 deste documento.
- `agendamentos` ganhou `cpf`, `telefone`, `servico`, `grade_horario`, `cancelado_em`,
  `motivo_cancelamento`, e o status `cancelado` voltou pro enum — **essas eram todas decisões já
  tomadas na fase anterior do projeto (Node, 14–15/09) com esse mesmo tipo de dado real**, que
  não tinham sido carregadas pro schema Go ainda. Recarregadas agora, não são decisões novas.
- **CPF/telefone — decisão revista (19/09)**: a fase Node mascarava esses campos pra quem não
  fosse gestor/admin (função `mascararCamposSensiveis`, ver `_archive/node-legado`). **Essa
  máscara foi removida da decisão**: "não precisa mascarar nada, pq os atendentes já têm acesso a
  esses dados na outra plataforma" — CPF/telefone ficam visíveis pra qualquer usuário autenticado
  do sistema, sem distinção de papel.

## Mudanças (19/09, terceira rodada) — painel de TV

- `unidades` ganhou `duracao_chamada_painel_segundos` (padrão 30) e
  `duracao_chamada_painel_fila_segundos` (padrão 15) — configuráveis pelo gestor, ver skill
  `painel-tv`.
- **Nenhuma tabela nova pra fila de exibição do painel** — decisão explícita de mantê-la efêmera,
  em memória no processo Go, não persistida no Postgres. Ver skill `painel-tv` pro raciocínio.

## Mudanças (22/09) — introdução da Grade de horário

Mudança estrutural grande, confirmada explicitamente pelo dono do produto depois de uma
pergunta direta de esclarecimento (a relação "Grade de horários" × "Local no mapa" estava
documentada como **não confirmada** desde 19/09 — ver skill `importacao-planilha` — e essa
pergunta finalmente resolveu). Definição do dono do produto, literal: **"unidade = local,
grade de horário = agenda para um serviço dentro daquele local"**. Ou seja: uma `unidade`
(local físico, ex. "Cadastro Único Sede Prazeres") pode oferecer **vários serviços** ao
mesmo tempo, e cada serviço tem sua própria fila/guichês/SLA/painel — isso é uma `grade`.

```sql
grades (
  id uuid pk,
  unidade_id fk,
  servico text,      -- vem da coluna "Serviço" da planilha — normalizado, é a CHAVE de
                      -- roteamento pra grade (análogo ao "Local no mapa" pra unidade)
  nome text,          -- nome de exibição, editável pelo gestor (default = servico)
  sla_chegada_minutos int default 60,
  sla_atendimento_minutos int default 10,   -- default mudou de 20 pra 10 em 23/09
  duracao_chamada_painel_segundos int default 30,
  duracao_chamada_painel_fila_segundos int default 15,
  repeticoes_chamada int default 3,
  intervalo_repeticao_segundos int default 5,
  cooldown_rechamada_minutos int default 1, -- default mudou de 3 pra 1 em 23/09
  permite_encaixe_recepcao boolean default true, -- novo em 23/09, ver nota abaixo
  ativo boolean default true,
  criado_em timestamptz default now(),
  unique (unidade_id, servico)
)
```

**`permite_encaixe_recepcao` (23/09)**: se `false`, a recepção não pode cadastrar encaixe
(walk-in) nessa grade especificamente — precisa pedir pro gestor. **Gestor e admin sempre
podem cadastrar encaixe, independente desse campo** (checado na aplicação — `PostEncaixe`,
`internal/handlers/fila.go` — não no banco). Configurável na tela de configurações da grade,
junto com o nome de exibição e o SLA de recepção.

**Mudança de default (23/09), só pra grades novas**: `sla_atendimento_minutos` (20→10) e
`cooldown_rechamada_minutos` (3→1) — pedido do dono do produto pra refletir o ritmo real de
atendimento. `sla_chegada_minutos` não mudou (continua 60). **Isso só afeta o valor usado
quando uma grade é criada automaticamente sem configuração explícita** (import de planilha,
via `ResolverOuCriarGrade`) — grades já existentes/configuradas não foram alteradas
retroativamente.

**Essas colunas SAÍRAM de `unidades`** (que agora é só `id, secretaria_id, nome`) e foram
todas pra `grades` — SLA de chegada/atendimento, durações do painel, repetição/intervalo de
chamada, cooldown de rechamada. `unidades` virou puramente identidade do local físico.

**`guiches` mudou de `unidade_id` pra `grade_id`** — cada grade tem seu próprio conjunto de
guichês/salas, não mais compartilhado entre todos os serviços de uma unidade.
`unique(unidade_id, nome)` virou `unique(grade_id, nome)`.

**`agendamentos` ganhou `grade_id` (not null)**, resolvida automaticamente na importação a
partir de (`unidade_id`, `servico`) — mesmo padrão de auto-criação já usado pra `unidade`
(`ResolverOuCriarGrade`, análogo a `ResolverOuCriarUnidade`). Serviço vazio cai num bucket
`Geral` por unidade. `agendamentos.unidade_id` continua existindo (a recepção ainda trabalha
por unidade inteira, todas as grades) — `unidade_id` e `grade_id` coexistem, não é uma
substituição.

**`usuarios` ganhou `grade_id` nullable** — só preenchido pra **atendente** (a fila
específica que ele chama). Gestor e recepcionista continuam só com `unidade_id` (trabalham a
unidade inteira). Isolamento de tenant pra endpoints de fila: além de checar
`grade.unidade_id == usuario.unidade_id`, um atendente só pode agir na **própria** grade
(`usuario.grade_id == grade.id`) — gestor tem acesso a qualquer grade da sua unidade.

**`usuarios` também ganhou `ultimo_acesso_em` timestamptz nullable**, atualizado a cada login
— base da tela "Usuários internos" do gestor pra distinguir "pendente" (nunca logou) de
"último acesso em X".

**Reflexo em todo o resto do sistema** (ver skills `regras-negocio-fila`/`papeis-e-telas`
atualizadas na mesma data): guichês, sala de espera, "chamando agora", chamar-próximo,
atendidos/ausentes-hoje e o painel de TV público passaram a ser escopados por **grade**, não
mais por unidade — cada grade tem seu próprio link de painel (`/painel/{gradeId}`). Só
recepção (lista do dia, confirmar chegada, encaixe) continua escopada por **unidade** — a
pessoa chega fisicamente num local, não numa grade especificamente, e pode não estar claro
ainda qual grade é dela até a recepção decidir (no caso de encaixe, quem cria explicitamente
escolhe a grade no formulário).

## Mudanças (22/09, rodada do Admin) — Admin, e Gestor passa de Unidade pra Secretaria

Pedido do dono do produto: hierarquia de acesso `Admin → Prefeitura → Secretaria → Usuários`,
com o Admin como acesso master da plataforma inteira. `prefeituras`/`secretarias` já
existiam no schema desde a v1 (só não tinham endpoint nenhum usando) — a mudança real foi em
`usuarios`:

```sql
usuarios (
  id uuid pk,
  secretaria_id fk nullable,  -- só gestor: gerencia a secretaria INTEIRA agora, não uma
                              -- unidade só (antes só existia unidade_id) — todo gestor
                              -- existente foi migrado (migration 000005) pra secretaria da
                              -- unidade em que estava, perdendo o vínculo direto com aquela
                              -- unidade específica
  unidade_id fk nullable,     -- agora opcional (era not null) — admin não tem nenhuma;
                              -- continua obrigatório NA PRÁTICA (validado na aplicação) pra
                              -- recepcionista/atendente
  grade_id fk nullable,       -- inalterado — só atendente
  papel text check (papel in ('admin', 'gestor', 'atendente', 'recepcionista')),
  ...
)
```

**Vínculo por papel, resumido** (ver skill `papeis-e-telas` pro detalhe de tela):
admin → nenhum; gestor → `secretaria_id`; recepcionista → `unidade_id`; atendente →
`unidade_id` + `grade_id`.

**Email virou único GLOBALMENTE** (`unique(email)`, era `unique(unidade_id, email)`) — o
login já buscava por email sem escopo de unidade desde o MVP (skill `auth-login-mvp`,
"assumido que email é único na prática"); o admin não tem `unidade_id` nenhum pra ancorar a
constraint antiga, então isso precisava ficar correto de qualquer jeito.

**`prefeituras` ganhou identidade visual configurável pelo admin**:
```sql
prefeituras (
  ...,
  cor_destaque text not null default '#146c8c',  -- ações/links, mapeia pro token --cor-primaria
  cor_clara text not null default '#e4f1f5'       -- fundo/cabeçalho de tabela, --cor-info-fundo
)
```
Aplicadas em runtime (não build-time) via `frontend/src/tema.ts`, chamado depois de resolver
a sessão em cada tela autenticada — ver skill `identidade-visual`.

**Decisão de escopo, documentada pra não presumir o contrário**: recepcionista continua
vinculada à UNIDADE inteira, não a uma grade — apesar do pedido original ter mostrado um
diagrama com Recepcionista embaixo de "Grade", isso contradiria a decisão de negócio já
confirmada e testada (a recepção precisa ver todo mundo que chega num local físico, não só
quem tem hora marcada numa fila específica). Atendente continua o único papel vinculado a uma
grade específica.

**Endpoints reorganizados** (ver skill `papeis-e-telas` pra lista completa): dashboard,
grade de horário e usuários internos do gestor migraram de `/unidades/{unidadeId}/...` pra
`/secretarias/{secretariaId}/...` (agregam todas as unidades da secretaria via subquery/
join). Um endpoint novo e distinto, `GET /unidades/{unidadeId}/grades`, foi criado
especificamente pro formulário de encaixe da recepção (só as grades daquela unidade, não a
secretaria inteira) — não confundir os dois.

**`internal/auth/permissoes.go` (novo)**: centraliza toda decisão de "quem pode acessar o
quê" (`EhAdmin`, `PodeAcessarSecretaria`, `PodeAcessarUnidade`, `PodeAcessarGrade`) — nenhum
handler deve voltar a comparar `usuario.unidadeId`/`secretariaId` diretamente contra um path
param, sempre passar por essas funções.

## Mudanças (23/09): multi-grade e exclusão (migration 000008)

**`usuarios.grade_id` (FK única, nullable) foi REMOVIDA** — a referência a ela em qualquer
seção anterior deste documento está desatualizada. Virou uma tabela N:N:

```sql
usuario_grades (
  usuario_id uuid references usuarios(id) on delete cascade,
  grade_id uuid references grades(id) on delete cascade,
  primary key (usuario_id, grade_id)
)
```

Pedido explícito do dono do produto: "atendente ou recepção podem ser alocados em mais de
uma grade". Atendente exige pelo menos 1 linha aqui; recepcionista pode ter 0 (sem
restrição — continua vendo/cadastrando encaixe em qualquer grade da própria unidade, como
sempre) ou mais (aí só nas marcadas). A checagem de acesso (`auth.PodeAcessarGrade`) recebe a
lista de IDs já resolvida pelo chamador — continua uma função pura, sem acesso a banco.

A sessão (`GET /sessao`) não devolve mais `gradeId`/`gradeNome` — devolve `grades:
[{id,nome}][]`. O frontend escolhe em qual grade trabalhar em RUNTIME (estado local, não
sessão), mesmo padrão já usado pro guichê — ver skill `papeis-e-telas`.

**Exclusão de grade/usuário/guichê** (pedido: "quero que seja possível excluir"): os três
podem ter histórico em `chamadas`/`agendamentos` (logs append-only, nunca apagados — ver
seção sobre integridade referencial mais acima), então nunca é um DELETE de verdade:

- `grades.excluido_em timestamptz` (coluna nova) e `guiches.excluido_em timestamptz` (coluna
  nova) — toda consulta ativa filtra `excluido_em is null`. Distinto do `ativo` que os dois
  já tinham: `ativo` é a desativação reversível de sempre (some da seleção do atendente mas
  continua na tela de gestão pra reativar); `excluido_em` some da tela de gestão também.
- **`usuarios` NÃO ganhou coluna nova** — "excluir" reaproveita `usuarios.ativo` (já existia,
  já bloqueava login, nunca tinha um botão de UI pra alternar). Criar uma segunda coluna só
  pra esse mesmo efeito prático seria redundante.
- Guichê tem uma regra extra: uma grade precisa de pelo menos 1 guichê/sala não-excluído
  sempre — `ExcluirGuiche` bloqueia (409 `ultimo_guiche`) tentando excluir o último, checagem
  e exclusão na MESMA query SQL (atômico). Toda grade nova (criada via import,
  `ResolverOuCriarGrade`) já nasce com 1 guichê padrão ("Guichê 1") pra nunca ficar sem
  nenhum.

**Armadilha real encontrada na auditoria de multi-grade (23/09)**: o `on delete cascade` de
`usuario_grades.grade_id` (acima) só dispara numa exclusão de LINHA de verdade — como
`ExcluirGrade` é soft-delete (só grava `excluido_em`, a linha de `grades` continua existindo),
o cascade nunca roda, e sem correção `usuario_grades` ficaria com um vínculo apontando pra
uma grade excluída (órfão real, filtrado só por sorte nas consultas que já faziam join com
`excluido_em is null`, mas não em todas). Corrigido fazendo `ExcluirGrade`
(`internal/repository/grades.go`) rodar dentro de uma transação que TAMBÉM apaga
explicitamente as linhas de `usuario_grades` daquela grade — não dá pra confiar no `on delete
cascade` do schema pra limpeza de soft-delete, só pra hard-delete de verdade. Verificado via
teste real: criar uma grade descartável, alocar um atendente nela, excluir a grade, confirmar
`usuario_grades` zerado e a sessão do atendente sem mais aquela grade.

**Validação extra em `PatchUsuario` (23/09)**: recepcionista trabalha o balcão de UMA
unidade física fixa (`usuarios.unidade_id`) — a lista de grades dela é só um FILTRO de quais
grades pode cadastrar encaixe DENTRO dessa mesma unidade (`GetGradesDaUnidade`), nunca um
vínculo que funcione fora dela. Sem checagem, um gestor/admin podia alocar uma recepcionista
numa grade de OUTRA unidade — o vínculo gravava sem erro, mas nunca teria efeito nenhum
(morto, silencioso). `PatchUsuario` agora rejeita isso com 400 `corpo_invalido` antes de
gravar. Atendente não tem essa restrição — pode legitimamente ter grades de unidades físicas
diferentes (exemplo real confirmado pelo dono do produto).

## Decisões em aberto (não implementar sem confirmar)

- **Gestão de atendentes** (alocar/trocar unidade, restringir acesso, trocar papel
  atendente↔recepcionista) — ✅ resolvida e implementada em 22/09, ver seção acima e a tela
  "Usuários internos" (skill `papeis-e-telas`). Não é mais decisão em aberto.
- **Múltiplas unidades reais simultâneas** (SASC opera várias fisicamente distintas, ver
  CLAUDE.md) e **múltiplas secretarias** continuam fora do piloto atual, mesmo com o modelo
  já suportando tecnicamente.
