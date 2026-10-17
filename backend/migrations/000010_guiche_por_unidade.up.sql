-- Guichê volta a pertencer à UNIDADE (local físico), não à grade — migration 000004 tinha
-- movido pra grade_id (22/09, quando "grade" nasceu). Pedido direto do dono do produto
-- (16/10): "não deve ter opção de escolher guichê pra cada grade que estiver alocado... a
-- pessoa só escolhe um e já serve pra todos, pq ali é o lugar que ela vai atender as
-- demandas". Fisicamente um guichê é uma mesa/sala dentro de um local — várias grades (filas
-- de serviços diferentes) do mesmo local físico compartilham os mesmos guichês; só o guichê
-- de verdade, único por local, fazia sentido escolher uma vez.

alter table guiches add column unidade_id uuid references unidades(id);

update guiches gu set unidade_id = g.unidade_id
from grades g where g.id = gu.grade_id;

alter table guiches alter column unidade_id set not null;

-- Dedup: duas grades da mesma unidade podiam ter, cada uma, seu próprio guichê com o MESMO
-- nome (ex.: "Guichê 1" nasceu automaticamente tanto na grade principal quanto na grade de
-- encaixe de uma unidade real, confirmado no banco de dev antes de escrever esta migration) —
-- unificar por unidade_id criaria uma colisão de unicidade. Mantém, por nome duplicado, a
-- linha com mais histórico de verdade (soma de referências em agendamentos+chamadas) e
-- repointa as referências da(s) outra(s) antes de excluí-la(s) — nenhum dado histórico é
-- perdido, só passa a apontar pro guichê físico único que de fato representa.
create temporary table guiche_merge_map as
with contagem as (
  select gu.id,
         (select count(*) from agendamentos a where a.guiche_id = gu.id) +
         (select count(*) from chamadas c where c.guiche_id = gu.id) as total_refs
  from guiches gu
  where gu.excluido_em is null
),
ranking as (
  select gu.id, gu.unidade_id, gu.nome, co.total_refs,
         row_number() over (
           partition by gu.unidade_id, gu.nome
           order by co.total_refs desc, gu.id
         ) as posicao
  from guiches gu
  join contagem co on co.id = gu.id
  where gu.excluido_em is null
)
select r.id as perdedor, s.id as sobrevivente
from ranking r
join ranking s on s.unidade_id = r.unidade_id and s.nome = r.nome and s.posicao = 1
where r.posicao > 1;

update agendamentos a set guiche_id = m.sobrevivente
from guiche_merge_map m where a.guiche_id = m.perdedor;

update chamadas c set guiche_id = m.sobrevivente
from guiche_merge_map m where c.guiche_id = m.perdedor;

update guiches gu set excluido_em = now(), ocupado_por_usuario_id = null, ocupado_em = null
from guiche_merge_map m where gu.id = m.perdedor;

drop table guiche_merge_map;

alter table guiches drop constraint guiches_grade_id_nome_key;
alter table guiches drop column grade_id;

-- Unique PARCIAL (só entre guichês não-excluídos) — upgrade em relação à constraint antiga
-- (que era "dura", bloqueava reaproveitar um nome mesmo de um guichê já excluído). Necessário
-- aqui porque o dedup acima pode deixar, no mesmo (unidade_id, nome), uma linha excluída
-- convivendo com a sobrevivente ativa.
create unique index guiches_unidade_id_nome_ativo_key on guiches (unidade_id, nome) where excluido_em is null;
