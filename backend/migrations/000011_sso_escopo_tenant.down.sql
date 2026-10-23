drop index if exists mapeamento_externo_prefeitura_tipo_external_id_key;
alter table mapeamento_externo drop column prefeitura_id;
alter table mapeamento_externo add constraint mapeamento_externo_tipo_external_id_key unique (tipo, external_id);

drop index if exists usuarios_external_issuer_sub_key;
alter table usuarios drop column external_issuer;
alter table usuarios add constraint usuarios_external_sub_key unique (external_sub);
