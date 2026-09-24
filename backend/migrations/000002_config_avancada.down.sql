alter table guiches
  drop column tipo,
  drop column andar,
  drop column capacidade;

alter table unidades
  drop column repeticoes_chamada,
  drop column intervalo_repeticao_segundos;
