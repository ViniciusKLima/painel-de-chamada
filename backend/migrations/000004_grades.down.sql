alter table unidades
  add column sla_chegada_minutos int not null default 60,
  add column sla_atendimento_minutos int not null default 20,
  add column duracao_chamada_painel_segundos int not null default 30,
  add column duracao_chamada_painel_fila_segundos int not null default 15,
  add column repeticoes_chamada int not null default 3,
  add column intervalo_repeticao_segundos int not null default 5,
  add column cooldown_rechamada_minutos int not null default 3;

-- Melhor esforço: recupera os valores da primeira grade de cada unidade (se várias grades
-- tiverem SLA diferente, essa diferença se perde ao voltar — inerente à reversão).
update unidades u
set sla_chegada_minutos = g.sla_chegada_minutos,
    sla_atendimento_minutos = g.sla_atendimento_minutos,
    duracao_chamada_painel_segundos = g.duracao_chamada_painel_segundos,
    duracao_chamada_painel_fila_segundos = g.duracao_chamada_painel_fila_segundos,
    repeticoes_chamada = g.repeticoes_chamada,
    intervalo_repeticao_segundos = g.intervalo_repeticao_segundos,
    cooldown_rechamada_minutos = g.cooldown_rechamada_minutos
from grades g
where g.unidade_id = u.id
  and g.id = (select id from grades g2 where g2.unidade_id = u.id order by g2.criado_em asc limit 1);

alter table usuarios drop column ultimo_acesso_em;
alter table usuarios drop column grade_id;

alter table guiches add column unidade_id uuid references unidades(id);
update guiches gu set unidade_id = g.unidade_id from grades g where g.id = gu.grade_id;
alter table guiches alter column unidade_id set not null;
alter table guiches drop constraint guiches_grade_id_nome_key;
alter table guiches drop column grade_id;
alter table guiches add constraint guiches_unidade_id_nome_key unique (unidade_id, nome);

alter table agendamentos drop column grade_id;

drop table grades;
