create extension if not exists pgcrypto;

create table prefeituras (
  id uuid primary key default gen_random_uuid(),
  nome text not null,
  slug text not null unique,
  issuer_jwt text unique
);

create table secretarias (
  id uuid primary key default gen_random_uuid(),
  prefeitura_id uuid not null references prefeituras(id),
  nome text not null,
  sigla text not null,
  unique (prefeitura_id, sigla)
);

create table unidades (
  id uuid primary key default gen_random_uuid(),
  secretaria_id uuid not null references secretarias(id),
  nome text not null,
  sla_chegada_minutos int not null default 60,
  sla_atendimento_minutos int not null default 20,
  duracao_chamada_painel_segundos int not null default 30,
  duracao_chamada_painel_fila_segundos int not null default 15,
  unique (secretaria_id, nome)
);

create table usuarios (
  id uuid primary key default gen_random_uuid(),
  unidade_id uuid not null references unidades(id),
  nome text not null,
  email text not null,
  senha_hash text,
  papel text not null check (papel in ('gestor', 'atendente', 'recepcionista')),
  external_sub text unique,
  ativo boolean not null default true,
  unique (unidade_id, email)
);

create table guiches (
  id uuid primary key default gen_random_uuid(),
  unidade_id uuid not null references unidades(id),
  nome text not null,
  ativo boolean not null default true,
  unique (unidade_id, nome)
);

create table lotes_importacao (
  id uuid primary key default gen_random_uuid(),
  secretaria_id uuid not null references secretarias(id),
  usuario_id uuid not null references usuarios(id),
  arquivo_nome text not null,
  data_upload timestamptz not null default now(),
  status text not null check (status in ('processando', 'concluido', 'erro')),
  total_linhas int,
  total_erros int
);

create table agendamentos (
  id uuid primary key default gen_random_uuid(),
  unidade_id uuid not null references unidades(id),
  lote_id uuid references lotes_importacao(id),
  protocolo text not null,
  nome_cidadao text not null,
  cpf text,
  telefone text,
  servico text,
  grade_horario text,
  tipo text not null check (tipo in ('agendado', 'encaixe')),
  horario_previsto timestamptz,
  chegada_em timestamptz,
  prioridade text check (prioridade in ('no_horario', 'atrasado')),
  guiche_id uuid references guiches(id),
  status text not null default 'aguardando_chegada' check (status in (
    'aguardando_chegada', 'sala_espera', 'chamado', 'atendido', 'ausente', 'cancelado'
  )),
  atendido_em timestamptz,
  ausente_em timestamptz,
  cancelado_em timestamptz,
  motivo_cancelamento text,
  criado_em timestamptz not null default now(),
  unique (unidade_id, protocolo)
);

create table chamadas (
  id uuid primary key default gen_random_uuid(),
  agendamento_id uuid not null references agendamentos(id),
  guiche_id uuid not null references guiches(id),
  usuario_id uuid not null references usuarios(id),
  chamado_em timestamptz not null default now()
);

create table mapeamento_externo (
  id uuid primary key default gen_random_uuid(),
  tipo text not null check (tipo in ('department', 'unit')),
  external_id text not null,
  secretaria_id uuid references secretarias(id),
  unidade_id uuid references unidades(id),
  unique (tipo, external_id)
);

create index on agendamentos (unidade_id, status);
create index on agendamentos (unidade_id, tipo, horario_previsto);
create index on chamadas (chamado_em desc);
create index on chamadas (agendamento_id, chamado_em desc);
