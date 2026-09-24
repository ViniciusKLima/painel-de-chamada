-- Alocação de múltiplas grades por usuário (23/09, pedido do dono do produto: "atendente ou
-- recepção podem ser alocados em mais de uma grade"). Antes era `usuarios.grade_id`, uma FK
-- nullable pra UMA grade só — vira uma tabela N:N. Recepcionista continua trabalhando no
-- nível de UNIDADE (decisão de 22/09, não revertida aqui) — pra ela, uma linha aqui é só um
-- filtro opcional de "em quais grades específicas ela pode cadastrar encaixe"; sem nenhuma
-- linha, continua vendo/cadastrando em qualquer grade da própria unidade, como já era.
create table usuario_grades (
    usuario_id uuid not null references usuarios(id) on delete cascade,
    grade_id uuid not null references grades(id) on delete cascade,
    primary key (usuario_id, grade_id)
);

insert into usuario_grades (usuario_id, grade_id)
select id, grade_id from usuarios where grade_id is not null;

alter table usuarios drop constraint usuarios_grade_id_fkey;
alter table usuarios drop column grade_id;

-- Exclusão de grade/guichê (23/09, pedido: "quero que seja possível excluir"). Os dois podem
-- ter histórico em `chamadas`/`agendamentos` (logs append-only, nunca apagados de propósito —
-- ver skill modelo-dados) então um DELETE de verdade quebraria a integridade referencial na
-- certa (já documentado como limitação conhecida desde a rodada do dia 23/09 que tentou
-- apagar 2 atendentes de teste). Em vez disso, "excluir" grava este timestamp; toda listagem/
-- consulta ativa passa a filtrar `excluido_em is null`. O dado em si nunca some do banco, só
-- some da UI — histórico permanece íntegro. Distinto do `ativo` que os dois já tinham: `ativo`
-- é a desativação reversível de sempre (some da seleção do atendente mas continua na tela de
-- gestão pra reativar); `excluido_em` some da tela de gestão também.
--
-- Usuário não ganhou uma coluna nova pra isso — `usuarios.ativo` já existe desde o início,
-- já bloqueia login (`internal/auth/context.go`), e nunca tinha um botão de UI pra alternar
-- (nenhuma tela expõe "desativar temporariamente" um usuário) — "excluir" reaproveita esse
-- campo em vez de criar uma segunda coluna redundante pro mesmo efeito prático.
alter table grades add column excluido_em timestamptz;
alter table guiches add column excluido_em timestamptz;
