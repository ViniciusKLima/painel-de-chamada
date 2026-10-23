-- Corrige 2 lacunas de isolamento multi-tenant achadas por revisão de segurança (22/10),
-- no mesmo dia da implementação do SSO (ver skill auth-conecta-cidades, "Implementação real"):
--
-- 1. usuarios.external_sub era único GLOBALMENTE, sem escopo de emissor. `sub` é um ID
--    interno por-município da Conecta Cidades (ex. "42", inteiro pequeno) — duas prefeituras
--    diferentes podem perfeitamente ter um servidor com sub="42" cada uma. Sem escopo, um
--    login de uma prefeitura podia colidir com a conta de um servidor de OUTRA, abrindo
--    caminho pra account takeover entre tenants.
--
-- 2. mapeamento_externo (department_id/unit_id -> secretaria/unidade) também não tinha
--    escopo de prefeitura — mesmo problema: um department_id=3 de uma prefeitura podia
--    resolver pro mapeamento cadastrado por OUTRA, vazando acesso entre tenants.

alter table usuarios add column external_issuer text;

-- Drop a unique antiga (sem escopo) e cria a nova, composta, parcial (só entre linhas que
-- de fato vieram de SSO — a maioria dos usuários continua sem external_sub/external_issuer).
alter table usuarios drop constraint usuarios_external_sub_key;
create unique index usuarios_external_issuer_sub_key on usuarios (external_issuer, external_sub)
  where external_issuer is not null and external_sub is not null;

alter table mapeamento_externo add column prefeitura_id uuid references prefeituras(id);
alter table mapeamento_externo drop constraint mapeamento_externo_tipo_external_id_key;
-- NOT NULL exigido daqui pra frente — toda linha nova precisa saber de qual prefeitura é.
-- (Não havia nenhuma linha gravada nesta tabela até agora, então não há dado legado a migrar.)
alter table mapeamento_externo alter column prefeitura_id set not null;
create unique index mapeamento_externo_prefeitura_tipo_external_id_key
  on mapeamento_externo (prefeitura_id, tipo, external_id);
