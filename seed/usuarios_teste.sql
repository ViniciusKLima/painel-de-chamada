-- Seed de teste para o MVP: unidade piloto + 3 usuários fictícios (um por papel), com
-- senha de verdade (via pgcrypto, compatível com bcrypt do Go) pra testar o login real
-- (skill auth-login-mvp). Senha de todos: "teste123".
--
-- A unidade usa o MESMO nome de "Local no mapa" do export real analisado (planilha de
-- 19/09) — "Cadastro Único Sede Prazeres" — pra que a importação real caia direto na
-- fila desses usuários de teste, sem precisar criar unidade nova nem trocar quem loga.

insert into prefeituras (id, nome, slug) values
  ('00000000-0000-0000-0000-000000000001', 'Jaboatão dos Guararapes', 'jaboatao')
on conflict (id) do nothing;

insert into secretarias (id, prefeitura_id, nome, sigla) values
  ('00000000-0000-0000-0000-000000000002', '00000000-0000-0000-0000-000000000001', 'Secretaria de Assistência Social e Cidadania', 'SASC')
on conflict (id) do nothing;

insert into unidades (id, secretaria_id, nome) values
  ('00000000-0000-0000-0000-000000000003', '00000000-0000-0000-0000-000000000002', 'Cadastro Único Sede Prazeres')
on conflict (id) do nothing;

insert into usuarios (id, unidade_id, nome, email, senha_hash, papel, ativo) values
  ('00000000-0000-0000-0000-000000000010', '00000000-0000-0000-0000-000000000003', 'Gestor Teste', 'gestor.teste@local.dev', crypt('teste123', gen_salt('bf')), 'gestor', true),
  ('00000000-0000-0000-0000-000000000011', '00000000-0000-0000-0000-000000000003', 'Atendente Teste', 'atendente.teste@local.dev', crypt('teste123', gen_salt('bf')), 'atendente', true),
  ('00000000-0000-0000-0000-000000000012', '00000000-0000-0000-0000-000000000003', 'Recepção Teste', 'recepcao.teste@local.dev', crypt('teste123', gen_salt('bf')), 'recepcionista', true)
on conflict (id) do nothing;

insert into guiches (id, unidade_id, nome) values
  ('00000000-0000-0000-0000-000000000020', '00000000-0000-0000-0000-000000000003', 'Guichê 1'),
  ('00000000-0000-0000-0000-000000000021', '00000000-0000-0000-0000-000000000003', 'Guichê 2')
on conflict (id) do nothing;
