alter table unidades
  add column repeticoes_chamada int not null default 3,
  add column intervalo_repeticao_segundos int not null default 5;

alter table guiches
  add column tipo text not null default 'guiche' check (tipo in ('guiche', 'sala')),
  add column andar text,
  add column capacidade int not null default 1;
