-- 23/09: gestor pode restringir se a recepção cria encaixe (walk-in) numa grade específica —
-- gestor SEMPRE pode, independente dessa flag (checado na aplicação, não aqui).
alter table grades add column permite_encaixe_recepcao boolean not null default true;

-- Novo padrão automático do sistema pra grade recém-criada (import auto-cria com os defaults
-- da coluna) — pedido do dono do produto: chegada continua 60min, atendimento cai pra 10min,
-- cooldown de rechamada cai pra 1min. Grades já existentes/configuradas NÃO são alteradas
-- retroativamente — isso muda só o valor usado quando uma grade nova é criada sem
-- configuração explícita.
alter table grades alter column sla_atendimento_minutos set default 10;
alter table grades alter column cooldown_rechamada_minutos set default 1;
