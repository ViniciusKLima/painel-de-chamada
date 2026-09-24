<!--
Sync Impact Report
Version change: (template, unratified) → 1.0.0
Rationale: Primeira ratificação formal — o projeto já operava sob estas regras de fato (ver
histórico em CLAUDE.md), mas nunca tinham sido escritas como constituição. Bump MAJOR por ser a
adoção inicial de uma constituição versionada (não há versão anterior pra comparar como PATCH/MINOR).
Principles adotados (7, número diferente do molde de 5 — ajustado ao conteúdo real do projeto):
  I.   Stack Fixa, Sem ORM
  II.  Dependências Mínimas e Justificadas
  III. Verificação Ponta a Ponta Antes de "Pronto" (NON-NEGOTIABLE)
  IV.  Esclarecer Antes de Decidir Modelo de Dados ou Escopo Ambíguo
  V.   Não Implementar Além do Combinado
  VI.  Documentação Contínua em CLAUDE.md e Skills
  VII. Simplicidade Sobre Abstração Prematura (YAGNI)
Added sections: "Fluxo de Trabalho de Mudanças" (era [SECTION_2_NAME]), "Padrões de Qualidade
Antes de Testar ao Vivo" (era [SECTION_3_NAME]).
Removed sections: nenhuma (primeira versão preenchida a partir do molde genérico).
Follow-up TODOs: nenhum placeholder deixado sem preencher.
Templates a revisar manualmente após esta ratificação: .specify/templates/plan-template.md,
.specify/templates/spec-template.md, .specify/templates/tasks-template.md — nenhum deles faz
referência direta a princípios específicos de projeto hoje, então não exigem edição imediata,
mas devem ser conferidos na próxima vez que forem usados pra garantir que não contradizem os
princípios acima (em especial III e IV).
-->

# Painel de Chamada — Jaboatão mais fácil Constitution

## Core Principles

### I. Stack Fixa, Sem ORM
O backend É Go (router `chi`, acesso a banco via `pgx` puro, migrations via `golang-migrate`);
o frontend É React + Vite + TypeScript; o banco É PostgreSQL. Acesso a dados é sempre SQL
explícito via `pgx`, nunca um ORM — cada entidade tem seu próprio arquivo de repository com
queries escritas à mão. Trocar qualquer uma dessas escolhas de stack, ou introduzir um ORM,
exige decisão explícita do dono do produto e uma amenda a esta constituição, não uma escolha
unilateral durante uma tarefa. Rationale: a stack foi decidida deliberadamente (ver skill
`stack-go-react-postgres`) depois de uma implementação anterior em Node/Fastify/Prisma ter sido
arquivada — reabrir essa decisão a cada tarefa custaria retrabalho e inconsistência de convenção
entre arquivos.

### II. Dependências Mínimas e Justificadas
Uma dependência nova só entra no projeto quando carrega peso real que não vale a pena reescrever
à mão, e cada uma precisa de uma justificativa registrada (ex.: `xuri/excelize` em vez de
`xlsx`/SheetJS por causa de CVEs de alta severidade sem correção). Na ausência dessa justificativa
clara, a solução correta é implementar à mão dentro da stack existente — como os gráficos SVG do
dashboard do gestor (sem biblioteca de charting) ou a sessão própria em JWT+cookie HttpOnly (sem
framework de auth completo). Rationale: o projeto é mantido por uma pessoa só por enquanto; cada
dependência extra é superfície de manutenção, segurança e aprendizado que o próximo dev também
precisa carregar.

### III. Verificação Ponta a Ponta Antes de "Pronto" (NON-NEGOTIABLE)
Nenhuma mudança de fluxo (endpoint novo, tela nova, regra de negócio alterada) é considerada
concluída só porque `go build`/`go vet`/`gofmt` ou `tsc --noEmit`/`npm run build` passaram limpos.
Build e type-check limpos são pré-requisito, não critério de conclusão — a mudança só está pronta
depois de testada de ponta a ponta com dado real, via `curl` contra o backend rodando e/ou
interação ao vivo no navegador (Chrome), cobrindo o caminho feliz e pelo menos um caso de borda
relevante (concorrência, transição de estado inválida, dado vazio/nulo). Rationale: bugs reais
deste projeto (slice nil virando `null` no JSON, `PATCH` faltando no CORS, cookie de sessão
cross-site) só apareceram sob teste ao vivo — nenhum deles seria pego por type-check ou build.

### IV. Esclarecer Antes de Decidir Modelo de Dados ou Escopo Ambíguo
Quando um pedido ou wireframe implica uma decisão de modelo de dados ou de escopo que admite mais
de uma interpretação razoável — especialmente quando escolher errado custaria reescrever uma
migration inteira — a pergunta deve ser feita ao dono do produto antes de escrever schema ou
código, nunca resolvida por suposição silenciosa. Rationale: o caso real que fundamenta este
princípio foi a pergunta sobre o que "grade de horário" significava em relação a "unidade" antes
da migration `000004_grades` — presumir a resposta errada ali teria custado uma reestruturação de
schema inteira já em produção de teste.

### V. Não Implementar Além do Combinado
Quando o dono do produto sinaliza que um próximo tópico será discutido depois (ex.: "vamos
prosseguir pra tela do admin e o fluxo de acesso dos perfis" — ainda não desenhado), esse tópico
não é implementado adiantado, mesmo que pareça óbvio ou reaproveitável a partir do que já existe.
Da mesma forma, uma tarefa de correção ou ajuste pontual não é oportunidade para refatorar,
generalizar ou adicionar funcionalidade não pedida. Rationale: escopo adiantado sem o
desenho/conversa correspondente já se mostrou, neste projeto, fonte de retrabalho quando o desenho
real chega diferente do presumido.

### VI. Documentação Contínua em CLAUDE.md e Skills
Toda decisão de produto, mudança de rumo, bug real encontrado e corrigido, e resultado de teste ao
vivo é registrado em `CLAUDE.md` como uma rodada datada, e o conteúdo estrutural correspondente
(schema, regras de negócio, papéis e telas, convenções de stack) é mantido atualizado nas skills
de `.claude/skills/`. Nenhuma dessas duas fontes substitui a outra: `CLAUDE.md` é o histórico
narrativo de como se chegou até aqui, as skills são o estado atual consolidado. Rationale: o
projeto é tocado por uma pessoa só, mas precisa ser retomável por outro dev (ou por uma sessão
futura deste mesmo assistente) só lendo esses arquivos, sem depender de contexto que só existe na
cabeça de quem implementou.

### VII. Simplicidade Sobre Abstração Prematura (YAGNI)
Resolver o problema real da frente, do jeito mais direto que a stack permite — sem camada
genérica, sem feature flag, sem parametrização para um caso hipotético que ainda não apareceu.
Três linhas parecidas são preferíveis a uma abstração prematura; metadados descritivos (como
`tipo`/`andar`/`capacidade` de um guichê) não implicam automaticamente uma regra de negócio nova
até que essa regra seja pedida explicitamente. Rationale: mesmo padrão do princípio V aplicado ao
nível de código — este projeto já documentou casos onde generalizar cedo (ex.: presumir que
"sala" com capacidade > 1 permitiria múltiplas chamadas simultâneas) teria sido especulação, não
necessidade.

## Fluxo de Trabalho de Mudanças

Antes de implementar qualquer mudança que toque regra de negócio, schema ou fluxo entre papéis,
consultar a skill relevante em `.claude/skills/` e a seção mais recente de `CLAUDE.md` — a decisão
mais recente sempre supera uma anterior no mesmo arquivo. Specs formais em `specs/` (fluxo
spec-kit: `/speckit-specify` → `/speckit-clarify` → `/speckit-plan` → `/speckit-tasks` →
`/speckit-implement`) são usadas para mudanças estruturais grandes o suficiente pra precisar de um
documento de especificação próprio (ex.: `001-fila-atendimento-presencial`,
`002-importacao-multi-local`, `003-painel-tv`); ajustes pontuais de UI/UX ou correções de bug não
precisam de spec formal, só do registro em `CLAUDE.md` (Princípio VI). Depois de implementar,
verificar ao vivo (Princípio III) antes de reportar a tarefa como concluída.

## Padrões de Qualidade Antes de Testar ao Vivo

Antes de qualquer verificação ao vivo, o código precisa passar limpo por: `go build ./...`,
`go vet ./...` e `gofmt` no backend; `npx tsc --noEmit -p tsconfig.app.json` (nunca `tsc --noEmit`
sem `-p`, que não verifica nada sob o `tsconfig.json` deste projeto) e `npm run build` no
frontend. Strings com acentuação (nomes de guichê, nomes de cidadão) devem ser verificadas via UI
real ou bytes UTF-8 explícitos, nunca só via `curl` direto no terminal Windows, que corrompe
acentuação por causa do encoding do shell — isso não é um bug do backend quando acontece. Qualquer
endpoint novo que use um método HTTP ainda não usado no projeto (ex.: `PATCH`, `PUT`) deve
verificar se esse método já está na lista de `Access-Control-Allow-Methods` do CORS de
desenvolvimento — já houve um caso real de 503 silencioso por método faltando ali.

## Governance

Esta constituição tem precedência sobre convenções informais anteriores registradas em
`CLAUDE.md` ou nas skills sempre que houver conflito direto; onde não há conflito, `CLAUDE.md` e
as skills continuam sendo a fonte de detalhe operacional que esta constituição não repete.
Emendas são feitas via `/speckit-constitution`, nunca editando o arquivo à mão fora desse fluxo,
para garantir que o relatório de impacto de sincronização seja sempre gerado. Versionamento
semântico: MAJOR para remoção ou redefinição incompatível de um princípio existente; MINOR para
adição de um princípio ou seção nova, ou expansão material de uma regra existente; PATCH para
correção de redação, typo ou esclarecimento sem mudança de regra. Como o projeto é mantido por
uma pessoa só (o dono do produto, com o assistente implementando), a "revisão de conformidade" é
o próprio dono do produto confirmando o resultado do teste ao vivo (Princípio III) antes de dar
uma mudança por aceita — não há processo de PR/aprovação por terceiros neste momento do piloto.

**Version**: 1.0.0 | **Ratified**: 2026-09-21 | **Last Amended**: 2026-09-21
