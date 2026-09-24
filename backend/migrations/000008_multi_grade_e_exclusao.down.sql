alter table guiches drop column excluido_em;
alter table grades drop column excluido_em;

alter table usuarios add column grade_id uuid references grades(id);

update usuarios u
set grade_id = (select ug.grade_id from usuario_grades ug where ug.usuario_id = u.id limit 1);

drop table usuario_grades;
