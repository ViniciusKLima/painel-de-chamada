-- Reversão best-effort: não há como "desfazer" o merge de guichês duplicados (migration up)
-- com fidelidade total — um guichê que passou a ser compartilhado por várias grades da mesma
-- unidade volta preso a só UMA delas (a de nome mais antigo/menor id). Aceitável só como
-- rollback de desenvolvimento, nunca pensado pra reverter dado de produção depois do merge.

alter table guiches add column grade_id uuid references grades(id);

update guiches gu set grade_id = (
  select g.id from grades g
  where g.unidade_id = gu.unidade_id and g.excluido_em is null
  order by g.id
  limit 1
);

alter table guiches alter column grade_id set not null;

drop index guiches_unidade_id_nome_ativo_key;
alter table guiches drop column unidade_id;
alter table guiches add constraint guiches_grade_id_nome_key unique (grade_id, nome);
