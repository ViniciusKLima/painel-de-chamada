delete from usuarios where email = 'admin.teste@local.dev';

alter table prefeituras drop column cor_destaque;
alter table prefeituras drop column cor_clara;

alter table usuarios drop constraint usuarios_email_key;

-- Melhor esforço: devolve o gestor pra primeira unidade da sua secretaria (perde
-- granularidade se ele tivesse acesso a várias unidades reais — inerente à reversão).
update usuarios u
set unidade_id = (select id from unidades where secretaria_id = u.secretaria_id order by nome limit 1)
where u.papel = 'gestor';

alter table usuarios drop column secretaria_id;
alter table usuarios alter column unidade_id set not null;
alter table usuarios add constraint usuarios_unidade_id_email_key unique (unidade_id, email);

alter table usuarios drop constraint usuarios_papel_check;
alter table usuarios add constraint usuarios_papel_check
  check (papel in ('gestor', 'atendente', 'recepcionista'));
