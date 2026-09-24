-- Grade de horário (22/09): uma unidade (local físico) pode oferecer vários serviços, cada
-- um com sua própria fila/guichês/SLA/painel — essa é a "grade". Antes, guichês e SLA
-- viviam direto em `unidades`, assumindo uma fila só por unidade; isso deixou de ser
-- verdade. Ver skill `modelo-dados` pro raciocínio completo.
create table grades (
  id uuid primary key default gen_random_uuid(),
  unidade_id uuid not null references unidades(id),
  servico text not null,
  nome text not null,
  sla_chegada_minutos int not null default 60,
  sla_atendimento_minutos int not null default 20,
  duracao_chamada_painel_segundos int not null default 30,
  duracao_chamada_painel_fila_segundos int not null default 15,
  repeticoes_chamada int not null default 3,
  intervalo_repeticao_segundos int not null default 5,
  cooldown_rechamada_minutos int not null default 3,
  ativo boolean not null default true,
  criado_em timestamptz not null default now(),
  unique (unidade_id, servico)
);

create index on grades (unidade_id);

-- Backfill: uma grade por combinação (unidade, serviço) já presente nos agendamentos
-- existentes, herdando o SLA/painel que hoje mora em `unidades`.
insert into grades (
  unidade_id, servico, nome, sla_chegada_minutos, sla_atendimento_minutos,
  duracao_chamada_painel_segundos, duracao_chamada_painel_fila_segundos,
  repeticoes_chamada, intervalo_repeticao_segundos, cooldown_rechamada_minutos
)
select distinct
  a.unidade_id,
  coalesce(nullif(a.servico, ''), 'Geral'),
  coalesce(nullif(a.servico, ''), 'Geral'),
  u.sla_chegada_minutos, u.sla_atendimento_minutos, u.duracao_chamada_painel_segundos,
  u.duracao_chamada_painel_fila_segundos, u.repeticoes_chamada, u.intervalo_repeticao_segundos,
  u.cooldown_rechamada_minutos
from agendamentos a
join unidades u on u.id = a.unidade_id;

-- Toda unidade precisa de pelo menos uma grade, mesmo sem agendamento importado ainda.
insert into grades (
  unidade_id, servico, nome, sla_chegada_minutos, sla_atendimento_minutos,
  duracao_chamada_painel_segundos, duracao_chamada_painel_fila_segundos,
  repeticoes_chamada, intervalo_repeticao_segundos, cooldown_rechamada_minutos
)
select u.id, 'Geral', 'Geral', u.sla_chegada_minutos, u.sla_atendimento_minutos,
  u.duracao_chamada_painel_segundos, u.duracao_chamada_painel_fila_segundos,
  u.repeticoes_chamada, u.intervalo_repeticao_segundos, u.cooldown_rechamada_minutos
from unidades u
where not exists (select 1 from grades g where g.unidade_id = u.id);

alter table agendamentos add column grade_id uuid references grades(id);
update agendamentos a
set grade_id = g.id
from grades g
where g.unidade_id = a.unidade_id
  and g.servico = coalesce(nullif(a.servico, ''), 'Geral');
alter table agendamentos alter column grade_id set not null;
create index on agendamentos (grade_id, status);

alter table guiches add column grade_id uuid references grades(id);
update guiches gu
set grade_id = (select g.id from grades g where g.unidade_id = gu.unidade_id order by g.criado_em asc limit 1);
alter table guiches alter column grade_id set not null;
alter table guiches drop constraint guiches_unidade_id_nome_key;
alter table guiches drop column unidade_id;
alter table guiches add constraint guiches_grade_id_nome_key unique (grade_id, nome);

-- Atendente passa a ser alocado numa grade (a fila específica que ele chama); recepcionista
-- e gestor continuam só em `unidade_id` (trabalham com a unidade inteira, todas as grades).
alter table usuarios add column grade_id uuid references grades(id);
update usuarios us
set grade_id = (select g.id from grades g where g.unidade_id = us.unidade_id order by g.criado_em asc limit 1)
where us.papel = 'atendente';

-- Rastreio de último acesso (22/09) — base pra tela "Usuários internos" do gestor mostrar
-- "pendente" (nunca logou) vs. "último acesso em X".
alter table usuarios add column ultimo_acesso_em timestamptz;

-- SLA/painel saem de `unidades` — cada grade tem o seu agora.
alter table unidades
  drop column sla_chegada_minutos,
  drop column sla_atendimento_minutos,
  drop column duracao_chamada_painel_segundos,
  drop column duracao_chamada_painel_fila_segundos,
  drop column repeticoes_chamada,
  drop column intervalo_repeticao_segundos,
  drop column cooldown_rechamada_minutos;
