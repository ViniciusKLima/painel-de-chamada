-- Ocupação de guichê (22/09): quando um atendente escolhe um guichê no modal de seleção,
-- outros atendentes da mesma grade precisam ver aquele guichê como "ocupado" e não
-- conseguir escolhê-lo também — hoje a escolha só era lembrada em localStorage (local ao
-- navegador de cada um, sem visibilidade entre atendentes). `ocupado_em` funciona como
-- heartbeat: o frontend reafirma a ocupação periodicamente (ver skill regras-negocio-fila);
-- um registro "velho" demais (sem heartbeat recente) é tratado como liberado automaticamente
-- na consulta, sem precisar de job de limpeza — cobre o caso de aba fechada/crash sem logout.
alter table guiches add column ocupado_por_usuario_id uuid references usuarios(id);
alter table guiches add column ocupado_em timestamptz;
