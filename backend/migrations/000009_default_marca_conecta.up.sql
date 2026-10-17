-- 24/09: alinhamento com o design system do Conecta Cidades — o default de cor pra uma
-- prefeitura NOVA (criada sem escolher cor ainda) passa a ser o valor canônico deles
-- (`#D52E5F`, "cada prefeitura tem a sua cor", mas esse é o usado nos mockups) em vez do
-- teal-azulado antigo. Prefeituras já existentes, com cor já configurada pelo admin
-- (ex.: Jaboatão, que já tem a própria cor escolhida na UI), NÃO são alteradas — isso só
-- muda o valor usado quando uma prefeitura nova é inserida sem informar `cor_destaque`/
-- `cor_clara` explicitamente, mesmo padrão já usado na migration 000007 pros defaults de
-- grade. `cor_clara` é o mesmo tom em ~12% sobre branco, mesma convenção do token CSS
-- `--cor-info-fundo`/`--cor-sucesso-fundo` (ver skill identidade-visual).
alter table prefeituras alter column cor_destaque set default '#d52e5f';
alter table prefeituras alter column cor_clara set default '#fae6ec';
