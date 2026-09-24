-- Hierarquia de acesso do Admin (22/09): Admin → Prefeitura → Secretaria → Usuários. O
-- Admin é master da plataforma inteira, sem vínculo com prefeitura/secretaria/unidade
-- nenhuma. O Gestor deixa de ser vinculado a uma UNIDADE (o que só funcionava porque o
-- piloto tem uma unidade só) e passa a ser vinculado à SECRETARIA inteira — pode enxergar
-- todas as unidades/grades dela, não só uma. Atendente/recepcionista continuam vinculados a
-- uma unidade (recepcionista) ou unidade+grade (atendente), como já era — ver skill
-- modelo-dados pro raciocínio completo dessa decisão.

alter table usuarios drop constraint usuarios_papel_check;
alter table usuarios add constraint usuarios_papel_check
  check (papel in ('admin', 'gestor', 'atendente', 'recepcionista'));

-- unidade_id vira opcional: admin não tem nenhuma, gestor não tem mais (ver abaixo).
alter table usuarios alter column unidade_id drop not null;

alter table usuarios add column secretaria_id uuid references secretarias(id);

-- Backfill: todo gestor existente ganha a secretaria da sua unidade atual, e perde o
-- vínculo direto com aquela unidade específica (a autoridade dele agora é sobre a
-- secretaria inteira).
update usuarios u
set secretaria_id = un.secretaria_id
from unidades un
where u.unidade_id = un.id and u.papel = 'gestor';

update usuarios set unidade_id = null where papel = 'gestor';

-- Email único globalmente, não só por unidade — o login já busca por email sem escopo de
-- unidade desde o MVP (skill auth-login-mvp, "assumido que email é único na prática"), e
-- o admin não tem unidade_id nenhuma pra ancorar a constraint antiga.
alter table usuarios drop constraint usuarios_unidade_id_email_key;
alter table usuarios add constraint usuarios_email_key unique (email);

-- As duas cores de identidade configuráveis pelo admin por prefeitura: a de destaque
-- (ações/links/aba ativa, mapeia pro token --cor-primaria do frontend) e a mais clara
-- (fundo/cabeçalho de tabela, mapeia pro token --cor-info-fundo). Default = paleta atual,
-- pra nenhuma prefeitura existente mudar de aparência até o admin escolher outra cor.
alter table prefeituras add column cor_destaque text not null default '#146c8c';
alter table prefeituras add column cor_clara text not null default '#e4f1f5';

-- Usuário admin de teste do piloto — sem unidade/secretaria, acesso master. Senha: teste123
-- (mesmo padrão dos outros usuários de teste, ver seed/usuarios_teste.sql).
insert into usuarios (id, nome, email, senha_hash, papel, ativo)
values (
  'a0000000-0000-0000-0000-000000000099',
  'Admin Teste',
  'admin.teste@local.dev',
  crypt('teste123', gen_salt('bf')),
  'admin',
  true
)
on conflict (email) do nothing;
