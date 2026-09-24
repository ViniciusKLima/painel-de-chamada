# Painel de Chamada — Jaboatão mais fácil

Contexto do projeto para quem (ou qual Claude/dev) for continuar a partir daqui.

**Este arquivo foi reescrito em 17/09** a partir de uma pasta de contexto (`contexto-claude-code/`)
trazida pelo usuário, que ele confirmou ser **a fonte de verdade mais atual** — supera as decisões
anteriores deste mesmo arquivo (que eram baseadas em Node/Fastify/Prisma e em 3 papéis
admin/gestor/atendente sem separar recepção). O histórico completo da versão anterior, incluindo
a análise de uma planilha real de agendamentos, está preservado em
`contexto-claude-code/` e na seção "Aprendizados da versão anterior (Node)" mais abaixo — não
descartar esse conhecimento, ele continua valendo mesmo com a troca de stack.

**Leia também os arquivos em `.claude/skills/`** — cada um cobre uma parte específica em mais
detalhe. Carregam automaticamente em qualquer sessão futura neste projeto.

- `modelo-dados` — schema Postgres completo, fonte da verdade
- `regras-negocio-fila` — fluxo de chegada/sala de espera/chamada, prioridade, timeouts
- `papeis-e-telas` — o que cada papel vê e faz
- `stack-go-react-postgres` — bibliotecas e convenções
- `importacao-planilha` — mapeamento de colunas real da planilha → schema
- `painel-tv` — tela pública de exibição de chamadas
- `auth-login-mvp` — login real do MVP (email+senha, sem SSO)
- `auth-conecta-cidades` — fluxo real de SSO (JWT via URL → cookie de sessão), **adiado pro pós-MVP**
- `api-conecta-cidades` — API externa de agendamentos da plataforma (limitações e uso)
- `identidade-visual` — paleta, tipografia, espaçamento e regras de layout do frontend (cabeçalho, tabelas, botões)

**Specs formais (spec-kit) em `specs/`**:
- `001-fila-atendimento-presencial` — recepção, sala de espera, atendente, dashboard do gestor
- `002-importacao-multi-local` — importação de planilha com roteamento automático por local
- `003-painel-tv` — exibição pública de chamadas

Todos os 3 specs estão com os checklists de qualidade fechados (sem `NEEDS CLARIFICATION`
pendente) — prontos pra `/speckit-plan` quando for a hora de implementar.

## O que é

Sistema de chamada de senha/nome para salas de espera das unidades da Prefeitura de Jaboatão
dos Guararapes, integrado à plataforma "Jaboatão mais fácil". Substitui o chamado gritado por
um painel de TV que exibe a chamada atual e as últimas 10 chamadas.

Projeto tocado sozinho por enquanto (outro dev deve entrar depois — por isso este contexto
está sendo formalizado em skills, não só na cabeça de quem começou).

## Escopo do MVP

Uma única unidade: **CRAS Piloto** (dentro da SASC). Sem interface multi-secretaria ativa
ainda, mas o modelo de dados já nasce com a hierarquia completa (ver skill `modelo-dados`).

## Stack (atualizado)

- **Backend**: Go (router `chi`, acesso a banco via `pgx`, migrations via `golang-migrate`)
- **Frontend**: React + Vite + TypeScript
- **Banco**: PostgreSQL
- Ver skill `stack-go-react-postgres` para convenções detalhadas e o que evitar.

## Papéis (atualizado — 3 níveis)

`gestor`, `atendente`, `recepcionista` — cada um com tela própria e permissão isolada.
Ver skill `papeis-e-telas` para o detalhe de cada um.

Resumo do fluxo: recepção importa planilha → cidadão chega → recepção confirma chegada →
vai pra sala de espera → atendente vê só a sala de espera (nunca a lista completa) → atendente
chama. Ver skill `regras-negocio-fila` para o fluxo completo e a regra de prioridade entre
agendados e encaixes.

## Identificação do cidadão

Identificação do dia a dia por **protocolo** (não por CPF). CPF é guardado quando presente na
planilha (campo `agendamentos.cpf`, ver `modelo-dados`). **Decisão final (19/09): sem máscara** —
"não precisa mascarar nada, pq os atendentes já têm acesso a esses dados na outra plataforma".
CPF/telefone ficam visíveis pra qualquer usuário autenticado, sem distinção de papel.

## Autenticação

**Decisão (19/09): o MVP usa login simples (email+senha), não o SSO real.** "Essas questões do
JWT ainda serão discutidas, mas pro MVP não vamos usar, vamos passar com login mesmo." Ver skill
`auth-login-mvp` pro que implementar de fato agora.

O fluxo real de SSO da plataforma (JWT via `?token=` na URL → cookie de sessão, documentado em
18/09 a partir de um produto irmão, "Emissão de Carteiras") continua todo desenhado na skill
`auth-conecta-cidades`, mas **fica adiado pro pós-MVP** — não implementar durante o piloto.

## Hospedagem (fase piloto, orçamento zero — decisão mantida)

Sem cartão de crédito até a prefeitura/empresa assumir infra oficial:
- Banco: Neon (Postgres serverless, tier gratuito)
- Backend: Render (free tier via Docker — dorme após ~15min ocioso)
- Frontend: Vercel ou Netlify

Tudo containerizado com Docker desde o início.

## Estado atual

- Regras de negócio e modelo de dados formalizados nas skills.
- Seed de teste disponível em `/seed/usuarios_teste.sql` (unidade CRAS Piloto + 3 usuários
  fictícios, um por papel).
- Documentação também mantida no Notion: página "Painel de Chamada - Jaboatão mais fácil"
  (pode estar desatualizada em relação a este arquivo — este é o mais recente).
- **Ambiente de dev (17/09)**: `docker-compose up -d` sobe Postgres (porta **5433** externa —
  5432 colide com um PostgreSQL nativo já instalado na máquina) e Redis (6379). Docker Desktop
  precisa estar aberto manualmente antes (não inicia sozinho no boot desta máquina). A máquina
  tem histórico de ficar sem memória (containers/servidores de dev já foram derrubados por OOM
  mais de uma vez) — se algo cair sozinho, provavelmente é isso; fechar programas pesados
  (Chrome com muitas abas) costuma resolver.
- **Implementação anterior em Node/Fastify/Prisma arquivada em `_archive/node-legado/`** (17/09)
  — não tem git neste projeto ainda, então optei por mover em vez de apagar, pra não perder
  trabalho testado sem ter como recuperar. Contém `backend/` (Node completo) e
  `frontend-paginas-antigas/` (as telas antigas de `/pages` + `api.ts`, que assumiam o modelo de
  3 papéis sem recepcionista e chamavam direto os endpoints REST antigos). Serve só de consulta
  (ex: como o parser de planilha real foi resolvido, como funcionava o mascaramento de CPF) —
  não é o ponto de partida do novo backend nem do novo frontend.
- **Banco Postgres resetado (17/09)**: o container `painel_postgres` tinha o schema antigo
  (Node/Prisma) com dado de teste real (CPF/telefone das 570 linhas importadas). Como o schema
  mudou de nome/estrutura (`locais`→`unidades`, `agendamentos` ganhou `tipo`/`chegada_em`, etc.)
  e os dados eram só de teste, dropei e recriei o banco `painel_chamada` do zero.
- **Backend Go: rodando de verdade (17/09)** — Go 1.27.1 instalado, `go mod tidy` resolveu as
  dependências, `go build ./cmd/api` compilou de primeira, e o binário sobe e responde
  `GET /health` → `{"status":"ok","db":true}` conectado no Postgres real (porta 5433). Em `backend/`:
  - Estrutura de pastas conforme skill `stack-go-react-postgres`: `cmd/api/`, `internal/{handlers,repository,domain,auth}/`, `migrations/`.
  - `migrations/000001_init.up.sql` / `.down.sql` — schema completo transcrito da skill `modelo-dados`, testado via `psql` direto no Postgres; `seed/usuarios_teste.sql` também rodou limpo em cima (3 usuários, 1 por papel).
  - `go.mod` (module `painel-chamada-backend`, go 1.22, deps chi + pgx) e `cmd/api/main.go` (rota `/health`) — **compilado e testado**, único endpoint existente até agora.
  - `.env` (copiado do `.env.example`) com `DATABASE_URL` (porta 5433) — variáveis do JWT real (`JWT_ISSUER` etc.) não são mais necessárias pro MVP, ver decisão em "Autenticação".
  - Próximo passo real de código: implementar o login (skill `auth-login-mvp`) e os endpoints de recepção (skill `papeis-e-telas`), rodando as migrations via `golang-migrate` de verdade em vez de aplicar a SQL manualmente como fizemos pra testar.
- **Frontend: reorganizado pra estrutura de telas por papel** (17/09), conforme skill `stack-go-react-postgres`:
  - `src/telas/{login,gestor,recepcionista,atendente,painel-tv}/` — cada uma só com um placeholder "Em construção" por enquanto, referenciando a skill relevante. Rotas em `App.tsx` já apontam pra elas (`/login`, `/gestor`, `/recepcao`, `/sala-espera`, `/painel/:unidadeId`).
  - `@tanstack/react-query` instalado e com `QueryClientProvider` já configurado em `main.tsx` (pro polling da sala de espera/dashboard, conforme a skill manda).
  - Build de produção (`npm run build`) testado e limpo, sem erros de tipo.
- **Pasta `contexto-claude-code/` removida** — conteúdo já totalmente absorvido em `.claude/skills/`, `/seed/` e neste `CLAUDE.md`, então ficou redundante.
- **Planilha real com colunas novas (19/09)**: um novo export trouxe 3 colunas que não existiam
  antes — `Grade de horários`, `Local no mapa`, `Atendido por`. Analisado em detalhe na skill
  `importacao-planilha`. Decisões tomadas: `Local no mapa` é a chave de roteamento pra unidade
  (cria a unidade automaticamente se não existir ainda), `Grade de horários` é só atributo
  informativo (não separa fila), `Atendido por` reconhecido mas não mapeado ainda (sempre vazio
  nos agendamentos futuros vistos até agora). Isso também corrigiu um erro de modelagem: 
  `lotes_importacao` estava ligado a `unidade_id`, mas devia ser `secretaria_id` (um upload cobre
  várias unidades de uma vez) — corrigido no `modelo-dados`.
- **Fluxo operacional completo do painel de chamada e do painel de TV levantados com o dono do
  produto (19/09)** — SLA de chegada/atendimento/exibição configuráveis por unidade, prioridade
  no-horário vs. atrasado+encaixe competindo por chegada, ausência automática em dois estágios,
  fila de exibição do painel em memória (sem campo booleano), nome parcial mantido no painel
  público. Tudo formalizado nas skills `regras-negocio-fila`, `papeis-e-telas`, `painel-tv` e nos
  specs 001/003.

## API externa "Conecta Cidades" (17/09)

O usuário trouxe a documentação da API externa da plataforma (mesmo produto por trás do
"Jaboatão mais fácil"/"Jornada" — nome comercial "Conecta Cidades"). Documentado em detalhe na
skill `api-conecta-cidades`; spec OpenAPI completo salvo em
`_docs-externas/conecta-cidades-openapi.json`.

⚠️ **Achado importante**: essa API é toda escopada por `citizen_cpf` (feita pra bots agirem em
nome de UM cidadão) — **não existe endpoint pra listar os agendamentos de forma geral/automática**
(o que o painel precisaria pra substituir o upload manual de planilha). **O filtro por
local/setor é responsabilidade do nosso próprio sistema, não da API externa** — não pedir pra
plataforma filtrar por local; o que falta é só a listagem em massa (do dia, ou geral), sem exigir
CPF por agendamento. Já foi aberta uma issue no GitLab da plataforma (17/09, com o Mateus) pedindo
esse endpoint interno/admin — resposta: "não temos atualmente", vai analisar viabilidade. Até
sair uma resposta, o upload manual continua sendo o caminho.

Também confirma (com nomes oficiais da fonte) o que a análise da planilha real já sugeria:
enum de status do agendamento na origem é `scheduled/attended/cancelled_by_citizen/
cancelled_by_user/no_show` — não bate 1:1 com o enum da skill `modelo-dados`
(`aguardando_chegada/sala_espera/chamado/atendido/ausente`); e o conceito de "dependente" é
real no domínio (schema `Dependent`, com CPF próprio quando cadastrado).

## Próximos passos sugeridos

1. ~~Resolver a tensão do CPF~~ ✅ resolvida (19/09) — sem máscara, ver "Identificação do cidadão".
2. ~~Scaffold do backend Go~~ ✅ feito e rodando de verdade (17/09).
3. ~~Migrations a partir do schema em `modelo-dados`~~ ✅ reescritas do zero e testadas (19/09).
4. ~~Login do MVP~~ ✅ implementado e testado (19/09) — `POST /api/v1/login`, skill `auth-login-mvp`.
5. ~~Endpoints de recepção e atendente~~ ✅ implementados e testados de ponta a ponta (19/09),
   incluindo concorrência real e ausência automática — ver seção "Backend implementado e
   testado" acima. Spec formal: `specs/001-fila-atendimento-presencial/spec.md`.
6. ~~Perguntar à equipe da plataforma Conecta Cidades se existe uma API interna/admin~~ ✅ issue
   aberta no GitLab da plataforma (17/09) — resposta inicial do Mateus: "não temos atualmente",
   vai analisar viabilidade. Acompanhar retorno; decide se o upload manual de planilha continua
   sendo permanente ou é só uma ponte até a integração real (ver skill `api-conecta-cidades`).
7. ~~Conectar o frontend aos endpoints novos~~ ✅ as 5 telas foram conectadas e testadas no
   navegador (19/09, quarta rodada) — ver seção "Frontend conectado e testado no navegador".
8. ~~Endpoint de escrita pra gestor configurar SLA/guichês de uma unidade~~ ✅ implementado,
   testado no navegador e com um bug de CORS real corrigido no processo (19/09, sexta rodada)
   — ver seção "Tela Gerenciar CRAS do gestor".
9. ~~Áudio/voz do painel de TV~~ ✅ implementado E verificado de verdade (19/09, quinta rodada,
   testado ao vivo no Chrome) — ver seção "Voz do painel verificada de verdade".

## Aprendizados da versão anterior (Node) — ainda válidos, independem da stack

Vieram da análise de um export real da plataforma Jornada (570 agendamentos de um dia da SASC,
14/09). São fatos sobre os dados de origem, não decisões de arquitetura — continuam relevantes
pro parser de planilha em Go:

- **Colunas reais da planilha**: `Protocolo, Data, Horário, Cidadão (responsável), CPF,
  Agendamento para (dependente), Serviço, Status, Confirmado em, Telefone, Cancelado em,
  Motivo do cancelamento, Observações, Notas do atendimento`.
- **CPF e telefone vêm preenchidos em quase 100% das linhas** (ver tensão não resolvida acima).
- **`Status` da planilha (`Agendado` / `Cancelado pelo cidadão`) é sobre o agendamento em si**,
  não sobre onde a pessoa está na fila física — mapear com cuidado pro novo enum
  (`aguardando_chegada/sala_espera/chamado/atendido/ausente`, que nem tem um estado equivalente
  a "cancelado pelo cidadão"; precisa decidir onde esses casos entram, ou se ficam de fora).
- **Dentro do "Motivo do cancelamento" tem reagendamentos disfarçados de cancelamento**: texto
  tipo "Reagendado: Local alterado de 'CRAS - Prazeres' para 'CRAS - Curado'; Data/hora alterada
  para 08 de Setembro..." — ou seja, o cidadão não desistiu, só mudou de dia/unidade. Se isso
  não for separado de cancelamento de verdade, distorce qualquer métrica de "não comparecimento"
  no dashboard do gestor.
- **Unidades reais confirmadas dentro desses textos de reagendamento**: "CRAS - Prazeres",
  "CRAS - Curado", "CRAS - Jardim Jordão/Guararapes", além de "CRAS - Vila Rica" e
  "CRAS - Cavaleiro" (exemplos que o usuário deu diretamente) — a SASC opera várias unidades
  fisicamente distintas, não só a "CRAS Piloto" do MVP atual.
- **`Serviço` (ex: "Agendar Atendimento no Cadastro Único (CadÚnico)") não é o mesmo conceito
  de unidade/local** — é o tipo de atendimento, não onde ele acontece.
- **`Agendamento para (dependente)`**: quando preenchido, é esse nome (não o do "Cidadão
  responsável") quem deve aparecer como o nome do atendimento — o agendamento foi feito por um
  responsável em nome de um dependente.
- **`Data` vem separada de `Horário`, em português por extenso e sem ano** (ex: "15 de
  Setembro") — parsear assumindo o ano do momento do upload; pode quebrar em uploads que
  atravessem virada de ano (risco baixo pro piloto, mas documentar a limitação).
- **Não usar o pacote `xlsx`/SheetJS do npm** pra parsing de Excel (se o Go também acabar tendo
  uma etapa em Node/script auxiliar) — a versão publicada no registry tem 2 CVEs de alta
  severidade sem correção (prototype pollution + ReDoS), exploráveis via arquivo malicioso, que
  é exatamente o vetor de um upload de planilha. Preferir uma lib mantida (no ecossistema Go,
  `qax-os/excelize` é a opção mais usada).
- Há **um arquivo de export real na raiz do projeto** (`agendamentos-*.xlsx`, o nome muda a cada
  novo export recebido) com CPF/telefone de cidadãos de verdade — nunca versionar isso no git
  (já está no `.gitignore`).

## Backend implementado e testado de ponta a ponta (19/09, terceira rodada)

MVP funcional de verdade — não é mais só schema/spec. Todo o ciclo abaixo foi validado via
`curl` contra o Postgres real, usando a **planilha real** (`agendamentos-10-29_20260919_102921.xlsx`,
175 linhas de dados) como dado de teste, a pedido do usuário ("ela será nosso banco por
enquanto" — interpretado como: essa planilha vira o dado de teste/seed real, o Postgres continua
sendo o banco de verdade).

**Migration reescrita do zero** (`backend/migrations/000001_init.{up,down}.sql`) — finalmente em
dia com a skill `modelo-dados` (que tinha acumulado 3 rodadas de mudança sem a migration
acompanhar). Banco resetado e recriado limpo.

**Estrutura Go implementada**:
- `internal/domain` — structs de todas as entidades.
- `internal/repository` — acesso a banco via pgx puro (sem ORM, como a skill `stack-go-react-postgres` já mandava). Um arquivo por entidade.
- `internal/auth` — login MVP (skill `auth-login-mvp`): bcrypt pra senha, sessão própria em JWT HS256 num cookie HttpOnly, middleware que relê o usuário do banco a cada requisição (papel/ativo nunca ficam só no token).
- `internal/painel` — a fila de exibição do painel de TV, em memória, exatamente como desenhado na skill `painel-tv` (sem tabela nova).
- `internal/handlers` — um arquivo por área (auth, importação, fila/recepção/atendente, painel, dashboard).
- `internal/util` — normalização de texto (acento/caixa/espaço) e o cálculo de nome parcial, compartilhados entre repository e handlers.

**Dependências novas**: `golang-jwt/jwt/v5`, `golang.org/x/crypto/bcrypt`, `xuri/excelize/v2` (Excel — não usar `xlsx`/SheetJS, ver "Aprendizados" acima).

**Endpoints implementados e testados**:
```
POST   /api/v1/login                                              (público)
GET    /api/v1/sessao
DELETE /api/v1/sessao
GET    /api/v1/painel/{unidadeId}                                 (público, sem cookie)
POST   /api/v1/secretarias/{secretariaId}/lotes                   (só gestor)
GET    /api/v1/unidades/{unidadeId}/recepcao                      (recepcionista/gestor)
POST   /api/v1/unidades/{unidadeId}/encaixe                       (recepcionista/gestor)
POST   /api/v1/agendamentos/{id}/confirmar-chegada                (recepcionista/gestor)
GET    /api/v1/unidades/{unidadeId}/sala-espera                   (atendente/gestor)
POST   /api/v1/unidades/{unidadeId}/guiches/{guicheId}/chamar-proximo  (atendente/gestor)
POST   /api/v1/agendamentos/{id}/rechamar                         (atendente/gestor)
POST   /api/v1/agendamentos/{id}/atendido                         (atendente/gestor)
GET    /api/v1/unidades/{unidadeId}/atendimento                   (atendente/gestor, pessoal)
GET    /api/v1/agendamentos/{id}/historico                        ("janela do cidadão")
GET    /api/v1/unidades/{unidadeId}/dashboard                     (só gestor)
```

**O que foi testado de verdade, via curl, e passou**:
- Login com os 3 papéis (senha certa, senha errada → 401, sem cookie → 401).
- Importação da planilha real: **175 linhas, 0 erros, 7 canceladas identificadas corretamente**, unidade resolvida pelo nome real ("Cadastro Único Sede Prazeres" — o seed já usa esse mesmo nome de propósito, pra planilha real cair direto na fila dos usuários de teste).
- Recepção: lista, confirmar chegada (grava `prioridade` corretamente comparando contra o SLA da unidade).
- Sala de espera: ordenação correta (no-horário antes; atrasado e encaixe competindo juntos por chegada — testado criando um encaixe de verdade via `POST /encaixe` e vendo ele cair depois dos agendados no-horário).
- **Concorrência real testada**: duas requisições `chamar-proximo` simultâneas no mesmo guichê (via `curl ... & curl ... & wait`) — cada uma pegou uma pessoa diferente, nenhuma duplicata. `FOR UPDATE SKIP LOCKED` funcionando.
- Painel de TV público: sem cookie nenhum, nome parcial correto ("JOSELMA SOUZA DE LIMA" → "JOSELMA L."), fila de exibição em memória funcionando (mostra a chamada mais antiga ainda "na vez" mesmo depois do agendamento já ter mudado de status — comportamento correto do desenho de fila de exibição).
- Rechamar, marcar atendido, e bloqueio de transição inválida (marcar atendido duas vezes → 409).
- Aba "Atendimento" pessoal do atendente (filtra por quem fez a última chamada).
- **Ausência automática nos dois estágios, testada de verdade**: forcei no banco um `chamado_em` de 30min atrás (SLA atendimento = 20min) e um `horario_previsto` de 2h atrás (SLA chegada = 60min), esperei o sweeper (roda a cada 20s) rodar, e os dois viraram `ausente` sozinhos, sem nenhuma ação manual — exatamente a regra confirmada com o usuário.
- Dashboard: contadores corretos quando filtrados pela data real dos dados (a planilha é de 21/09, então com `current_date` real dá 0 — comportamento esperado, não é bug).

**Simplificações/decisões tomadas durante a implementação, não pedidas explicitamente antes**
(documentando pra não esquecer, nenhuma delas é definitiva):
- Login por email **sem escopo de unidade** — `usuarios.email` só é único dentro de uma unidade no schema, mas a tela de login não pergunta unidade. Assumido que email é de fato único na prática (poucos usuários conhecidos no piloto).
- Protocolo duplicado: a constraint do banco é `(unidade_id, protocolo)`, mas a checagem de verdade na importação é **por secretaria inteira** (consulta com join), como a skill `importacao-planilha` já pedia. Não criei uma constraint de banco nova pra isso — checagem só na aplicação.
- "Quem atendeu" pro dashboard (`atendidos por atendente`) é resolvido pela **última linha de `chamadas`** daquele agendamento — não existe um campo `atendido_por` explícito no schema (a coluna "Atendido por" da planilha de origem continua não mapeada, como já estava documentado).
- CORS liberado de forma simples/permissiva pra desenvolvimento (`internal/main.go`, função `corsSimples`) — endurecer antes de qualquer deploy real.
- Endpoint pra gestor configurar SLA/guichês de uma unidade (a tela "Gerenciar CRAS" da skill `papeis-e-telas`) **não foi implementado ainda** — os campos existem no schema e já são usados pela lógica (confirmar chegada, sweeper, painel), só falta o endpoint de escrita.

**Ainda não feito** (não é decisão em aberto, é trabalho que falta):
- Testes automatizados (Go `testing`) — tudo foi validado manualmente via `curl`/navegador nesta sessão, nenhum teste ficou no repositório.
- Gestão de atendentes e SSO real seguem fora de escopo, como já combinado.

## Frontend conectado e testado no navegador (19/09, quarta rodada)

Todas as 5 telas saíram do estado "placeholder" e foram conectadas de verdade aos
endpoints acima — testadas no Chrome de ponta a ponta, não só por tipo/build.

- `src/api.ts` reescrito: cliente fetch com `credentials: "include"` (cookie de sessão
  HttpOnly, nada em `localStorage`), tipos pra cada resposta do backend.
- **Login** (`telas/login`): email+senha de verdade contra `POST /api/v1/login`, redireciona
  pro papel certo. Testado com `gestor.teste@local.dev` / `teste123`.
- **Recepção**: lista real (165+ agendamentos da planilha importada), linha destacada quando
  o horário já chegou, botão "Confirmar chegada" funcionando (testado ao vivo — linha some da
  lista de pendentes e vira "na sala de espera"), form de encaixe.
- **Atendente**: as duas abas de verdade (Sala de Espera compartilhada com chips de
  prioridade coloridos — no_horario verde, atrasado amarelo, encaixe roxo — e Atendimento
  pessoal). "Chamar próximo" testado ao vivo (por enquanto pede o UUID do guichê colado à
  mão, já que a tela de gerenciar guichês ainda não existe — ver pendência acima).
- **Dashboard do gestor**: contadores reais + upload de planilha (`<input type="file">` →
  `POST /api/v1/secretarias/{id}/lotes`) — o resultado da importação (criados/cancelados/erros)
  aparece na tela.
- **Painel de TV**: tela de "toque pra iniciar" (libera áudio, restrição do navegador — ver
  skill `painel-tv`), depois exibição real testada ao vivo — protocolo em destaque, nome
  parcial correto, lista de últimas chamadas com os 3 ícones de status (✓ atendido, … chamado,
  ✕ ausente — vi um "✕ MARIA S." de verdade, do teste do sweeper). **Áudio/voz implementado**
  (Web Speech API + um beep antes, nome parcial + guichê, sem protocolo) mas não testado com
  som de verdade nesta sessão (o navegador automatizado não reproduz áudio de forma
  verificável) — testar ouvindo de propósito antes de considerar validado.
- CSS: reaproveitado o `index.css` da versão anterior (`.pagina`, `.cartao`, `.painel-tv` etc.)
  com adições pontuais (`.tabs`, `.chip`, `.status-icone`, `.descanso-lista`).

## Voz do painel verificada de verdade + seletor de guichê (19/09, quinta rodada)

Duas lacunas da rodada anterior foram fechadas:

- **Endpoint novo**: `GET /api/v1/unidades/{unidadeId}/guiches` (papel atendente/gestor),
  reaproveitando `ListarGuichesAtivos` que já existia no repository. Rota registrada em
  `cmd/api/main.go`.
- **Frontend**: `api.guiches()` novo em `api.ts`; `SalaDeEspera.tsx` trocou o campo de texto
  onde o atendente colava o UUID do guichê por um `<select>` de verdade, populado por esse
  endpoint. A escolha é lembrada em `localStorage` (só conveniência local do navegador —
  não é estado que precisa ser confiável entre dispositivos) pra não pedir de novo a cada
  recarregamento da tela.
- **Áudio/voz confirmado de verdade, não só "implementado"**: a rodada anterior tinha
  deixado marcado que a voz nunca tinha sido efetivamente ouvida/verificada (só existia no
  código). Nesta rodada, instrumentei `window.speechSynthesis.speak` e `window.AudioContext`
  via injeção de JS no painel de TV real (Chrome), disparei uma chamada de verdade
  (`chamar-próximo`) a partir da tela do atendente em outra aba, e confirmei nos logs do
  console: `[VOZ-TESTE] beep AudioContext criado` seguido de
  `[VOZ-TESTE] falou: CARLOS S., Guichê 1 pt-BR` — na ordem certa (beep antes da fala), com
  nome parcial e guichê corretos, sem protocolo, exatamente como desenhado na skill
  `painel-tv`. Também confirmado visualmente que o painel sai do "modo descanso" e mostra a
  chamada em destaque (protocolo grande, nome, guichê) enquanto a exibição está ativa, e volta
  pro modo descanso depois que a duração configurada (`duracao_chamada_painel_segundos`,
  default 30s) expira — comportamento correto do `internal/painel/fila_exibicao.go`.

## Tela "Gerenciar CRAS" do gestor — SLA e guichês (19/09, sexta rodada)

Fechada a última pendência de escrita que faltava (item 8 dos "Próximos passos"): o gestor
agora configura SLA e guichês pela própria UI, sem precisar mexer no banco.

**Backend, endpoints novos** (`internal/handlers/configuracao.go`, só papel gestor):
```
GET   /api/v1/unidades/{unidadeId}/configuracao        (unidade + todos os guichês, inclusive inativos)
PATCH /api/v1/unidades/{unidadeId}/configuracao        (SLAs e durações do painel)
POST  /api/v1/unidades/{unidadeId}/guiches             (criar guichê)
PATCH /api/v1/unidades/{unidadeId}/guiches/{guicheId}  (ativar/desativar — sem exclusão de
                                                         verdade, guichê com chamadas no
                                                         histórico não pode sumir)
```
Repository ganhou `AtualizarConfiguracaoUnidade`, `CriarGuiche`, `AtualizarAtivoGuiche`,
`ListarTodosGuiches` em `internal/repository/unidades.go`.

**Frontend**: nova seção "Gerenciar CRAS" em `DashboardGestor.tsx` — formulário de SLA
(tolerância de chegada/atendimento, duração de exibição normal/com fila) e tabela de guichês
com criar/ativar/desativar. `SalaDeEspera.tsx` (rodada anterior) já usava
`GET /unidades/{id}/guiches` pro seletor do atendente — mantido como está, só leitura.

**Bug real encontrado e corrigido durante o teste no navegador**: os dois PATCHs novos
davam **503 no Chrome sem nenhum log no servidor** — sintoma de bloqueio de CORS no
preflight, não erro do handler. Causa: `corsSimples` em `cmd/api/main.go` fixava
`Access-Control-Allow-Methods: GET,POST,PUT,DELETE,OPTIONS` — **faltava `PATCH`**, que nunca
tinha sido usado por nenhum endpoint anterior. Corrigido adicionando `PATCH` à lista;
reproduzido o erro e confirmado o fix no mesmo teste (rede mostrou `PATCH → 503` antes,
`PATCH → 200` depois do rebuild).

**Testado ao vivo no Chrome**: login como gestor, formulário de SLA carrega os valores reais
da unidade, criar guichê novo (nome com acento — "Guichê Teste UI" — confirmado salvo com a
codificação UTF-8 correta direto pela UI, diferente de um teste anterior via `curl` no
terminal que corrompeu o acento por causa do encoding do shell, não do backend), desativar
guichê (agora funcionando pós-fix de CORS) e reativar — tudo verificado via
`read_network_requests`, não só visualmente. Dado de teste (guichê e SLA alterados) revertido
ao estado original depois do teste.

**Não implementado ainda no frontend**: nada crítico restante nesta área — telas de
gerenciamento de atendentes seguem fora de escopo por decisão já registrada em "Decisões em
aberto".

## Painel de TV: nome completo, voz melhorada e novo layout (20/09)

Ajustes pedidos diretamente pelo dono do produto depois de ver o painel funcionando:

- **Nome completo, não mais parcial** — decisão revertida (era "nome parcial" desde 19/09,
  por privacidade). Agora `chamadaAtual.nome` e `ultimasChamadas[].nome` retornam o nome
  completo do cidadão. Mudança nos dois lados:
  - Backend: `internal/painel/fila_exibicao.go` (`ItemExibicao.NomeParcial` renomeado pra
    `.Nome`), `internal/handlers/fila.go` e `internal/handlers/painel.go` param de passar
    `a.NomeCidadao` direto em vez de `util.NomeParcial(...)`. A função `util.NomeParcial`
    continua existindo (não foi removida) caso a decisão volte atrás.
  - **Atenção**: isso есtá em tensão com a doc anterior de privacidade do painel público
    (nome completo agora aparece numa tela sem autenticação, visível a qualquer um na sala de
    espera) — decisão explícita do dono do produto, mas vale confirmar se ele está ciente
    dessa implicação antes de considerar definitivo.
- **Pronúncia do guichê corrigida**: o Web Speech estava lendo "Guichê 1" colado
  ("guicheum"). Agora `textoGuicheFalado()` em `PainelTV.tsx` separa em duas frases com
  ponto ("Guichê. Um.") e escreve o número por extenso (1 a 20) em vez do dígito — o ponto
  força uma pausa real na síntese, testado e confirmado via injeção de script no Chrome.
- **Repetição do anúncio**: cada chamada nova agora repete a cada 5s, até 3 vezes no total
  (constantes `INTERVALO_REPETICAO_MS`/`MAX_REPETICOES`). Achado um bug real durante o teste:
  o StrictMode do React (dev) causava dois `setInterval` de repetição vivos ao mesmo tempo,
  dobrando a voz. Corrigido com uma trava de "id de anúncio vigente" (`idAnuncioVigenteRef`)
  — qualquer repetição de um anúncio "superado" por um mais novo vira no-op, e o primeiro
  disparo também foi adiado um tick (`setTimeout(...,0)`) pra se proteger do mesmo cenário.
  Confirmado via teste real: exatamente 3 falas, sem duplicata, ~5s de intervalo.
- **Layout 70/30**: painel dividido em duas colunas — chamada atual ocupa 70% da largura à
  esquerda, lista "Últimas chamadas" ocupa 30% à direita (antes era tudo empilhado
  verticalmente, largura única). Dentro da chamada: **nome** é a informação principal (maior,
  no topo), **protocolo** é complemento logo abaixo, **guichê** por último — ordem confirmada
  pelo dono do produto. CSS em `index.css` (`.painel-tv-corpo`, `.chamada-atual`,
  `.historico`). O modo descanso (sem chamada ativa) não mudou — continua tela cheia com a
  lista completa, já que não há "chamada" pra dividir espaço com a lista nesse estado.
- Tudo testado ao vivo no Chrome: nome completo aparecendo correto, texto exato da fala
  capturado via injeção de script (`"RAFAEL AUGUSTO NUNES BARBOSA. Guichê. dois."`), timing
  das 3 repetições confirmado (~0s/5s/10s), e o layout 70/30 com a ordem nome→protocolo→guichê
  visto por screenshot real.

## Painel de TV: tema branco, 100vw/100vh e animações (20/09, segunda rodada)

Pedido do dono do produto depois de ver o layout anterior: "faça tudo caber na tela sem
quebrar", fundo branco minimalista, animações na tabela, e tabela mais compacta.

- **100vw/100vh sem scroll na página** — `.painel-tv` agora é `width:100vw; height:100vh;
  overflow:hidden`, com `box-sizing:border-box`. Testado com nome bem longo ("FERNANDA
  CRISTINA ALMEIDA ROCHA DOS SANTOS", quebra em 3 linhas) e confirmado via
  `document.documentElement.scrollHeight === clientHeight` — zero overflow. Só a `<ul>` da
  tabela tem `overflow-y:auto` interno como proteção, caso a lista de 10 itens não caiba
  numa tela muito baixa.
- **Tema branco minimalista** — trocado o fundo escuro (`#0f172a`) por branco, com tokens
  novos (`--painel-fundo`, `--painel-texto`, `--painel-card`, `--painel-acento` etc. no
  topo do `index.css`). Protocolo mudou de amarelo (não teria contraste em fundo claro)
  pra azul (`--painel-acento`). Ícones de status recoloridos pra fundo claro (verde/âmbar/
  vermelho com fundo pastel, mesmo padrão dos `.chip` já usados nas outras telas).
- **Animações**: a estrutura do JSX deixou de alternar entre duas árvores diferentes
  (`{!emDescanso && ...}` / `{emDescanso && ...}`) e virou uma estrutura única sempre
  montada, só trocando classe CSS (`modo-ativo`/`modo-descanso`) — isso é o que permite
  transição suave em vez de troca abrupta:
  - Chamada em destaque (`.chamada-atual`) colapsa pra `flex-basis:0` + `opacity:0` no
    descanso, e a tabela (`.historico`) cresce e centraliza (`margin:0 auto`) ocupando o
    espaço que sobrou — com `transition` de 0.6s. Quando uma chamada nova chega, o processo
    inverte: a tabela "volta" pra 30% à direita e a chamada reaparece à esquerda.
  - Cada nova linha que entra no topo da tabela tem uma animação de entrada
    (`@keyframes entrarNaTabela`, fade + leve deslocamento vertical) — dispara sozinha
    porque cada chamada tem um `id` novo (React monta um `<li>` genuinamente novo).
  - Confirmado por medição real do DOM (`getBoundingClientRect`) que a tabela fica
    matematicamente centralizada no modo descanso — a suspeita inicial de estar
    "deslocada pra direita" era só o screenshot sendo capturado numa escala de pixels
    diferente da viewport CSS real (1254 lógico vs 1568 da imagem), não um bug de verdade.
- **Tabela compacta**: cada linha agora mostra só **primeiro nome** (`primeiroNome()`) +
  **início do protocolo** com reticências (`protocoloAbreviado()`, corta em 9 caracteres) +
  guichê + o ícone de status de sempre — o nome/protocolo completos continuam só na chamada
  em destaque, a tabela é pra identificar rápido sem ocupar espaço.
- Gate de "toque pra iniciar" também migrado pro tema branco, pra não haver um flash escuro
  antes do painel de verdade aparecer.

## Painel de TV: voz final, tabela com nome/protocolo completos, fix de animação (20/09, terceira rodada)

Três ajustes finos pedidos depois de ouvir/ver o painel funcionando:

- **Entonação da voz corrigida**: o guichê estava sendo falado com tom de pergunta
  ("guichê um?"). Causa: `textoGuicheFalado()` separava com PONTO ("Guichê. Um.") — várias
  vozes pt-BR do Web Speech leem uma sentença isolada de uma palavra só com entonação de
  incerteza/pergunta. Trocado o ponto por vírgula ("Guichê, um.") — a pausa continua
  existindo, mas agora é uma cláusula dentro de uma frase só, com ponto final único no fim
  de tudo, o que lê como afirmação. Confirmado via injeção de script capturando o texto
  exato falado.
- **Tabela do painel**: voltou a mostrar nome **completo** (só o primeiro nome sozinho
  "ficava ruim", nas palavras do dono do produto) e protocolo **completo** (o início
  abreviado não ajudava a diferenciar porque todos os protocolos começam igual — ex.
  "ENC-2026091..."). CSS ajustado pra layout em grid com `minmax(0, fr)` nas duas colunas
  (nome com peso 1.3, protocolo com peso 1), cada uma com reticências (`text-overflow:
  ellipsis`) se não couber — nunca quebra a linha, só corta com "…" no limite.
- **Bug real de animação encontrado e corrigido**: durante a transição descanso→ativo, por
  uma fração de segundo o nome aparecia "espremido" numa coluna de letras soltas (uma letra
  por linha, tipo "D/O/M/A/R/T/I/N/S"). Causa: a opacidade e a largura da caixa
  (`.chamada-atual`) estavam mudando ao mesmo tempo — o texto ficava visível antes da caixa
  terminar de crescer. Corrigido com transições escalonadas (`transition-delay`): ao entrar,
  a caixa cresce primeiro e o texto só aparece depois (delay de 0.3s na opacidade); ao sair,
  o texto some primeiro (0.2s) e só depois a caixa encolhe. Efeito colateral bom: a troca
  fica mais suave visualmente também.
- Durante o teste apareceu uma falsa pista de duplicação de voz (6 falas em vez de 3) — não
  era bug de produto, era o próprio script de teste (injetado via devtools) tendo sido
  reinjetado mais de uma vez na mesma aba sem recarregar a página, o que aninhava dois
  wrappers de `speechSynthesis.speak` um dentro do outro. Um teste limpo (recarregar a
  página, instrumentar uma única vez, comparar contagem de `new SpeechSynthesisUtterance()`
  contra contagem de `.speak()`) confirmou 3=3, sem duplicação real. Anotado aqui pra não
  cair na mesma pegadinha de novo ao testar manualmente via injeção de script.

## Repetição/intervalo configuráveis, guichês vs. salas, contagem regressiva de ausência (20/09, quarta rodada)

Pedido do dono do produto: personalizar quantas vezes e de quanto em quanto tempo a voz
repete a chamada (antes fixo em 3x/5s no código), reorganizar "Gerenciar CRAS" com blocos
expansíveis por guichê/sala (tipo, andar, capacidade), separar os blocos de SLA de recepção
e de atendimento, e mostrar ao atendente uma contagem regressiva até o prazo de ausência.

**Migration nova** (`backend/migrations/000002_config_avancada.{up,down}.sql`, aplicada):
- `unidades`: `repeticoes_chamada int default 3`, `intervalo_repeticao_segundos int default 5`.
- `guiches`: `tipo text default 'guiche' check (guiche|sala)`, `andar text` (opcional),
  `capacidade int default 1`.

**Decisão de escopo importante**: `tipo`/`andar`/`capacidade` do guichê são só metadados
**descritivos** pra organizar a tela — não mudam a regra de negócio de "um chamado por vez"
do `chamar-próximo` (skill regras-negocio-fila). Uma "sala" com capacidade 4 não permite 4
chamadas simultâneas de verdade ainda; isso ficaria pra uma decisão de produto futura e
separada, se algum dia for necessário. Documentando pra não presumir esse comportamento sem
confirmar antes.

**Backend**:
- `internal/domain/domain.go`: `Unidade` ganhou `RepeticoesChamada`/`IntervaloRepeticaoSegundos`;
  novo tipo `TipoGuiche` (`guiche`/`sala`); `Guiche` ganhou `Tipo`, `Andar *string`, `Capacidade`.
- `internal/repository/unidades.go`: reescrito — `AtualizarConfiguracaoUnidade` agora grava os
  6 campos juntos; `CriarGuiche`/`AtualizarGuiche` (nova, edição completa) recebem tipo/andar/
  capacidade; `colunasUnidade`/`colunasGuiche` centralizam a lista de colunas pra não duplicar
  entre as várias queries.
- `internal/repository/agendamentos.go`: `ListarAtendimentoDoAtendente` agora retorna
  `ItemAtendimento{Agendamento, ChamadoEm}` (precisa do `chamado_em` pra calcular o prazo).
- Endpoints: `PUT /api/v1/unidades/{unidadeId}/guiches/{guicheId}` (edição completa, novo,
  só gestor) além do `PATCH` que já existia (liga/desliga rápido). `GET /painel/{unidadeId}`
  agora inclui `unidade.repeticoesChamada`/`intervaloRepeticaoSegundos`. `GET
  /unidades/{unidadeId}/atendimento` agora inclui `prazoAusenciaEm` (chamado_em + SLA de
  atendimento) em cada item.
- Validação: repetições entre 1-10, intervalo entre 1-120s.

**Frontend**:
- `PainelTV.tsx`: não usa mais constantes fixas de repetição — lê
  `resposta.unidade.repeticoesChamada`/`intervaloRepeticaoSegundos` a cada poll e passa pro
  `iniciarAnuncio`, com fallback pros valores antigos (3x/5s) só até a primeira resposta.
- `SalaDeEspera.tsx`: aba "Atendimento" ganhou `<ContagemAusencia>` — contagem regressiva
  (mm:ss) ao final de cada linha, calculada a partir de `prazoAusenciaEm`, atualizando a cada
  segundo, ficando vermelha no último minuto e mostrando "prazo esgotado" se passar (o
  sweeper do backend continua sendo quem de fato marca ausente, isso é só o atendente vendo
  o relógio correndo).
- `DashboardGestor.tsx`: seção "Gerenciar CRAS" reorganizada na ordem pedida — Guichês e
  salas (cada linha fechada com nome/tipo/status, clique abre um bloco de edição com nome,
  tipo guichê/sala, andar opcional, capacidade), SLA da recepção, SLA do atendimento,
  Configurações do painel (durações de exibição + repetições + intervalo). As três últimas
  seções salvam o mesmo registro de configuração (não é PATCH parcial de verdade no
  backend), só estão visualmente separadas por pedido do dono do produto.

**Verificado via curl (não via clique no navegador nesta rodada — ver nota abaixo)**: GET
configuração retorna os campos novos com os defaults corretos (3/5), PUT completo de guichê
grava tipo/andar/capacidade corretamente (incluindo acento em nome verificado com bytes
UTF-8 explícitos), GET atendimento retorna `prazoAusenciaEm` calculado certo (chamado_em +
20min de SLA testado). `go build`, `go vet`, `gofmt`, `tsc --noEmit` e `npm run build` todos
limpos.

**Atualização (20/09, sexta rodada)**: a tela "Gerenciar CRAS" foi confirmada visualmente
depois que o login virou acesso rápido (ver seção abaixo) — clique numa linha de guichê abre
o bloco de edição com nome/tipo/andar/capacidade corretos, e os três blocos de SLA/painel
renderizam separados como desenhado. A instabilidade de clique mencionada abaixo era mesmo
só da ferramenta de automação, não do app.

## Bug real encontrado e corrigido: login "voltava sozinho" pra tela de login (20/09, quinta rodada)

O usuário reportou: "quando eu clico em entrar, ele não acessa, ele só recarrega a página e
continua no login". Investigado e confirmado como bug real (não só instabilidade de
automação) — causa raiz: **cookie de sessão cross-site bloqueado pelo navegador**.

**O que causava**: `frontend/.env` tinha `VITE_API_URL` fixo no IP da rede
(`http://192.168.15.86:3333`), decisão da rodada anterior pra permitir acesso via Remote
Control. Só que se o frontend fosse acessado por um **hostname diferente** do IP fixo (ex.:
`localhost:5173`, que é como esta sessão estava testando), a chamada de login ia pro IP mas
a PÁGINA estava em `localhost` — hosts diferentes = o navegador trata como **cross-site**.
Um cookie `SameSite=Lax` (o que a sessão usa) não é enviado/aceito de volta em contexto
cross-site. Resultado exato reproduzido e confirmado via `fetch` direto no console do
navegador:
```
POST /login (cross-site)  → 200 OK, mas o cookie não gruda
GET  /sessao (logo depois) → 401 "sessao ausente"
```
Isso faz o app: login "funciona" (recebe os dados do usuário) → navega pra `/gestor` →
`/gestor` confere sessão → 401 → `navigate("/login")` de volta. Do ponto de vista de quem
usa, parece só "recarregar e continuar no login".

**Correção**: `frontend/src/api.ts` não usa mais um host fixo — deriva a URL da API a partir
do MESMO host que serviu a página (`window.location.hostname`), sempre na porta 3333:
```ts
const API_URL = import.meta.env.VITE_API_URL || `${window.location.protocol}//${window.location.hostname}:3333`;
```
`VITE_API_URL` no `.env` foi esvaziado de propósito (comentário explica por quê) — continua
existindo como override manual pro dia em que front/back precisarem ficar em hosts
realmente diferentes, mas o padrão agora é "sempre o mesmo host da página", o que funciona
automaticamente por `localhost`, `127.0.0.1` ou o IP da rede, sem precisar editar `.env`
toda vez que o IP mudar (resolve de vez essa fragilidade, não só o sintoma).

**Confirmado via teste direto** (não por clique nesta descoberta — a automação de clique no
Chrome estava instável no momento, ver adiante que se resolveu sozinha): reproduzi o bug de
propósito (login cross-site → 200 seguido de 401) pra provar a causa, depois testei o
cenário corrigido em dois hosts diferentes (`localhost:5173` e `192.168.15.86:5173`, ambos
apontando pra si mesmos na porta 3333) — os dois deram login 200 seguido de `/sessao` 200 com
os dados corretos do usuário. Frontend reiniciado pra aplicar o `.env` novo (Vite só lê
`.env` na inicialização do servidor).

## Login por email/senha desligado — acesso rápido por papel pra teste (20/09, sexta rodada)

Pedido do dono do produto: "esse login não vai ser usado" (o login real será via SSO da
plataforma, pós-MVP — ver skill `auth-conecta-cidades`) — então trocar a tela de
email/senha por 3 botões (Gestor / Recepção / Atendente) que entram direto, sem digitar
nada, pra agilizar teste manual.

`frontend/src/telas/login/Login.tsx` reescrito: em vez de formulário, mostra 3 botões que
chamam `api.login()` com as credenciais fixas dos usuários do seed (`gestor.teste@local.dev`,
`recepcao.teste@local.dev`, `atendente.teste@local.dev`, senha `teste123` pros três) e
navegam pra tela certa. **O mecanismo de autenticação em si não mudou** — continua criando
sessão de verdade via cookie HttpOnly no backend; só a etapa de digitar email/senha foi
removida da UI. Fácil de reverter pro formulário se um dia precisar (o backend nunca soube
da diferença).

Testado ao vivo no Chrome (clique funcionou normalmente desta vez — a instabilidade das
rodadas anteriores era mesmo intermitente/da ferramenta): entrar como Gestor levou direto
pro dashboard com dados reais, e a partir daí também foi possível confirmar visualmente a
tela "Gerenciar CRAS" da rodada anterior (ver nota acima).

## Posse de chamada, cooldown de rechamada e chamados órfãos (20/09, sétima rodada)

Reformulação grande da tela do atendente e da regra de chamada, pedida pelo dono do
produto depois de pensar no fluxo físico de verdade: hoje a fila de chamados é
**compartilhada e com posse** — todo mundo vê quem está sendo chamado por quem, mas só o
dono pode agir, e existe uma regra de prioridade pra não deixar gente esquecida enquanto
atendentes ociosos só chamam gente nova. Ver skill `regras-negocio-fila` (a atualizar) pro
detalhe fino; aqui vai o resumo de como ficou:

**As regras, na prática:**
1. Ao chamar alguém, o atendente **não pode chamar mais ninguém** até rechamar essa pessoa
   pelo menos uma vez.
2. Rechamar só é permitido depois de um **tempo mínimo configurável** (`cooldown_rechamada_minutos`,
   default 3min) desde a última chamada — enquanto isso, o botão mostra "Aguarde Xm Ys".
3. Depois de rechamar, se a pessoa ainda não apareceu, o atendente já pode chamar outra —
   a pessoa anterior não desaparece, vira um **chamado órfão**.
4. **Enquanto existir qualquer órfão na unidade, nenhum atendente ocioso pode puxar gente
   nova da sala de espera** — tem que assumir um órfão primeiro (ou esperar o sweeper marcar
   ausência automática).
5. Só quem chamou (ou quem assumiu) pode **marcar presente** ou **rechamar** — visível pra
   todo mundo, acionável só pelo dono.
6. O prazo de ausência automática (SLA de atendimento, ex. 20min) é um **orçamento fixo
   desde a primeira chamada** — rechamar ou assumir não dá tempo extra, só usa o que já
   tinha. Isso mudou o `SweepAusenciaAutomatica`, que antes recontava os 20min a cada
   rechamada (usava `max(chamado_em)`) e agora usa `min(chamado_em)`.

**Schema novo**: migration `000003_regras_chamada` — `unidades.cooldown_rechamada_minutos`
(default 3), configurável na tela "Gerenciar CRAS" (bloco "SLA do atendimento").

**Backend, endpoints novos/alterados**:
```
GET  /api/v1/unidades/{unidadeId}/chamados         (novo — lista compartilhada "chamando agora")
GET  /api/v1/unidades/{unidadeId}/atendidos-hoje   (novo — consulta pessoal, substitui a antiga aba "Atendimento")
POST /api/v1/agendamentos/{id}/assumir             (novo — atendente ocioso assume um órfão)
POST /api/v1/unidades/{unidadeId}/guiches/{guicheId}/chamar-proximo  (agora pode dar 409
     "chamada_pendente" ou "existem_chamados_pendentes")
POST /api/v1/agendamentos/{id}/rechamar            (agora exige ser o dono — 403 "nao_e_dono"
     — e respeita o cooldown — 409 "cooldown_ativo")
POST /api/v1/agendamentos/{id}/atendido            (agora exige ser o dono — 403 "nao_e_dono")
```
`GET /unidades/{id}/atendimento` foi removido (a tela não usa mais abas, virou um grid único).

**Detecção de "órfão" via SQL** (`internal/repository/agendamentos.go`): um agendamento
`status='chamado'` é órfão quando (a) o dono já fez ≥2 chamadas nele (já rechamou) E (b)
esse mesmo dono já tem uma chamada MAIS RECENTE em outro agendamento (ou seja, seguiu em
frente). Reaproveitado em três lugares via os fragmentos `juncoesUltimaChamada` e
`condicaoOrfao` pra não duplicar a lógica.

**Frontend**: `SalaDeEspera.tsx` reescrito — sem abas, grid de 3 blocos: "Chamando agora"
(compartilhado, com posse) em cima à esquerda, "Sala de espera" embaixo à esquerda,
"Atendidos por mim hoje" (só consulta) à direita. O botão "Chamar próximo" fica desabilitado
com uma mensagem explicativa quando bloqueado por qualquer uma das duas regras.

**Bug real encontrado e corrigido durante o teste**: a query nova de `chamados` dava erro
`column reference "id" is ambiguous` — a lista de colunas compartilhada `colunasAgendamento`
não tem os nomes prefixados, e ao juntar `guiches`/`usuarios` (que também têm `id` e
`unidade_id`) o Postgres não sabia de qual tabela vinha cada coluna. Corrigido com uma
segunda constante `colunasAgendamentoComAlias` (prefixada com `a.`), usada só nessa query —
as outras continuam com a versão sem prefixo porque são usadas também em `UPDATE ...
RETURNING`, que não aceita alias.

**Testado de ponta a ponta via curl** (dois atendentes simulados — `atendente.teste` e
`gestor.teste` fazendo o papel do segundo atendente): chamar → bloqueado de chamar outro →
rechamar bloqueado por cooldown → backdate manual no banco pra simular os 3min → rechamar
funciona → chamar outro agora libera → órfão aparece corretamente pro segundo atendente →
segundo atendente bloqueado de pegar gente nova da sala de espera → assume o órfão →
guichê/dono trocam corretamente na lista compartilhada → dono antigo barrado de marcar
presente (403) → dono novo consegue. Todos os 6 comportamentos bateram exatamente com o
desenhado. Testado também ao vivo no navegador (chamar, ver o card com contagem e botão
"Aguarde Xm Ys", marcar presente, ver sumir do bloco e aparecer em "Atendidos hoje").

## Oitava rodada (20/09): voz definitiva, guichê obrigatório, cabeçalho, ícones

Pedido do dono do produto depois de usar o sistema de verdade: a voz do painel AINDA soava
como pergunta (mesmo depois do fix de vírgula da rodada anterior), sentiu delay entre chamar
e o painel reagir, quis tornar a escolha de guichê obrigatória ao entrar, e pediu uma
repaginada geral de UI/UX nas 3 telas (ícones, cabeçalho melhor).

**Voz — causa raiz de verdade, desta vez**: não era pontuação. O problema é que a frase
inteira terminava num numeral solto ("...um.") — várias vozes pt-BR leem isso com entonação
de pergunta *independente* de vírgula ou ponto, porque o numeral isolado no fim é que
dispara o efeito, não a pontuação ao redor dele. Corrigido de vez trocando a estrutura da
fala inteira: agora é `"{NOME}. {Guichê} número {extenso}. Por favor."` — o número nunca é
mais a última palavra (some no meio da frase), e "Por favor." fecha tudo com uma entonação
afirmativa/calma que praticamente nenhuma voz lê como pergunta. `internal` não mudou nada
aqui, é 100% frontend (`PainelTV.tsx`, funções `textoGuicheFalado`/`falarChamada`).

**Delay entre chamar e o painel reagir**: o polling do painel (e também da tela do
atendente/recepção) caiu de 4s pra 2s. Medido à parte: o endpoint `chamar-proximo` em si
responde em ~80ms — o delay percebido era mesmo o intervalo de poll, não lentidão do
servidor.

**Seleção de guichê obrigatória**: `SalaDeEspera.tsx` ganhou um gate — ao entrar, o
atendente **tem que** escolher o guichê/sala numa tela própria ("Qual guichê você vai
atender?") antes de ver a fila. A escolha anterior (se houver, salva no `localStorage`) vem
pré-selecionada por conveniência, mas precisa confirmar de novo — não pula sozinho. Depois
de confirmado, o guichê aparece fixo no cabeçalho com um botão "Trocar" pra reabrir o gate
se precisar mudar no meio do turno.

**Cabeçalho novo, compartilhado entre recepção/atendente/gestor**: componente
`src/componentes/Cabecalho.tsx` — esquerda quem está logado (nome + papel), **meio o nome
da unidade** (pedido explícito), direita as ações da tela (seletor de guichê pro atendente,
sempre "Sair"). Precisou de um campo novo no backend: `usuarioParaJSON` (em
`handlers/auth.go`) agora também devolve `unidadeNome` — já resolvia a unidade pra pegar
`secretariaId`, só faltava aproveitar o nome dela também.

**Ícones em vez de texto puro**: instalado `lucide-react` (biblioteca de ícones SVG, leve,
sem custo de licença). Botões relevantes ganharam ícone + texto: Chamar próximo
(`Megaphone`), Rechamar (`Repeat`), Assumir (`Volume2`), Marcar presente (`CheckCircle2`,
com destaque verde novo — `--cor-sucesso`), Confirmar chegada (`UserCheck`), Cadastrar
encaixe (`UserPlus`), Sair (`LogOut`), Adicionar/Salvar/Desativar/Reativar/Enviar na tela do
gestor, e a engrenagem da linha de guichê virou um ícone de verdade (`Settings`) em vez do
caractere Unicode solto de antes.

**Testado ao vivo no Chrome**: gate de guichê obrigatório funcionando (não deixa passar sem
escolher), cabeçalho mostrando "Atendente Teste / ATENDENTE" à esquerda, "Cadastro Único
Sede Prazeres" ao centro, guichê + Trocar + Sair à direita — confirmado nas telas de
recepção e atendente (gestor usa o mesmo componente, não testado por clique nesta rodada
mas é o mesmo código).

**Voz ajustada de novo, na sequência, depois de ouvir ao vivo**: a versão "Guichê número
um. Por favor." tinha fala demais pro gosto do dono do produto — pediu de volta o formato
simples de antes, só com dois-pontos: `"{NOME}. {Guichê}: {número por extenso}"` (ex.:
"TESTE VOZ SIMPLES. Guichê: um"). Confirmado via injeção de script que é exatamente esse o
texto falado agora, 3 repetições, sem duplicata.

**Seleção de guichê virou modal, não mais tela separada**: pedido de ajuste rápido — em vez
de uma tela cheia perguntando o guichê, agora é um modal por cima da tela do atendente
(que já fica montada e visível, escurecida atrás — `rgba(15,23,42,0.75)`), com um quadrado
clicável por guichê/sala (`aspect-ratio:1/1`, grid `auto-fill`). Clicar no quadrado já
escolhe E confirma de uma vez (sem botão "Confirmar" separado). Testado ao vivo: modal
aparece corretamente sobre o conteúdo desfocado atrás, clicar num quadrado fecha o modal e
mostra a tela principal já com o guichê certo no cabeçalho.

## Nona rodada (21/09): painel de TV no login, e mensagem de confirmação fixa na tela

**Painel de TV como acesso rápido no login**: `Login.tsx` ganhou um 4º botão ("Painel de TV
(público)", ícone `Tv`) que navega direto pra `/painel/{unidadeId}` da unidade piloto — só
conveniência de teste (o painel já era público, sem cookie, desde sempre), não muda
autenticação em nada.

**Bug relatado**: "não está dando pra clicar no em marcar como presente. ajuste lá na tela de
recepção" — investigação em duas etapas. Primeiro testei clicar direto em "Confirmar chegada"
(o botão real da Recepção — não existe "marcar como presente" nessa tela, isso é da tela do
Atendente) numa linha do topo da lista e funcionou de primeira, sem erro. Perguntei ao usuário
via `AskUserQuestion` pra confirmar qual botão/tela era o problema de fato, e ele confirmou:
é mesmo "Confirmar chegada" na Recepção.

**Causa raiz encontrada**: a lista da Recepção tem 135+ pessoas no dia. A mensagem de
sucesso/erro (`.mensagem`, classe compartilhada com Sala de Espera e Dashboard do Gestor)
era renderizada no fluxo normal da página, perto do topo — clicando num botão 30+ linhas
abaixo, a confirmação aparecia fora da área visível da tela. O clique funcionava (a linha
sumia da lista de pendentes normalmente), mas sem nenhum feedback visível na posição em que
o usuário estava olhando, parecia que "não tinha feito nada".

**Correção**: `.mensagem` (em `index.css`) virou `position: fixed` centralizada no topo da
viewport (com sombra, como um toast), em vez de inline no fluxo do documento. Afeta as 3
telas que usam essa classe (Recepção, Sala de Espera, Dashboard do Gestor) — todas passam a
mostrar a confirmação sempre visível, não importa o scroll. **Testado ao vivo**: rolei a
lista de recepção até a linha 30 ("GRACIETE CAVALCANTI CUNHA FILHA"), cliquei em "Confirmar
chegada" com essa linha em destaque no meio da tela, e a mensagem "Chegada confirmada."
apareceu flutuando bem onde o cursor estava, sem precisar rolar — comportamento confirmado
corrigido.

## Décima rodada (21/09): recepção e atendente reorganizadas a partir de wireframes do dono do produto

O dono do produto subiu dois desenhos próprios (`layout-recepção.png` e `layout atendente.png`,
na raiz do projeto) pedindo pra reorganizar as duas telas nesse formato mais compacto,
baseado em tabela. Estrutura replicada fielmente:

**Recepção** (`Recepcao.tsx`): a seção separada "Cadastrar encaixe" virou um modal, aberto por
um botão "Adicionar" ao lado de uma barra de busca nova (nome/protocolo/CPF, normalizando
pontuação pra "12345678900" encontrar "123.456.789-00"). A tabela caiu de 5 colunas pra 3:
horário, uma célula única com nome + protocolo + serviço + CPF/telefone + status (tudo
empilhado, como o desenho pedia — "dados principais do agendamento do cidadão" numa célula
só), e uma coluna de ação **só com ícone** (`UserCheck` clicável se ainda não confirmou,
`CheckCircle2` estático e verde se já está na sala de espera — o desenho já mostrava um ícone
em toda linha, não só nas pendentes).

**Atendente** (`SalaDeEspera.tsx`): os três blocos (Pendentes/Chamando agora, Sala de espera,
Atendidos por mim) continuam no mesmo grid de 3 áreas de antes, mas o bloco "Chamando agora"
deixou de ser uma lista de cartões e virou tabela (Guichê | Cidadão | Prazo | ícones de
rechamar/marcar presente — cooldown agora aparece como legenda pequena embaixo do ícone
desabilitado, tipo "2:34", já que não há mais texto no botão pra caber "Aguarde..."). O botão
"Chamar próximo" saiu de cima da seção e foi embutido no cabeçalho da própria coluna de ação
da tabela "Sala de espera" — exatamente como o desenho anotava ("BOTÃO CHAMAR PRÓXIMO" na
linha do cabeçalho). "Atendidos por mim hoje" também virou tabela (Nome | Protocolo) em vez
de lista.

CSS novo em `index.css`: `.barra-busca`/`.campo-busca` (busca da recepção), `.botao-icone` /
`.botao-icone-sucesso` / `.botao-icone-estatico` (ação em ícone único, reutilizado nas duas
telas), `.botao-icone-com-legenda` / `.legenda-botao-icone` (cooldown do rechamar),
`.modal-cartao` (modal genérico de conteúdo livre, reaproveitando o fundo escurecido que já
existia pro modal de guichê), `.coluna-acao` (célula de tabela reservada pra ação, alinhada à
direita). As regras antigas `.chamado-item`/`.chamado-info`/`.botao-sucesso` ficaram órfãs
com a troca de cards por linhas de tabela e foram removidas.

**Testado ao vivo no Chrome**: busca por nome filtrando a lista em tempo real, modal de
encaixe abrindo/fechando, confirmar chegada por ícone (linha muda pra "na sala de espera",
ícone vira check estático), fluxo completo do atendente — marcar presente por ícone (some de
Pendentes, entra em Atendidos), "Chamar próximo" habilitando de volta e chamando a próxima
pessoa, cooldown aparecendo como "2:34" embaixo do ícone de rechamar desabilitado.

## Décima primeira rodada (21/09): colunas explícitas na recepção + abas + "puxar de volta"

Ajuste fino pedido em cima da reorganização da rodada anterior: a célula única de "dados do
cidadão" virou colunas de verdade — **Horário | Cidadão | Protocolo | Status** — com o nome
do serviço (só ele, sem CPF/telefone) como legenda pequena embaixo do nome, do mesmo jeito
que já tinha sido feito antes. CPF/telefone saíram da lista visível (não foram pedidos nas
colunas nem na legenda) — continuam gravados no banco e disponíveis via busca, só não
aparecem mais na tabela.

**Abas novas**: "Recepção" (quem ainda está aguardando chegada) e "Sala de espera" (quem já
confirmou), cada uma com contador. Pedido novo do dono do produto: na aba "Sala de espera",
dá pra **puxar alguém de volta pra recepção** (desfazer uma "confirmar chegada" feita por
engano) — um ícone (`Undo2`) que reverte o agendamento pro estado `aguardando_chegada`,
limpando `chegada_em`/`prioridade`.

**Backend novo**: `POST /api/v1/agendamentos/{id}/desfazer-chegada` (papel
recepcionista/gestor) — `Repo.DesfazerChegada` só reverte enquanto o status ainda é
`sala_espera` (se já foi chamado, não reverte mais — nesse ponto já é responsabilidade do
atendente/sweeper). Bloqueado pra `tipo == "encaixe"` (409 `tipo_invalido`): encaixe nunca
teve etapa de "aguardando chegada" pra voltar, foi criado direto em sala de espera.

**Testado ao vivo no Chrome**: confirmar chegada de alguém → aparece na aba "Sala de espera"
com o ícone de puxar de volta → clicar reverte de fato, a pessoa volta pra aba "Recepção" e o
contador da sala de espera zera — mensagem de confirmação "Cidadão puxado de volta pra
recepção." apareceu no toast fixo.

## Décima segunda rodada (21/09): aba "Faltaram" na recepção

Pedido do dono do produto: uma terceira aba na recepção pra quem já foi marcado `ausente`
(automático, pelo sweeper) — só informativa, "pra saber, caso alguém chegue depois já sabe
que aquele tá atrasado". Sem ação nenhuma nessa aba (não reabre o agendamento) — é só visão.

**Backend**: `Repo.ListarParaRecepcao` passou a trazer também status `ausente`, mas **só do
dia corrente** (`coalesce(horario_previsto::date, criado_em::date) = current_date`, mesmo
padrão já usado em `dashboard.go`) — sem esse filtro a aba acumularia todo o histórico de
faltas do piloto inteiro, não só hoje, o que não seria útil pro caso de uso descrito. As
outras duas condições (`aguardando_chegada`/`sala_espera`) continuam sem filtro de data, como
já estava (na prática sempre é "hoje" de qualquer forma, dada a curta janela de SLA de
chegada). Campo `ausenteEm` (já existia no backend) passou a ser exposto no tipo
`Agendamento` do frontend.

**Frontend**: terceira aba "Faltaram (N)" em `Recepcao.tsx`, ordenada por `ausenteEm` mais
recente primeiro (quem faltou por último é o caso mais provável de aparecer batendo na
porta). Chip vermelho novo (`.chip.ausente`, "faltou") na coluna Status. Sem coluna de ação —
confirmado o guard pra não reaproveitar por engano o ícone de "puxar de volta" (que é só da
aba Sala de espera) nessas linhas.

**Testado ao vivo**: forcei um agendamento pra `ausente` direto no banco (simulando o
sweeper), a aba "Faltaram" mostrou a contagem e a linha com o chip vermelho "FALTOU" no topo
da lista (mais recente primeiro) — revertido ao estado original (`aguardando_chegada`) depois
do teste, pra não deixar dado de teste artificial na base.

## Décima terceira rodada (21/09): sistema de notificações reorganizado

Reclamação direta do dono do produto: as mensagens de feedback ("Chegada confirmada.",
"Rechamado.", etc.) "descem e ficam presas" na tela. Causa raiz: cada uma das 3 telas
(Recepção, Atendente, Gestor) tinha reinventado seu próprio estado `mensagem`/`setMensagem`,
renderizado como um `<p className="mensagem">` fixo no topo — sem timer de auto-dismiss (só
sumia quando outra ação disparava uma mensagem nova, por isso "ficava presa") e **sempre
verde**, inclusive pra mensagem de erro (bug real: um erro tipo "Erro ao salvar." aparecia
com fundo de sucesso, confuso).

**Reorganizado numa funcionalidade só**: `frontend/src/componentes/Notificacoes.tsx` — um
`NotificacoesProvider` (montado uma vez em `main.tsx`, em volta de `<App />`) e um hook
`useNotificar()` que qualquer tela chama como `notificar(texto)` (sucesso, verde) ou
`notificar(texto, "erro")` (vermelho). Empilhadas no **canto inferior direito** (antes
disputava espaço com o cabeçalho/abas no topo), cada notificação:
- some sozinha depois de 4,5s (`DURACAO_MS`);
- pode ser fechada na hora com um X;
- tem ícone e cor por tipo (`CheckCircle2` verde / `AlertCircle` vermelho);
- anima entrada (fade + leve deslize).

As 3 telas (`Recepcao.tsx`, `SalaDeEspera.tsx`, `DashboardGestor.tsx`) tiveram seus estados
locais de mensagem removidos e passaram a usar `useNotificar()`. O `.mensagem` antigo em
`index.css` foi substituído pelas classes `.notificacoes-area`/`.notificacao*`. O `<p
className="erro">` inline do upload de planilha (Gestor) também virou notificação — só o
resultado persistente da importação (contadores + tabela de erros) continua fixo na página,
por ser conteúdo de verdade, não um aviso passageiro.

**Testado ao vivo no Chrome**: confirmar chegada mostrou o toast verde "Chegada confirmada."
no canto inferior direito, sem sobrepor cabeçalho/abas; forcei um erro de verdade (clique
duplo rápido no mesmo "puxar de volta", a segunda chamada bateu num agendamento que já não
estava mais em sala de espera) e as duas notificações empilharam corretamente — verde em
cima, vermelha embaixo, cada uma com seu ícone e X.

## Décima quarta rodada (21/09): identidade visual reformulada de ponta a ponta

Pedido explícito do dono do produto: agir como especialista em frontend/design/UI-UX e
remodelar visualmente o sistema, que "está com cara de protótipo ainda, vazio, teste" — e
documentar as decisões numa skill nova pra não se perder de novo (`identidade-visual`, ver
`.claude/skills/`). Ele também reforçou que o cabeçalho devia seguir **exatamente** os
desenhos anexados na raiz (`layout-recepção.png`/`layout atendente.png`), que já existiam
desde a rodada de reorganização de telas mas cujo detalhe do cabeçalho não tinha sido
replicado à risca da primeira vez.

**Tipografia**: `Public Sans` (fonte do design system do governo americano — combina
tematicamente com "sistema de prefeitura" sem cair no clichê Inter/Space Grotesk) +
`IBM Plex Mono` pra números (protocolo, horário, contadores, estatísticas), carregadas via
Google Fonts em `index.html`. Nova classe utilitária `.numerico`.

**Paleta**: reescrita do zero em `frontend/src/index.css` — fundo off-white quente (não mais
cinza-azulado genérico), neutros com leve matiz teal em vez de cinza puro, uma cor de marca
própria (`--cor-marca`, teal-marinho escuro) reservada só pro cabeçalho, separada da cor de
ação (`--cor-primaria`, teal-azulado — não mais o azul-Bootstrap `#1d4ed8` de antes). Cada
cor semântica (sucesso/erro/aviso/info/encaixe) ganhou um par texto+fundo nomeado, usado
consistentemente em chip/notificação/aviso. Detalhe corrigido: `.aviso` (caixa informativa
neutra) usava o mesmo amarelo do chip "atrasado" — virou azul-teal (info), amarelo ficou
reservado só pra semântica de atraso/atenção.

**Cabeçalho (`componentes/Cabecalho.tsx`) reescrito pra bater com o desenho de verdade desta
vez**: fundo **sólido** (nunca transparente, pedido explícito — gradiente teal-marinho),
3 blocos fixos — esquerda "Painel de Chamada" + "Prefeitura de Jaboatão dos Guararapes"
(marca fixa do componente, não é prop, já que o MVP é uma prefeitura só), meio nome da
unidade + "{PAPEL}: {nome}" embaixo (ex. "RECEPÇÃO: MARIA SILVA" — isso mudou: antes o nome
do usuário ficava desconectado à esquerda), direita guichê/ações/sair. Esse é o erro que a
rodada anterior cometeu (colocou nome do usuário à esquerda em vez do bloco de marca) —
corrigido agora com a skill documentando a estrutura pra não repetir.

**Tabelas com identidade própria**: cabeçalho tintado, zebra sutil, hover de linha, células
numéricas em mono. Botões de ação em tabela continuam ícone puro (decisão da rodada
anterior), mantida.

**Timer de cooldown do atendente corrigido** (reclamação direta: "tá feio ali abaixo do
botão"): era um ícone desabilitado com uma legenda minúscula empilhada embaixo, desalinhado.
Virou um único botão-pílula (`.botao-cooldown`) com relógio + contagem lado a lado, mesma
altura dos outros ícones da linha — documentado como padrão geral na skill pra qualquer botão
futuro que precise mostrar informação extra.

**Estatísticas do dashboard do gestor**: números "Hoje" (agendados/atendidos/ausentes)
tinham cor inline hardcoded (`#166534`/`#b91c1c`) desalinhada da paleta nova — viraram
`.estatistica`/`.estatistica-valor` com os tokens semânticos certos.

**Testado ao vivo no Chrome**: login, recepção (cabeçalho + tabela + abas), atendente
(cabeçalho + modal de guichê + tabela "Pendentes" com o cooldown-pílula funcionando de
verdade, mostrado com "🕐 1:00" ao lado do ícone de presença) e dashboard do gestor
(estatísticas + tabelas) — todos consistentes com a paleta/tipografia nova. `tsc --noEmit` e
`npm run build` limpos.

**Fora de escopo por enquanto**: o dono do produto avisou que "em breve" vai pedir uma tela
de admin master — não implementada nem desenhada ainda, só sinalizada aqui pra lembrar de
seguir a skill `identidade-visual` quando ela vier (e conversar antes se precisar de
cor/fonte fora dessa paleta).

## Décima quinta rodada (22/09): ajustes finos de UI/UX pedidos depois de usar o sistema novo

Lista de ajustes pontuais em cima da reforma visual da rodada anterior, todos testados ao
vivo no Chrome:

**Hover muito forte**: `tbody tr:hover` (recepção/sala de espera/faltaram) e outros hovers de
"local selecionável" (`.quadro-guiche`, `.linha-guiche-resumo`) usavam `--cor-info-fundo`, a
mesma cor do cabeçalho de tabela — forte demais só pra indicar posição do mouse. Token novo
`--cor-hover` (~4% de opacidade, quase imperceptível, só "você está aqui") substituindo nos
três lugares. Confirmado via `getComputedStyle` ao vivo: `rgba(14, 46, 56, 0.043)`.

**Cabeçalho não era 100vw de verdade**: a primeira versão só sangrava até a borda de
`.pagina` (que tem `max-width`), não até a borda real do navegador. Corrigido renderizando
`<Cabecalho>` como **irmão** de `.pagina` em vez de filho, nas 3 telas — agora ocupa a
largura real da viewport. Documentado na skill `identidade-visual` como o padrão a seguir em
telas novas.

**Bug real de layout, corrigido**: trocar entre uma aba/tela com lista longa (precisa rolar)
e uma curta (não precisa) fazia o conteúdo "pular" uns pixels pro lado — sintoma clássico da
barra de rolagem aparecendo/sumindo e redistribuindo o espaço do container centralizado.
Primeira tentativa (`scrollbar-gutter: stable` sozinho no `html`) não se mostrou confiável ao
testar de verdade (o gutter colapsava de volta quando o conteúdo cabia sem rolar). Fix final:
`overflow-y: scroll` (técnica clássica, garantida em qualquer navegador) **junto com**
`scrollbar-gutter: stable`. Confirmado via `document.documentElement.clientWidth` ao vivo:
ficou estável em 1521px alternando entre uma aba de 3 linhas e uma de 129, antes variava
entre 1536 (sem barra) e 1521 (com barra).

**Telas do atendente reorganizadas** (pedido específico, não vale pra recepção): os 3 blocos
("Pendentes", "Sala de espera", "Meus atendimentos hoje") perderam o `.cartao` branco em
volta — título solto direto sobre o fundo da página, tabela logo abaixo, sem moldura. O botão
"Chamar próximo" saiu de dentro do cabeçalho da tabela e foi pra uma `.secao-cabecalho` ao
lado do título "Sala de espera", como uma ação de seção de verdade.

**Nova aba "Ausentes" no bloco de atendimentos pessoais do atendente**: pedido — "pra ver
essas pessoas que estavam na fila de espera e não apareceram". Contraparte exata de
"Atendidos por mim hoje", mas filtrando `status = 'ausente'` em vez de `'atendido'`. Backend
novo: `Repo.ListarAusentesHoje` (mesmo padrão de `ListarAtendidosHoje` — só quem teve
**alguma** chamada de verdade deste atendente conta; quem nunca chegou a ser chamado não
pertence a ninguém) e `GET /api/v1/unidades/{unidadeId}/ausentes-hoje`. Vira uma aba
(`Atendidos (N)` / `Ausentes (N)`) dentro do mesmo bloco, reaproveitando o padrão de abas já
usado na recepção.

**Guichê + "Trocar" fundidos**: eram dois elementos lado a lado no cabeçalho do atendente
(indicador "Guichê 2" só-visual + um botão "Trocar" separado) — viraram um botão só: clicar
no próprio indicador de guichê reabre o modal de seleção.

Tudo documentado na skill `identidade-visual` (seções de hover, cabeçalho 100vw, scroll
estável e "blocos sem cartão") pra não regredir nem repetir a tentativa que não funcionou
(`scrollbar-gutter` sozinho). `tsc --noEmit`, `go build` e `npm run build` limpos.

## Décima sexta rodada (22/09): tabela branca, busca no bloco pessoal, Ausentes virou compartilhado

Três ajustes em cima da rodada anterior, todos testados ao vivo:

**Tabela do atendente sem fundo nenhum**: ao tirar o `.cartao` dos 3 blocos, a tabela ficou
"flutuando" sem nenhuma superfície atrás — só o cabeçalho tintado. Dono do produto confirmou
que a tabela em si pode ser branca, só não queria a moldura em volta do título. Fix: classe
nova `.tabela-cartao` (fundo `--cor-superficie`, borda, raio, sombra leve) envolvendo só a
`<table>`, título continua solto fora dela.

**Busca em "Meus atendimentos hoje"**: reaproveitado `.barra-busca`/`.campo-busca` (mesmo
padrão da recepção), filtrando por nome ou protocolo. Testado ao vivo (via injeção de valor
+ evento `input`, já que a digitação por clique da automação estava instável no momento —
não é bug do app, confirmado lendo o DOM depois: filtrou "valeria" pra só
"VALERIA DE LIMA GUIMARAES" corretamente).

**Aba "Ausentes" movida de "Meus atendimentos hoje" pra "Sala de espera"**: pedido do dono do
produto — "já que isso vai ser uma tela compartilhada". Mudança de escopo, não só de lugar:
antes era pessoal (só quem EU chamei e não apareceu — `ListarAusentesHoje(unidadeID,
usuarioID)`), agora é da unidade inteira (`ListarAusentesHoje(unidadeID)`, sem filtro de
usuário, mantendo a condição de só contar quem teve alguma chamada de verdade). Mesmo
endpoint (`GET /unidades/{unidadeId}/ausentes-hoje`), assinatura da função no repository
simplificada. "Sala de espera" ganhou abas "Aguardando"/"Ausentes"; o botão "Chamar próximo"
some quando a aba "Ausentes" está selecionada (não faz sentido chamar o próximo olhando pra
lista de quem já faltou). Testado ao vivo: contagem foi de 5 (pessoal) pra 6 (unidade
inteira, outros atendentes também têm ausentes hoje).

Skill `identidade-visual` atualizada com a distinção "bloco compartilhado vs. bloco pessoal"
(pra não repetir o erro de colocar informação da unidade inteira dentro de um bloco filtrado
por usuário) e a regra "tabela sempre com `.tabela-cartao`, mesmo em bloco sem `.cartao`
em volta do título". `tsc --noEmit`, `go build` e `npm run build` limpos.

## Décima sétima rodada (22/09): entidade Grade de horário + tela do gestor reconstruída

A maior mudança estrutural do projeto até agora — motivada por uma pergunta de
esclarecimento que revelou que "unidade" não bastava mais: o dono do produto confirmou que
**uma unidade (local físico) pode oferecer vários serviços ao mesmo tempo, cada um com sua
própria fila** ("unidade = local, grade de horário = agenda para um serviço dentro daquele
local"). Isso resolveu uma ambiguidade sinalizada desde 19/09 na skill `importacao-planilha`
("Grade de horários vs. Local no mapa — relação não confirmada").

### Modelo de dados (ver skill `modelo-dados` pro detalhe completo)

Nova tabela `grades` (unidade_id, servico, nome, + todo o SLA/painel/cooldown que antes
vivia em `unidades`). Migration `000004_grades` escrita, testada via psql, com **backfill
real**: as 175 linhas da planilha real viraram 2 grades automaticamente (uma pro serviço
real "Agendar Atendimento no Cadastro Único (CadÚnico)", uma "Geral" pros encaixes de teste
sem serviço). `guiches` passou de `unidade_id` pra `grade_id`. `agendamentos` ganhou
`grade_id` (not null). `usuarios` ganhou `grade_id` (nullable, só atendente) e
`ultimo_acesso_em` (base do status "pendente"/"último acesso" da nova tela de usuários).

### Backend — reescrita grande, mas incremental e testada em cada passo

- `internal/repository/grades.go` (novo): CRUD de grade + guichês agora escopados por
  `grade_id`. `unidades.go` simplificado (só identidade do local).
- `internal/handlers/fila.go`: `exigirGrade` (novo, análogo a `exigirUnidade`) — checa que a
  grade pertence à unidade do usuário E, se for atendente, que é exatamente a grade dele
  (gestor tem acesso a qualquer grade da própria unidade). Rotas de fila (guichês, sala de
  espera, chamados, chamar-próximo, atendidos/ausentes-hoje) migraram de
  `/unidades/{unidadeId}/...` pra `/grades/{gradeId}/...`. Recepção (`recepcao`, `encaixe`,
  `confirmar-chegada`, `desfazer-chegada`) continua em `/unidades/...` — a pessoa chega
  fisicamente num local, não numa grade.
- `internal/handlers/painel.go`: `GET /painel/{gradeId}` (era `{unidadeId}`) — cada grade tem
  seu próprio painel público agora. Nome exibido combina "Unidade — Grade" quando a unidade
  tem mais de uma grade ativa.
- `internal/handlers/usuarios.go` (novo): `GET /unidades/{unidadeId}/usuarios` e
  `PATCH /usuarios/{usuarioId}` (nome, email, senha opcional, grade alocada) — resolve de
  vez a "Gestão de atendentes" que estava marcada como pendente desde 19/09.
- `internal/handlers/importacao.go` + novo `importacao_preview.go`: importação agora resolve
  `grade_id` via (unidade, serviço) além de `unidade_id` via local — mesma lógica de
  auto-criação. Novo endpoint `POST /secretarias/{id}/lotes/preview` faz toda a validação
  **sem gravar nada**, pro fluxo de confirmação em duas etapas do gestor.
- `internal/repository/dashboard.go`: `ResumoDoDia` ganhou `serieDiaria` (últimos 7 dias,
  atendido vs. ausente, dias sem movimento preenchidos com zero pra não quebrar o eixo do
  gráfico) e `porGrade` (atendimentos por grade hoje).
- **Bug real encontrado e corrigido ao testar ao vivo**: vários slices Go declarados como
  `var x []T` (nil quando vazios) viravam `null` no JSON em vez de `[]` — o frontend quebrava
  tentando ler `.length` de `null` (caso mais comum: nenhum erro/unidade nova na análise de
  importação). Corrigido inicializando como slice vazio (`x := []T{}`) em
  `importacao_preview.go`, `importacao.go` e `dashboard.go`; guardas defensivas (`?? []`)
  também adicionadas no frontend por precaução.

### Frontend — tela do gestor reconstruída do zero

Sidebar lateral clássica (`GestorLayout.tsx`, ver skill `identidade-visual`) substituindo a
página única de antes, com roteamento aninhado (`/gestor`, `/gestor/grades`,
`/gestor/grades/:gradeId`, `/gestor/usuarios`, `/gestor/importar`):
- **Dashboard**: 3 blocos de total + lista de progressão "atendido por atendente" +
  gráfico de linha SVG feito à mão (sem biblioteca) atendido-vs-ausente 7 dias + lista de
  progressão extra "atendimentos por grade".
- **Grade de horário**: lista com copiar-link-do-painel + engrenagem → tela de configurações
  por grade (era "Gerenciar CRAS" por unidade).
- **Usuários internos**: tabela nome/cargo/grade/status + modal de edição completo.
- **Importar**: zona de arrastar-e-soltar + fluxo de confirmação em duas etapas (analisa →
  mostra resumo por dia + unidades/grades novas + erros → confirma).

Todas as telas que dependiam de `unidadeId` pra fila (`SalaDeEspera.tsx`, `PainelTV.tsx`)
migraram pra `gradeId`; `Recepcao.tsx` ganhou um seletor de grade no modal de encaixe (só
aparece se a unidade tiver mais de uma).

### Testado ao vivo, de ponta a ponta

Login retornando `gradeId`/`gradeNome` corretos; guichês/sala-espera/chamados/atendidos-hoje/
ausentes-hoje por grade via curl; dashboard com `serieDiaria`/`porGrade` reais; preview de
importação com a planilha real (175 linhas, todas já importadas antes → 175 erros
"protocolo já existe", confirmando a checagem funcionando sem gravar nada); upload real via
Chrome (drag-and-drop simulado) reproduziu e confirmou o bug do slice nil, corrigido e
re-testado; fila completa do atendente (guichê → chamar próximo → painel de TV) funcionando
de ponta a ponta pelo novo modelo de grade; tela de usuários com edição via modal; grade
settings salvando SLA de verdade (PATCH 200 confirmado via rede).

### Skills atualizadas nessa rodada

`modelo-dados` (nova entidade, mudanças em `unidades`/`guiches`/`agendamentos`/`usuarios`),
`importacao-planilha` (resolvida a ambiguidade Grade vs. Local vs. Serviço, fluxo de duas
etapas documentado), `papeis-e-telas` (reescrita quase completa — estava desatualizada desde
antes da rodada de órfãos/cooldown do dia 20/09), `regras-negocio-fila` (nota de tradução
unidade→grade pro SLA/cooldown/órfão).

### Pendências explícitas pra próxima etapa

Dono do produto avisou: "a partir daí vamos prosseguir pra a tela do admin e o fluxo de
acesso dos perfis" — próximo passo previsto é uma tela de admin master, ainda não desenhada
nem discutida em detalhe. Não implementar nada disso sem o desenho/conversa correspondente,
mesmo padrão já seguido neste projeto.

## Décima oitava rodada (22/09): 7 ajustes finos na tela do gestor reconstruída

Depois de ver a tela do gestor da rodada anterior funcionando, o dono do produto pediu 7
ajustes pontuais, todos implementados e verificados ao vivo no Chrome:

1. **Gráfico "Atendimentos por grade" virou proporção, não só volume** — antes mostrava só o
   total de atendidos por grade; agora mostra atendido-vs-ausente lado a lado com porcentagem
   de cada um. `internal/repository/dashboard.go`: `AtendimentosPorGrade` ganhou os campos
   `Atendidos`/`Ausentes` (era só `Total`), query trocou pra
   `count(*) filter (where status = 'atendido')` / `... 'ausente'` numa passada só. Novo
   componente `frontend/src/componentes/ListaProporcao.tsx` (barra de progresso dividida em
   duas cores) substituindo o uso de `ListaProgresso` nesse bloco específico do
   `Dashboard.tsx` (`ListaProgresso` continua sendo usado no bloco "atendido por atendente",
   que é mesmo sobre volume/ranking, não proporção).
2. **Hover nas bolinhas do gráfico de linha** — `GraficoLinha.tsx` reescrito com estado
   (`useState` de hover) em vez de só o `<title>` nativo do SVG (pequeno, demora a aparecer).
   Cada ponto tem um alvo de hover invisível maior (`r=9`, só pra facilitar acertar o mouse)
   sobreposto ao ponto visível menor; ao passar o mouse, um tooltip HTML posicionado por
   porcentagem (`left/top: ${(x/largura)*100}%`) mostra o dia e o valor exato daquela série.
   **Verificado ao vivo**: hover automatizado via clique/coordenada é impreciso demais pro
   alvo pequeno de um SVG escalado (várias tentativas não acertaram o alvo de 9px), mas
   invocar o handler React diretamente (via `__reactProps$...`) confirmou o tooltip renderiza
   corretamente com o texto certo ("21/09" / "Ausente: 152") na posição certa — a lógica em
   si está correta, a dificuldade era só de automação de mouse, não um bug do produto.
3. **Sidebar fixa, sem crescer com o scroll da tela** — `.gestor-shell` (index.css) ganhou
   `height: 100vh; overflow: hidden`; `.gestor-sidebar` e `.gestor-conteudo` ganharam
   `height: 100%; overflow-y: auto` cada um independente. Confirmado ao vivo rolando a tela
   de configurações de grade (mais longa que a viewport) — a sidebar (nav + rodapé com
   usuário/sair) ficou parada, só o conteúdo à direita rolou.
4. **Blocos de "Grade de horário setting" reorganizados** — a causa raiz do "ficaram pequenos
   e encostados na esquerda" era `.cartao` ter `max-width: 480px` por padrão (pensado pro
   formulário de login) e a página do gestor herdar isso sem override. Adicionado
   `.pagina-gestor .cartao { max-width: none; }` no `index.css` — os blocos de Guichês/SLA
   recepção/SLA atendimento/Configurações do painel agora ocupam a largura real do conteúdo.
5. **Gestor pode alterar o cargo (papel) de um usuário** — antes só dava pra alocar a grade,
   trocar nome/email/senha. `internal/repository/usuarios.go`: `AtualizacaoUsuario` ganhou
   `Papel domain.Papel`, incluído no `UPDATE`. `internal/handlers/usuarios.go`: novo
   `papelValido()` valida contra os 3 papéis reais antes de aceitar. Frontend
   (`Usuarios.tsx`): modal de edição ganhou um `<select>` "Cargo" com Gestor/Atendente/
   Recepcionista.
6. **Correção semântica: nome da grade vem da planilha, não duplica o serviço** — bug real da
   rodada anterior: `ResolverOuCriarGrade` estava usando o texto de "Serviço" tanto pra
   `servico` (chave de roteamento) quanto pra `nome` (rótulo de exibição), quando na real
   existe uma coluna própria "Grade de horários" na planilha que deveria ser o nome.
   `internal/handlers/importacao.go`: agora lê `nomeGradeTexto := celula(linha, indices,
   "grade")` separado de `servicoTexto`, e passa os dois pra `ResolverOuCriarGrade(ctx,
   unidadeID, servico, nome)` — `nome` só cai de volta pro valor de `servico` se a célula da
   coluna "Grade de horários" vier vazia. As 2 grades já existentes no banco (criadas pela
   rodada anterior, antes desse fix) continuam com nome=serviço porque foram criadas pelo
   código antigo — não é um bug remanescente, só dado histórico; qualquer grade nova criada
   por um upload a partir de agora usa a coluna certa.
7. **Bloco "Identificação da grade" removido** — como o nome não é mais editável na UI (vem
   da planilha), o card com o campo "Nome de exibição" não fazia mais sentido.
   `GradeConfiguracao.tsx`: cabeçalho da página reestruturado — o nome da grade virou o
   `<h1>` da página, "Serviço: X" virou um subtítulo abaixo dele, e o botão de ativar/
   desativar a grade (que antes vivia dentro desse card removido) subiu pro topo da página,
   ao lado do título.

**Tudo verificado ao vivo no Chrome** (não só por build): dashboard com a barra de proporção
mostrando "14 atendidos (8%) · 152 ausentes (92%)", tooltip do gráfico de linha confirmado via
invocação direta do handler React, sidebar parada durante scroll em duas telas diferentes
(dashboard e grade settings), blocos de configuração ocupando a largura cheia, modal de
usuário com o select de cargo com as 3 opções corretas, e o cabeçalho novo da grade settings
(nome como título + serviço como subtítulo + botão ativar/desativar no topo, sem o card de
identificação). `go build ./...`, `go vet`, `gofmt` e `npx tsc --noEmit -p tsconfig.app.json`
+ `npm run build` todos limpos antes da verificação ao vivo.

## Décima nona rodada (22/09): spec-kit real instalado via CLI oficial + constituição ratificada

O dono do produto perguntou se o spec-kit (`.specify/`, skills `speckit-*`) tinha mesmo sido
instalado pela ferramenta oficial do GitHub ou só recriado manualmente numa sessão anterior.
Verificação confirmou que já era uma instalação real (versão `1.0.9.dev0`, estrutura de
scripts/templates/manifests idêntica à do instalador oficial) — não uma recriação manual — mas
mesmo assim rodei `uvx --from git+https://github.com/github/spec-kit.git specify init --here
--force --non-interactive --integration claude --script ps` pra garantir que fosse a versão mais
recente de verdade, direto do repositório oficial, em vez de confiar num estado antigo. O
comando atualizou a instalação de `1.0.9.dev0` pra `1.0.10.dev0` e trouxe a skill
`speckit-converge` (nova desde a instalação anterior), sem tocar nos specs já existentes em
`specs/` nem nas skills de projeto (`modelo-dados`, `papeis-e-telas` etc., que vivem separadas
das skills `speckit-*`).

**"Adaptar" o spec-kit pro projeto**: a constituição (`.specify/memory/constitution.md`) ainda
estava com o molde genérico do template, nunca preenchida, apesar dos 3 specs formais já
existirem. Rodei `/speckit-constitution` de verdade (script `resolve-template.ps1` executado,
relatório de sync impact gerado) pra ratificar como constituição os princípios que o projeto já
seguia de fato, extraídos do próprio histórico deste arquivo: stack fixa sem ORM, dependências
mínimas e justificadas, verificação ponta a ponta antes de "pronto" (não-negociável), esclarecer
antes de decidir modelo de dados/escopo ambíguo, não implementar além do combinado, documentação
contínua em CLAUDE.md/skills, e simplicidade sobre abstração prematura — 7 princípios (o molde
tinha 5 de exemplo, ajustado ao conteúdo real). Primeira ratificação: versão 1.0.0, 22/09.

**Sem alterar**: nenhum código de aplicação foi tocado nesta rodada — escopo estritamente
limitado à instalação/atualização do spec-kit e ao preenchimento da constituição, como o próprio
comando `/speckit-constitution` exige (não mexe em rotas, componentes ou specs de feature).

**Aviso do instalador, não aplicado**: o CLI sugeriu adicionar `.claude/` ao `.gitignore` por
risco genérico de credenciais em pastas de agente. Não apliquei — `.claude/skills/` neste
projeto guarda documentação de produto real (schema, regras de negócio, convenções), não
credenciais, e é exatamente o conteúdo que precisa ir pro controle de versão pro próximo dev
herdar o contexto. Sinalizando aqui caso o dono do produto queira revisitar essa escolha quando
o projeto ganhar git de verdade (ainda não é um repositório git nesta máquina).

## Vigésima rodada (22/09): bug real de hover em botão, descrições redundantes removidas, princípios de UI/UX viraram regra na skill

Pedido: corrigir o hover na tela do gestor, tirar as caixas azuis de descrição que "não
precisa disso", colocar o título "Guichês e salas" dentro do bloco como os outros de SLA, e
formalizar na skill `identidade-visual` os princípios de UI/UX que o dono do produto vem
repetindo pra não precisar pedir de novo.

**Bug real de CSS encontrado investigando o hover**: `.linha-guiche-resumo:hover` (linha de
guichê/sala clicável) estava tecnicamente definida com a cor suave certa (`--cor-hover`), mas
na prática o hover mostrava um preenchimento sólido azul-marinho escuro com texto quase
ilegível — bem mais forte que o pretendido. Causa raiz: especificidade de CSS. O botão global
já tem `button:not(:disabled):hover { background: var(--cor-primaria-forte); }` (elemento +
duas pseudo-classes); `.linha-guiche-resumo:hover` é só classe + uma pseudo-classe, menos
específico, então **perde** — o hover forte do botão genérico vencia por baixo dos panos.
Confirmado visualmente (zoom numa captura real, não só lendo CSS — testes via
`getComputedStyle` sozinhos deram falso negativo por flakiness da automação de mouse, mesmo
problema já visto antes nesta sessão com o hover do gráfico de linha). O mesmo bug afetava
mais 3 lugares que usam exatamente o mesmo padrão (`<button className="X">` com hover
próprio): `.botao-voltar` (botão "← Grade de horário"), `.aba` (abas da recepção/sala de
espera) e `.quadro-guiche` (quadrados do modal de seleção de guichê do atendente) — nenhum
desses tinha sido notado quebrado antes porque o sintoma só aparece passando o mouse de
verdade, nunca lendo o código. Corrigido nos 4 lugares adicionando `:not(:disabled)` ao
seletor (`.linha-guiche-resumo:not(:disabled):hover` etc.), igualando a especificidade —
mesmo padrão que `.acesso-secundario:not(:disabled):hover` (Login) já usava certo desde
antes. Comentário de alerta adicionado direto no `index.css`, ao lado da regra global, pra
qualquer hover customizado futuro em `<button>` lembrar de repetir o padrão.

**Caixas `.aviso` (fundo azul) removidas por serem redundantes** — `Usuarios.tsx` ("Nome,
cargo e status..."), `Grades.tsx` ("Cada grade é a fila de um serviço..."),
`GradeConfiguracao.tsx` ("Clique numa linha pra abrir e personalizar..."),
`Dashboard.tsx` (duas: "Quem atendeu mais hoje..." e "Proporção de atendido vs. ausente...")
— todas só reafirmavam o que o título ou as colunas da tabela já deixavam óbvio. Duas
permaneceram, mas trocaram de estilo: "Serviço: X" (subtítulo da grade) e as duas explicações
de SLA (que esclarecem uma regra de negócio genuinamente não óbvia — ex. que o tempo mínimo
entre chamadas é uma fatia do prazo de ausência, não somado) viraram uma classe nova, `.dica`
— texto discreto, sem fundo, sem o peso visual de uma caixa `.aviso`. `.aviso` continua
existindo e sendo usado como antes nos lugares genuinamente dinâmicos/funcionais (resultado
de importação, motivo de bloqueio do atendente) — não foi removida a classe, só o uso
indevido dela como "legenda de tela".

**"Guichês e salas" agora dentro do cartão**: era um `<h2>` solto acima de `<section
className="cartao">`, inconsistente com os outros 3 blocos da mesma tela (SLA da recepção,
SLA do atendimento, Configurações do painel), todos com `<h3>` dentro do cartão. Virou
`<h3>Guichês e salas</h3>` como primeiro filho do cartão, igual aos outros. A regra `.secao-
titulo` (só usada ali) ficou órfã e foi removida do CSS.

**Skill `identidade-visual` ganhou 3 seções novas** pra essas regras não precisarem ser
repetidas: "Armadilha de especificidade: hover custom num `<button>`" (a regra do
`:not(:disabled)`), "Texto sem função — não descrever o óbvio" (quando usar `.aviso` vs.
`.dica` vs. nada) e "Título de bloco de configuração fica dentro do cartão" (distinguindo
esse padrão do padrão diferente, e também válido, dos blocos de tabela com título solto).

**Testado ao vivo no Chrome**: hover do "Guichê 1" e do botão "← Grade de horário" confirmados
com a cor suave certa (nem sinal do preenchimento escuro de antes); Dashboard, Grade de
horário e Grade settings confirmados sem nenhuma caixa azul redundante, com "Serviço: X" e as
duas notas de SLA como texto discreto. `tsc --noEmit -p tsconfig.app.json` e `npm run build`
limpos.

## Vigésima primeira rodada (22/09): tela e estrutura do Admin da plataforma

Pedido grande do dono do produto: uma área de Admin com controle master sobre todas as
prefeituras, secretarias e usuários da plataforma, seguindo a hierarquia
`Admin → Prefeitura → Secretaria → Usuários`, com autenticação/autorização "tratadas como
base de segurança" antes de qualquer tela de negócio nova. Pedi pra analisar o projeto atual
antes de codificar (como pedido) e expliquei o plano antes de implementar — resumo do que foi
decidido e feito:

**Duas decisões de interpretação, sinalizadas explicitamente antes de implementar** (em vez
de travar perguntando, já que o custo de errar num banco de dev é baixo):
1. **Recepcionista continua vinculada à UNIDADE inteira**, não a uma grade — o diagrama do
   pedido original mostrava Recepcionista embaixo de "Grade", mas isso contradiria a decisão
   de negócio já confirmada e testada (a recepção vê todo mundo que chega num local físico,
   não só quem tem hora marcada numa fila específica). Atendente continua o único papel preso
   a uma grade.
2. **`PAINEL_CHAMADA` continua público, sem login** — só formalizado como valor no enum
   central de papéis (`domain.PapelPainelChamada`), sem criar tela de autenticação pra TV.

**Mudança estrutural real**: Gestor deixou de ser vinculado a uma UNIDADE (só funcionava
porque o piloto tem uma unidade só) e passou a ser vinculado à SECRETARIA inteira — pode ver/
gerenciar todas as unidades e grades dela agora. Isso exigiu migrar dashboard, grade de
horário e usuários internos do gestor de `/unidades/{id}/...` pra `/secretarias/{id}/...`
(agregando via subquery/join) — ver skill `modelo-dados` pro detalhe do schema e
`papeis-e-telas` pro detalhe de tela.

**Backend**:
- Migration `000005_admin_hierarquia`: `usuarios.papel` ganha `admin`; `unidade_id` vira
  nullable; nova coluna `secretaria_id` (gestor, com backfill a partir da unidade antiga);
  email virou único globalmente (era só por unidade); `prefeituras` ganha `cor_destaque`/
  `cor_clara` (default = paleta atual); usuário admin de teste inserido direto na migration.
- `internal/auth/permissoes.go` (novo): centraliza toda decisão de acesso (`EhAdmin`,
  `PodeAcessarSecretaria`, `PodeAcessarUnidade`, `PodeAcessarGrade`) — pedido explícito do
  dono do produto de não espalhar `if papel == "x"` em cada handler. `exigirUnidade`/
  `exigirGrade` (já existiam) e o novo `exigirSecretaria` passaram a delegar pra essas
  funções em vez de comparar IDs direto — refatorado em ~15 pontos espalhados por
  `fila.go`/`usuarios.go`/`importacao.go`/`configuracao.go`.
- `internal/handlers/admin.go` (novo): CRUD mínimo de prefeitura (+ as 2 cores), criar
  secretaria, criar gestor, criar atendente/recepcionista, listagens enriquecidas
  (`/admin/gestores`, `/admin/funcionarios` — já vêm com nome de prefeitura/secretaria/
  unidade/grade resolvido via join, sem N+1 no frontend) e `/admin/resumo` (contadores).
- **Bug real encontrado e corrigido durante o teste ao vivo**: ao mover `GetGrades` pra
  escopo de secretaria (só gestor/admin), a recepção perdeu acesso ao endpoint que usava pra
  listar as grades da PRÓPRIA unidade no formulário de encaixe (recepcionista não tem
  `secretaria_id` direto nem pode acessar o endpoint gestor-only). Corrigido criando um
  endpoint distinto e propositalmente separado, `GET /unidades/{unidadeId}/grades`
  (`GetGradesDaUnidade`, recepcionista/gestor/admin) — não confundir com `GET
  /secretarias/{id}/grades` (gestor/admin, secretaria inteira, tela de gestão).

**Frontend**: `/admin` novo, mesmo padrão de sidebar do `GestorLayout` (Dashboard,
Prefeituras, Gestores, Funcionários — reaproveitando as classes CSS `gestor-*`, que já eram
genéricas apesar do nome). Botão "Admin" novo na tela de login (mesmo padrão de acesso
rápido). `frontend/src/tema.ts` (novo): aplica as 2 cores da prefeitura em runtime nos tokens
`--cor-primaria`/`--cor-info-fundo` já existentes, chamado depois de resolver a sessão em
cada tela autenticada — sem precisar de build separado por prefeitura.

**Correção correlata**: o nome da prefeitura no cabeçalho/sidebar (`Cabecalho.tsx`,
`GestorLayout.tsx`) era um texto **fixo** desde 21/09, documentado explicitamente como "só
muda quando o sistema atender mais de uma prefeitura" — esse dia chegou nesta rodada, então
virou dinâmico (`usuario.prefeituraNome`, resolvido no backend via secretaria → prefeitura),
com o texto antigo como fallback só pro caso raro da sessão não resolver ainda.

**Testado ao vivo, de ponta a ponta**: login admin (sem unidade/secretaria/cores, como
esperado); criar prefeitura → criar secretaria → criar gestor via curl, confirmando a cadeia
inteira; **isolamento de tenant confirmado** (o gestor novo recebeu 403 tentando acessar
grades da secretaria antiga); PATCH de cor testado e confirmado aplicando ao vivo no Chrome —
logou como o gestor real (SASC) depois de mudar a cor da prefeitura pra roxo e viu Dashboard/
Grade de horário/Usuários internos inteiros com a paleta nova (`--cor-primaria` e
`--cor-info-fundo` seguindo a cor escolhida), sem quebrar nenhuma tela existente; recepção e
atendente testados ao vivo depois do fix do bug de grades (modal de encaixe voltou a mostrar
o seletor de grade). Cores revertidas pro padrão e dado de teste (prefeitura/secretaria/
gestor fictícios) removido do banco depois da verificação, pra não confundir o estado real.
`go build`/`go vet`/`gofmt` e `tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos o
tempo todo.

**Não implementado nesta rodada** (deliberadamente, conforme prioridade pedida — estrutura
de acesso primeiro, CRUDs de negócio depois): edição/desativação de prefeitura ou secretaria
existente além do que já existe, dashboard de negócio pro admin (só contadores simples),
qualquer tela de "trocar meu próprio papel", e o fluxo de JWT real (a sessão já era desenhada
pra trocar sem reestruturar — nada mudou nesse mecanismo, só o que `Usuario` carrega).

## Vigésima segunda rodada (22/09): login real de verdade + ocupação de guichê em tempo real

Dois pedidos pontuais depois da tela de Admin:

**1. Login real (email + senha), sem cadastro nenhum.** Reverte a decisão de 20/09 (que
tinha trocado o login por 4 botões de acesso rápido, porque "esse login não vai ser usado" —
o SSO real ficaria pro pós-MVP). Agora que o Admin cria contas de verdade (gestor,
atendente, recepcionista — rodada anterior), o login real voltou a fazer sentido: contas só
existem se alguém da gestão criar. `frontend/src/telas/login/Login.tsx` reescrito — formulário
de email/senha, sem link de cadastro, com uma linha de apoio ("Não tem acesso? Fale com o
gestor da sua secretaria ou com o administrador da plataforma"), redirecionando por papel
(`admin→/admin`, `gestor→/gestor`, `recepcionista→/recepcao`, `atendente→/sala-espera`). O
atalho de acesso direto ao Painel de TV também saiu da tela de login — o link público de
cada grade já se copia direto na tela "Grade de horário" do gestor. O mecanismo de sessão em
si (cookie HttpOnly, `POST /api/v1/login`) não mudou nada, só a UI que chega até ele.

**2. Ocupação de guichê em tempo real.** Pedido: se são 3 guichês e um atendente escolhe o
2, os outros atendentes devem ver o 1 e o 3 livres e o 2 como ocupado — antes disso, a
escolha só era lembrada em `localStorage` (local a cada navegador, sem visibilidade entre
atendentes).

- **Migration `000006_ocupacao_guiche`**: `guiches` ganha `ocupado_por_usuario_id` +
  `ocupado_em` (heartbeat).
- **Mecanismo** (ver skill `regras-negocio-fila` pro detalhe completo): escolher um guichê
  no modal chama `POST /grades/{id}/guiches/{id}/ocupar`; o frontend reafirma a cada 5s
  enquanto o atendente continua ali. Um guichê só aparece ocupado pros outros enquanto o
  heartbeat estiver fresco (15s) — depois disso volta a aparecer livre sozinho, sem job de
  limpeza, cobrindo aba fechada/crash sem logout. "Sair" libera explicitamente
  (`POST .../liberar`) na hora, sem esperar a janela expirar. Conflito (dois atendentes
  escolhendo o mesmo guichê ao mesmo tempo) é resolvido atomicamente no `UPDATE` — só um
  sucede, o outro recebe 409 `guiche_ocupado`.
- **Frontend**: modal de seleção (`SalaDeEspera.tsx`) mostra guichês ocupados por outro
  desabilitados, com "Ocupado por {nome}"; atualiza a lista a cada 2s enquanto o modal está
  aberto, pra refletir mudanças de outros atendentes em tempo quase real.
- **Escopo documentado explicitamente**: essa regra é sobre a ESCOLHA do guichê no modal, não
  uma trava adicional em cima de `chamar-próximo` (que continua aceitando qualquer guichê da
  grade sem checar ocupação) — não presumir uma trava mais rígida do que foi pedido.

**Testado ao vivo no Chrome, com dois atendentes reais** (criado um segundo atendente de
teste via Admin especificamente pra esse teste, removido depois): logado como "Atendente
Teste" escolhendo Guichê 2, logado como "Atendente Dois" em outra aba viu Guichê 1 e Sala 1
livres e **Guichê 2 desabilitado, com "Ocupado por Atendente Teste"** — exatamente o pedido.
Confirmado também via curl: conflito real dá 409 `guiche_ocupado`; liberar libera na hora;
guichê "abandonado" (heartbeat parado) volta a aparecer livre sozinho depois da janela, sem
ação manual. `go build`/`go vet`/`gofmt` e `tsc --noEmit -p tsconfig.app.json`/`npm run build`
limpos.

## Vigésima terceira rodada (22/09): criação condicional de usuário, botão do gestor e reverificação da fila

Três pedidos: consolidar a criação de usuário do admin numa tela só com formulário
condicional, dar ao gestor seu próprio botão de criar funcionário (estava faltando) com
restrições corretas, e investigar um possível bug na fila de chamada relatado pelo dono do
produto.

**Admin — "Usuários internos" consolidada.** Antes eram duas telas separadas ("Gestores" e
"Funcionários"); agora é uma só (`/admin/usuarios`, `UsuariosInternos.tsx`), com um único
botão "Criar usuário" cujo formulário muda pelo Cargo escolhido: Gestor pede só Prefeitura →
Secretaria; Atendente/Recepcionista pedem Prefeitura → Secretaria → **Grade** (a grade já diz
a unidade — não pergunta Unidade separadamente). Backend: `GET /admin/usuarios` (novo,
substitui `/admin/gestores` e `/admin/funcionarios`, removidos) — `ListarTodosUsuariosInternos`
faz um `select` só com `papel in ('gestor','atendente','recepcionista')`.

**Gestor — botão de criar funcionário que estava faltando.** `PostFuncionario` (antes
admin-only) passou a aceitar gestor também, validando que a unidade pertence à própria
secretaria (reaproveita `usuarioPodeAcessarUnidade`, a mesma função central de sempre — nada
de checagem nova solta). O formulário do gestor (`Usuarios.tsx`) **não pergunta Prefeitura
nem Secretaria** — são sempre as dele, óbvio e obrigatório — só Cargo + Grade + dados
pessoais.

**Gestor não pode mudar o próprio cargo.** Backend (`PatchUsuario`) rejeita com 403
`nao_pode_mudar_proprio_cargo` se `usuarioID == usuarioLogado.ID && papel != papel atual`, só
pra gestor (admin nem aparece nessa lista, então não se aplica na prática). Frontend desabilita
o select "Cargo" no modal de edição quando é a própria linha, com uma nota explicando por quê;
editar outra pessoa continua deixando trocar o cargo livremente.

**Fila de chamada — investigado, não era bug.** O dono do produto relatou: chamou uma pessoa
no Guichê 2, depois logou como outro atendente e ficou bloqueado de chamar alguém pro próprio
guichê "porque o outro estava pendente". Reproduzi o cenário exato com 4 atendentes reais via
curl (documentado em detalhe na skill `regras-negocio-fila`): uma chamada pendente **só
bloqueia quem a fez** — dois outros atendentes conseguiram chamar normalmente enquanto ela
esperava rechamada. Só depois que o dono rechamou e seguiu em frente (virou um chamado órfão
de verdade) é que um quarto atendente, sem nenhuma pendência própria, foi corretamente
bloqueado até assumir o órfão. Ou seja, a regra já estava certa — o relato provavelmente veio
de um chamado órfão real deixado por teste anterior na mesma grade, confundido com "o outro
atendente está bloqueando". Nenhuma mudança de código nessa parte, só verificação.

**Testado ao vivo**: form condicional do admin confirmado trocando Cargo entre Gestor/
Atendente no Chrome (campo Grade aparece/some corretamente); botão "Criar funcionário" do
gestor confirmado abrindo sem pedir Prefeitura/Secretaria; edição da própria conta do gestor
confirmada com o Cargo desabilitado e a nota de aviso, e edição de outra pessoa confirmada
com o Cargo livre; os 403 de auto-mudança de cargo e de criação fora da secretaria confirmados
via curl. Dado de teste (4 atendentes extras, 5 encaixes de teste, ocupação de guichê)
totalmente revertido ao final. `go build`/`go vet`/`gofmt` e
`tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos.

## Décima nona rodada (22/09): redesenho visual do painel de TV + barra de contagem regressiva

Pedido direto do dono do produto, enquanto ele testava a rodada anterior (admin/usuários
internos): o painel de TV estava com fundo escuro, tabela de últimas chamadas sem nenhuma
separação visual ("feia e sem graça, principalmente quando ela estiver completa na tela"),
protocolo/guichê pequenos demais pra leitura a distância, sem indicação visual de quanto
tempo falta pra chamada atual sair de destaque, e pediu pra não usar uma fonte "clássica" —
usar a mesma já usada no resto do site.

**Backend** (`internal/painel/fila_exibicao.go` + `internal/handlers/painel.go`): `Atual(...)`
passou a devolver também `expiraEm` (timestamp) e a duração total usada pra aquela chamada,
sem mudar a lógica de decisão em si (continua só o backend quem decide quando trocar de
exibição). `GET /painel/{gradeId}` passou a expor `chamadaAtual.expiraEm`/
`duracaoTotalSegundos` (pra desenhar a barra) e `corDestaque`/`corClara` da prefeitura no
nível raiz da resposta, resolvidos via unidade→secretaria→prefeitura (o painel é público,
sem sessão, não pode reaproveitar dado de sessão pra isso).

**Frontend — tema**: nova função `aplicarTemaPainel` em `tema.ts`, separada de
`aplicarTemaPrefeitura` (que só roda em tela autenticada) — aplica `corClara`/`corDestaque`
só nos tokens `--painel-*`, mantendo o namespace do painel isolado dos tokens operacionais
`--cor-*`, mas agora alimentado pela mesma configuração de cor da prefeitura. Fundo da tela
passou a ser a cor clara/secundária; os dois blocos de conteúdo (chamada atual + tabela)
viraram cartões brancos por cima, pra manter contraste/legibilidade a distância — mesmo
padrão "fundo tintado + cartão branco" já usado nas outras telas.

**Frontend — `PainelTV.tsx`**: novo componente `BarraTempo` — barra de progresso regressiva
sob a chamada em destaque, puramente decorativa (só reflete a decisão do backend, nunca
influencia ela). A duração restante é calculada uma única vez por chamada (`useState` com
inicializador lazy, `key` no componente pai baseada em protocolo+guichê pra só remontar
quando a chamada muda de verdade) — evita que o polling de 2s reinicie a animação a cada
ciclo. A barra em si é CSS puro (`animation` + `transform: scaleX()`), sem JS atualizando
estado por frame.

**Frontend — CSS**: tabela de últimas chamadas reorganizada com cabeçalho de coluna (Nome/
Protocolo/Guichê), zebra striping, protocolo em destaque (cor de acento + fonte mono) e
guichê como pill arredondado — antes era texto corrido sem nenhuma separação. Protocolo e
guichê da chamada em destaque aumentados (`clamp()` responsivo, maior mas não exagerado).
Tons derivados do acento (cabeçalho da tabela, pill) calculados via `color-mix(in srgb,
var(--painel-acento) 12%, white)` em vez de referenciar um token operacional (`--cor-info-
fundo` nunca seria atualizado nessa tela, que não roda `aplicarTemaPrefeitura`). Tipografia
já era `Public Sans`/`IBM Plex Mono` (mesma do resto do site desde a décima quarta rodada) —
mantida, não trocada por nada "clássico".

**Testado ao vivo no Chrome**: criado um encaixe de teste ("Teste Painel Visual") e chamado
de verdade via curl pra popular uma chamada ativa — confirmado fundo tintado (`#e4f1f5`),
os dois blocos como cartões brancos com sombra, protocolo+guichê maiores com divisor vertical
entre eles, barra de contagem visivelmente decrescendo. Depois que a exibição expirou sem
próxima chamada na fila, o painel entrou em modo descanso e a tabela completa (10 linhas)
renderizou corretamente centralizada, com cabeçalho, zebra striping, protocolo em mono e
pills de guichê — confirmando que o caso "tabela cheia" que o dono do produto sinalizou
como o pior cenário também ficou organizado. Fonte confirmada via `getComputedStyle`
(`Public Sans` no corpo, `IBM Plex Mono` no protocolo). Chamada de teste marcada como
atendida ao final pra não deixar dado de teste artificial pendente na fila.
`go build`/`go vet`/`gofmt` e `npx tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos.

Skills atualizadas: `identidade-visual` (seção do painel reescrita — não é mais "tema 100%
isolado", agora recebe as cores da prefeitura via `aplicarTemaPainel`) e `painel-tv` (nova
seção documentando a barra de contagem e o redesenho da tabela).

## Vigésima quarta rodada (23/09): modo "aguardando" no painel, regra de cooldown corrigida, 2 bugs reais

Pedido grande do dono do produto depois de observar o painel/fila de verdade em uso — uma
mensagem só, organizada em partes por mim antes de implementar (bugs reais primeiro, depois
o modo novo). Tudo verificado ao vivo (curl com timestamps forjados no banco pra não esperar
minutos reais de cooldown, e no Chrome pro botão novo).

**1. Regra de cooldown corrigida (mudança de comportamento real, não só bug)**: até aqui, um
atendente ficava livre pra chamar outra pessoa **imediatamente depois de rechamar uma vez**
(`TemChamadaPendenteNaoRechamada` só olhava `cnt.total >= 2`), mesmo com um cooldown novo
(contando até a rechamada seguinte) ainda visível na tela. Relatado como errado: "se eu estou
chamando alguém nesse momento e contando os 3min, eu não posso chamar outra pessoa até o
prazo terminar." Corrigido pra comparar `now() < última_chamada + cooldown` em vez de contar
quantas vezes já rechamou — o gate agora é sempre o cooldown em si, não o número de chamadas.
A mecânica de chamado órfão (rechamar e seguir em frente abandona a pessoa) continua
existindo do mesmo jeito, só que só fica disponível depois que o cooldown da última chamada
passar, nunca "logo depois de rechamar uma vez". `TemChamadaPendenteNaoRechamada` e
`ChamarProximo` (`backend/internal/repository/agendamentos.go`) ganharam um parâmetro
`cooldownMinutos`. Ver skill `regras-negocio-fila`.

**2. Bug real: "últimas chamadas" duplicava a mesma pessoa depois de rechamar.**
`UltimasChamadas` (`backend/internal/repository/chamadas.go`) lia direto o log `chamadas`
(uma linha por chamar/rechamar, append-only de propósito) sem deduplicar — corrigido pra uma
linha por agendamento (a mais recente), reaproveitando os mesmos fragmentos SQL
(`juncoesUltimaChamada`/`condicaoOrfao`) já usados em `ChamarProximo`/
`ListarChamadosCompartilhados`, que também passaram a expor `chamada_id`/`guiche_id` (antes
só `usuario_id`/`chamado_em`).

**3. Bug real: narração podia sobrepor sob chamadas em sequência rápida** (vários atendentes
chamando quase ao mesmo tempo) — `speechSynthesis.speak()` enfileira por padrão, não cancela
a fala anterior, então o painel podia já mostrar a pessoa B na tela com o áudio ainda
terminando de falar o nome de A. Corrigido com `speechSynthesis.cancel()` no início de cada
novo anúncio (`PainelTV.tsx`, `iniciarAnuncio`).

**4. Novo modo "aguardando" no painel de TV** (pedido central desta rodada): quando a
exibição de 30s/15s de uma chamada expira e não há chamada nova pra mostrar, a pessoa não
desaparece mais direto pra tabela se o atendente ainda estiver genuinamente esperando por
ela (chamado, não órfã, dentro do prazo) — fica visível como um bloco na área de destaque, e
se houver 1-3 pessoas nessa situação ao mesmo tempo, todas aparecem lado a lado (tamanho de
fonte se adapta à quantidade). Sem narração, sem barra de tempo — presença visual só. Chamado
órfão explicitamente NUNCA entra aqui ("se estiver órfão não fica no painel") — continua só
na tabela. Backend: `ListarAguardandoExibicao` (nova, `agendamentos.go`) + `GET
/painel/{gradeId}` ganhou o campo `aguardando` (só preenchido quando não há `chamadaAtual`) e
`ultimasChamadas` ganhou `ehOrfao` por linha, além de excluir da tabela quem já estiver
ocupando a chamada atual ou os blocos de aguardando. Frontend: `PainelTV.tsx` ganhou um
terceiro modo (`modo-aguardando`, mesmo layout 70/30 do modo ativo) e um quarto ícone de
status **cinza** (`Clock`, lucide-react) — "chamado, dentro do prazo, ainda não apareceu" —
distinto do ícone âmbar (`Volume2`), que ficou reservado só pra órfão de verdade. Ver skill
`painel-tv`.

**5. Botão de ausência manual pro atendente** (`POST /agendamentos/{id}/ausencia`, ícone
`UserX` vermelho ao lado de "marcar presente" em "Chamando agora") — pro caso de alguém
avisar que o cidadão já foi embora, sem precisar esperar o sweeper automático rodar até o
fim do SLA de atendimento. Mesma regra de posse dos outros botões (só dono, status precisa
ser `chamado`), sem cooldown associado.

**Testado ao vivo**: regra de cooldown reproduzida via curl com `chamado_em` forjado no banco
(psql) pra simular os minutos sem esperar de verdade — confirmado bloqueio logo após
rechamar, depois liberação só após o cooldown "passar", órfão se formando corretamente na
sequência certa. Modo aguardando confirmado via curl (pessoa aparece em `aguardando` só
depois que `chamadaAtual` fica nulo, órfão real ficando de fora, dedup correto na tabela) E
visualmente no Chrome (bloco único centralizado, sem barra de tempo, ícone âmbar certo pro
órfão na tabela). Botão de ausência manual testado ao vivo no Chrome (clique real → toast
"Marcado como ausente." → some de Pendentes → conta em Ausentes). Todo dado de teste
resolvido ao final (marcado atendido/ausente, guichês liberados) — nenhum ficou pendente na
fila real. `go build`/`go vet`/`gofmt` e `npx tsc --noEmit -p tsconfig.app.json`/
`npm run build` limpos.

## Vigésima quinta rodada (23/09): título do painel só com a grade, tabela do cabeçalho corrigida

Dois ajustes rápidos pedidos logo depois da rodada anterior:

**Título do painel sem o nome da unidade/serviço**: `nomeExibicaoPainel`
(`backend/internal/handlers/painel.go`) combinava "Unidade — Grade" — pedido: "tire o nome do
serviço do título do painel, deixe somente a grade de horário". Agora devolve só o nome da
grade (a unidade continua como fallback só quando a grade é "Geral"/sem nome, a grade
auto-criada pra encaixe sem serviço). **Ressalva importante**: pra grade real (CadÚnico) já
existente no banco, o título ainda vai aparecer igual ao nome do serviço na prática — não
porque o código esteja errado, mas porque essa grade específica foi criada ANTES da correção
da décima oitava rodada (22/09) que separou `nome` de `servico` na importação, então
`grades.nome` dela ainda está gravado igual a `grades.servico` (dado histórico, não bug —
já documentado na décima oitava rodada). Corrige sozinho num upload novo da planilha; não
mexi no dado antigo por não ter certeza de qual devia ser o nome real da grade de horário.

**Tabela "últimas chamadas" alinhada ao padrão do resto do sistema**: cabeçalho
(`.historico-cabecalho`) tinha os 4 cantos arredondados (`border-radius: 10px`), fazendo ele
flutuar como uma pílula solta acima da lista em vez de parecer um `<th>` de tabela de
verdade — corrigido pra arredondar só os cantos de CIMA (`8px 8px 0 0`, mesmo valor do
`th:first-child`/`th:last-child` do resto do site) com uma borda embaixo costurando ele na
lista. Fundo trocado de um `color-mix` com `--painel-acento` (ficava escuro/saturado demais —
"clareie mais a linha com cor azul") pra `--painel-fundo` puro, que é literalmente a mesma
cor (`corClara`) que `--cor-info-fundo` usa nos `<th>` padrão do resto do sistema — mesma
origem, resultado visual consistente. Zebra da lista (`.historico li:nth-child(even)`)
também trocada: usava o mesmo `--painel-fundo` cheio (forte demais pra zebra, e coincidia
com a cor do cabeçalho novo), virou um tint neutro bem sutil (`rgba(24, 36, 40, 0.035)`),
mesmo espírito do zebra padrão do resto do site.

Verificado ao vivo no Chrome: título mostrando só o nome da grade, cabeçalho da tabela com
fundo bem mais claro e cantos de baixo retos, costurado direto na primeira linha da lista.
`go build`/`go vet`/`gofmt` e `tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos.

## Vigésima sexta rodada (23/09): grade do modo "aguardando" com arranjo por quantidade (até 4)

Pedido de ajuste fino no modo "aguardando" (rodada anterior): layout específico por
quantidade de pessoas, em vez do flex-row simples de antes.

**Arranjo pedido, implementado como CSS Grid** (`.aguardando-grade-N` em `index.css`): 1
pessoa ocupa o espaço todo (igual antes); 2 empilham numa coluna só, mais recente em cima; 3
ficam com a mais recente em cima ocupando a largura toda + as outras duas lado a lado embaixo
(`grid-column: 1 / span 2` na primeira, via `nth-child`); 4 viram uma grade 2x2. O cap subiu
de 3 pra 4 (`ListarAguardandoExibicao` agora chamado com limite 4 em `painel.go`).

**"Mais recente em cima"**: o backend continua ordenando por chamada mais antiga primeiro
(assim decide quem entra quando há mais candidatos que o limite — quem espera há mais tempo
tem prioridade). `PainelTV.tsx` inverte essa lista só na hora de renderizar
(`[...aguardando].reverse()`), então o primeiro elemento do grid (topo/esquerda) é sempre o
mais recente, não importa quantos blocos existam.

**Estrutura interna do bloco corrigida pra "mesma estrutura do padrão"**: nome, protocolo e
guichê agora empilham em coluna dentro de cada bloco (antes: nome grande + protocolo/guichê
lado a lado com um traço vertical). Protocolo e guichê reaproveitam literalmente as mesmas
classes CSS da tabela padrão (`historico-protocolo`/`historico-guiche-pill`) em vez de uma
tipografia bespoke — o tamanho de cada bloco escala via `font-size` no próprio `.aguardando-
bloco` por quantidade (1/2/3/4), e como as duas classes reaproveitadas usam unidades `em`,
tudo escala junto automaticamente.

**Testado ao vivo com 4 pessoas reais simultâneas** — criei 2 contas de atendente descartáveis
via gestor (`Teste Layout C`/`D`, únicas assim porque só havia 2 contas de teste minhas
disponíveis pra simular concorrência) e chamei 4 cidadãos de teste de 4 contas diferentes
(atendente.teste, gestor.teste, e as 2 novas) quase ao mesmo tempo. Reduzi temporariamente
`duracaoChamadaPainelSegundos`/`duracaoChamadaPainelFilaSegundos` da grade real (via PATCH)
pra não esperar minutos reais o esvaziamento da fila de exibição — confirmado visualmente no
Chrome a grade 2x2 renderizando exatamente como pedido (2 mais recentes em cima, 2 mais
antigas embaixo, incluindo duas pessoas de teste antigas que "venceram" no critério de
antiguidade sobre 2 das 4 novas, que ficaram só na tabela com o ícone cinza — comportamento
correto do cap por idade). Configuração da grade restaurada ao valor real (30s/15s/3
repetições) depois do teste.

**Erro real cometido e corrigido na hora**: tentei configurar a grade via `curl -d` com
acentos digitados direto no shell — mesma armadilha de encoding já documentada nesta sessão
(rodada da "Tela Gerenciar CRAS") — corrompeu `grades.nome` momentaneamente
("Cadastro �nico"). Corrigido na hora escrevendo o JSON num arquivo (`Write` tool, UTF-8
correto) e usando `curl --data-binary @arquivo` em vez de `-d "string digitada"`. Lição já
estava documentada, mas o hábito de digitar JSON com acento direto no `-d` ainda causou o
mesmo erro — reforçando: **sempre escrever em arquivo quando o payload tiver acento**.

**Pendência de limpeza não resolvida**: as 2 contas de atendente descartáveis (`Teste Layout
C`/`D`) não puderam ser removidas — `usuarios` tem uma chamada registrada em `chamadas`
(FK), que é log append-only de propósito (mesma proteção de integridade já documentada no
`modelo-dados`), então o DELETE foi corretamente recusado pelo Postgres. Ficaram no banco
como contas inativas-na-prática (sem uso real, nome claramente de teste) — se quiser um
banco de teste 100% limpo, precisaria de um endpoint de exclusão de usuário que não existe
ainda (só há "ativo" pra guichê, não pra usuário).

## Vigésima sétima rodada (23/09): blocos do modo "aguardando" viraram cartões brancos independentes

Ajuste rápido em cima da rodada anterior: "não quero eles sendo um bloco dentro do bloco
branco, quero que eles sejam blocos brancos independentes" — os blocos do modo aguardando
viviam como tiles coloridos (`--painel-fundo`) dentro de UM cartão branco único
(`.chamada-atual`). Invertido: `.chamada-atual`, só no modo aguardando, perde fundo/borda/
sombra/padding (vira um contêiner de posicionamento neutro); cada `.aguardando-bloco` ganhou
o mesmo tratamento visual de cartão que `.chamada-atual`/`.historico` sempre tiveram (fundo
`--painel-card` branco, borda, `border-radius: 24px`, `box-shadow: var(--painel-sombra)`) —
agora são cartões de verdade, independentes, direto sobre o fundo tintado, com o `gap` do
grid criando a separação visual entre eles em vez de uma borda de tile dentro de um cartão só.
O modo ativo (chamada única) e o modo descanso (tabela cheia) não mudaram nada.

**Testado ao vivo**: numa aba nova do Chrome (a aba antiga estava sendo usada ativamente pelo
dono do produto em paralelo, testando de verdade — evitei mexer nela, criei outra pra não
atrapalhar), confirmado visualmente 4 cartões brancos com sombra, bem separados, cada um
mostrando nome/protocolo/guichê em coluna. Dado de teste (2 encaixes criados pra esse
teste) resolvido ao final. **Nota**: pra acelerar o teste sem esperar minutos reais, reduzi
temporariamente a duração de exibição da grade real via PATCH — como o dono do produto
estava testando ao vivo na mesma grade nesse momento, isso pode ter alterado brevemente o
timing que ele via na própria tela; revertido ao valor real (30s/15s/3 repetições) assim que
percebido.

## Vigésima oitava rodada (23/09): auditoria de autenticação (bug real corrigido), permissão de encaixe por grade, nome editável, defaults de SLA, central de painéis

Duas mensagens grandes do dono do produto nesta rodada: uma auditoria completa pedindo pra
investigar um bug de "perde a permissão depois de ficar ocioso", e uma lista de ajustes na
configuração de grade + uma feature nova (central de painéis).

### Auditoria de autenticação — causa raiz real encontrada e corrigida

**Sintoma relatado**: na Recepção (e nas 4 telas), depois de ficar ocioso, uma ação passava a
dar "Apenas recepção ou gestor" mesmo estando logado com o papel certo — só logout+login
resolvia.

**Causa raiz, reproduzida de ponta a ponta via curl** (não só hipótese): o cookie de sessão é
por ORIGEM do navegador, não por aba. Cada tela busca `api.sessao()` uma única vez ao montar e
guarda o usuário em estado React pro resto da vida da página, sem nunca revalidar. Se
**qualquer outra aba/janela do mesmo navegador** faz login como outra pessoa (comum ao testar
papéis diferentes — e este projeto faz isso o tempo todo), o cookie da aba antiga é
sobrescrito nos bastidores; a aba antiga continua mostrando a UI de quem estava logado antes,
e só quando uma ação de verdade é enviada é que a identidade REAL da sessão atual aparece,
falhando a checagem de papel daquele endpoint.

**Correção**: interceptor central em `frontend/src/api.ts` — qualquer chamada que receba 401
agora redireciona pra `/login?sessao=expirada` (antes só a busca inicial de sessão de cada
tela fazia isso); qualquer 403 com código de descompasso de identidade
(`papel_sem_acesso`/`unidade_incorreta`/`secretaria_incorreta`/`nao_e_dono`) dispara uma
revalidação (`GET /sessao`) — se a identidade realmente mudou, redireciona igual; se for a
mesma pessoa, é uma restrição de permissão genuína e o erro normal continua aparecendo (nada
muda pra esse caso). `Login.tsx` mostra uma mensagem clara quando chega assim.
`Recepcao.tsx`/`SalaDeEspera.tsx` também ganharam a checagem explícita de papel que
`GestorLayout`/`AdminLayout` já tinham (antes, papel errado nessas duas deixava a tela
travada/vazia em silêncio, sem redirecionar).

**Achados adicionais, sem correção nesta rodada (precisam de decisão de produto)**: admin não
tem caminho de UI pra "Grade de horário"/"Importar" (essas telas só existem em `/gestor/*`,
escopado por UMA secretaria — admin gerencia várias, precisaria de um seletor antes). Ocupação
de guichê revisada e confirmada já robusta ao mesmo cenário (heartbeat falha → guichê libera
sozinho em 15s, nunca fica preso).

### Permissão de encaixe por grade

Gestor agora escolhe, na tela de configuração da grade, se a recepção pode cadastrar encaixe
(walk-in) **naquela grade especificamente** — `grades.permite_encaixe_recepcao` (migration
`000007`, default `true`). **Gestor e admin sempre podem, independente dessa configuração**
(checado em `PostEncaixe`, `internal/handlers/fila.go` — não é um valor lido do banco pra
eles, é bypass direto no código). `GET /unidades/{id}/grades` (usado pelo formulário de
encaixe da recepção) também filtra pra só mostrar grades onde a recepção pode mesmo cadastrar
— evita oferecer uma opção que o POST recusaria de qualquer jeito.

### Nome de exibição da grade voltou a ser editável

Revertendo a rodada 18 (22/09), que tinha removido o campo por achar redundante com o nome
vindo da planilha: o backend (`PatchConfiguracaoGrade`) **nunca deixou de aceitar/exigir**
`nome` no corpo — só a UI parou de expor um campo editável. Agora tem um input "Nome de
exibição" na seção "Recepção" da tela de configurações da grade (mesmo formulário que já
salvava `nome` sem editar, só lendo o valor antigo sem mudança).

### Novos defaults de SLA (só pra grades novas)

`sla_atendimento_minutos` 20→10min, `cooldown_rechamada_minutos` 3→1min — `sla_chegada_minutos`
continua 60min. Só o **default da coluna** mudou (migration `000007`) — grades já configuradas
não foram alteradas retroativamente, isso afeta só criação automática nova (import de
planilha).

### Central de painéis (`/gestor/paineis`, feature nova)

Pedido: como o painel de TV é por GRADE (cada local pode ter várias filas, cada uma com seu
próprio painel), uma secretaria com várias unidades pode ter muitos painéis — uma tela nova
lista todos (com busca por unidade/grade/serviço), e escolher um mostra ele "em foco" dentro
da própria tela via `<iframe src="/painel/{gradeId}">` (zero duplicação da lógica do
painel-tv — a mesma rota pública de sempre). Botão "Tela cheia" chama
`iframeRef.requestFullscreen()` no próprio elemento — cobre a tela física inteira de verdade,
não só o card. Só grades ativas aparecem na lista.

### Import de planilha mista (respondido, não é bug)

Confirmado que já funciona e já foi testado de ponta a ponta (rodada 17): cada linha resolve
`unidade` (via "Local no mapa") e `grade` (via "Serviço" + "Grade de horários") de forma
independente — uma planilha misturando vários locais/serviços é separada automaticamente sem
ação manual. Depois da importação, o gestor aloca atendentes por grade em "Usuários internos".
Reconfirmado via `POST /lotes/preview` com a planilha real do projeto (não gravou nada, é só
preview) — deu 175 erros "protocolo já existe" porque esse arquivo específico já tinha sido
importado antes, confirmando a proteção contra duplicata funcionando, não um problema novo.

**Testado ao vivo (via curl)**: bug de identidade reproduzido de propósito (login A → ação ok
→ login B mesmo cookie → ação de A falha) confirmando a causa raiz; permissão de encaixe
testada nos 3 cenários (recepção bloqueada, gestor permitido mesmo com a flag desligada, grade
some da lista da recepção) — dado de teste revertido ao final. `go build/vet/gofmt` e
`tsc --noEmit`/`npm run build` limpos. **Validação visual no navegador ainda pendente** — a
extensão do Chrome não conectou nesta sessão.

## Vigésima nona rodada (23/09): auditoria de responsividade mobile + cabeçalho em 2 barras + centralização no desktop

Pedido do dono do produto: "muitas coisas quebram [em mobile], dão espaço extra no scroll
lateral" — auditoria completa de responsividade, mais o cabeçalho reorganizado (não queria
mais ele virando coluna em telas estreitas, queria sempre em linha) e o conteúdo do gestor/
admin centralizado em telas largas (estava sempre colado na esquerda).

**Método**: como a extensão do Chrome não permite redimensionar a janela de verdade nesta
máquina (`resize_window` reporta sucesso mas a viewport real não muda — testado e confirmado,
sempre volta pro tamanho da janela maximizada), a auditoria foi feita via um iframe de 390px
injetado na própria página (mesmo origin, então a sessão/cookie funciona normalmente dentro
dele) com um script que varre todos os elementos e reporta quais ultrapassam a viewport **sem
estar contidos por um ancestral com scroll próprio** (`overflow-x: auto/scroll/hidden`) — essa
segunda checagem foi essencial pra não confundir "conteúdo rolável dentro do próprio cartão"
(comportamento correto) com "vazamento de verdade pra fora da página" (bug real).

**Bugs reais encontrados e corrigidos**:
1. **A tabela da Recepção era a única do sistema sem o wrapper `.tabela-cartao`** — sem
   container próprio, ela estourava a largura da PÁGINA inteira em mobile (confirmado:
   612px de conteúdo numa viewport de 387px), exatamente a causa do "espaço extra no scroll
   lateral" relatado. Corrigido envolvendo a tabela em `<div className="tabela-cartao">`
   (`Recepcao.tsx`).
2. **`.tabela-cartao` usava `overflow: hidden`, não `overflow-x: auto`** — isso não causava
   scroll na página (o hidden impedia), mas CORTAVA silenciosamente as colunas que não
   coubessem, tipicamente a coluna de ação com os ícones. Achado num teste real na tela
   "Usuários internos" do admin: a tabela tinha 730px de conteúdo numa viewport de 387px, e a
   coluna de ação inteira (editar usuário) ficava fora da área visível, sem nenhum aviso —
   ou seja, o admin não conseguia editar ninguém pelo celular. Trocado pra `overflow-x: auto`
   em todo o sistema (afeta Recepção, Sala de Espera, Usuários do gestor/admin, Grades,
   Prefeituras, erros de importação — todas as tabelas do sistema usam essa classe).
3. **`<select>` de grade no modal de encaixe da Recepção estourava o modal** — um `<select>`
   nativo se auto-dimensiona pela opção MAIS LARGA, e com dados reais importados (grades tipo
   "CRAS - Jardim Jordão/Guararapes (ENCAIXE)"), isso empurrava o select 418px de largura
   dentro de um modal de ~350px. Corrigido com uma regra global nova em `index.css`:
   `input, select, button { max-width: 100%; min-width: 0; }` — testado depois também no
   formulário "Criar usuário" do gestor (mesma classe de bug, mesma correção resolveu).
4. **Barra de busca da Recepção**: sem `min-width: 0` explícito no campo de busca e
   `flex-shrink: 0` no botão "Adicionar", era o BOTÃO que saía da tela em vez do campo
   encolher (achado residual depois do fix nº1, 26px de overflow).

**Cabeçalho reorganizado em 2 barras** (pedido explícito: "não gosto de header com conteúdo
em coluna, quero em linha ainda"): o `.cabecalho` antigo era um grid de 3 colunas
(`1fr auto 1fr`) que estourava ou empilhava em coluna abaixo de 700px porque a coluna do meio
("auto") crescia pelo conteúdo sem limite. Virou 2 barras, cada uma só com 2 blocos:
- Barra principal (cor de marca): produto+prefeitura à esquerda, ações (guichê/sair) à
  direita.
- Subcabeçalho novo (fundo branco, logo abaixo): papel+nome de quem está logado à esquerda,
  nome da unidade à direita.

Textos longos truncam com reticências (`min-width:0` + `text-overflow:ellipsis`) em vez de
forçar a barra a crescer — é isso que garante uma linha só em qualquer largura, sem precisar
de nenhum breakpoint reorganizando o layout (`componentes/Cabecalho.tsx`, usado por Recepção
e Sala de Espera).

**Conteúdo do gestor/admin centralizado**: `.pagina-gestor` tinha `max-width: 1180px` desde
22/09 mas nunca `margin: 0 auto` — em qualquer monitor mais largo que ~1180px, o conteúdo
ficava sempre colado na esquerda com um vão vazio à direita. Adicionado `margin: 0 auto`
(técnica padrão de centralização com max-width). Não foi possível demonstrar visualmente o
efeito nesta sessão (a janela de teste tem 1568px de largura, e nesse tamanho específico o
espaço disponível já é bem próximo de 1180px, sobrando pouca folga pra centralizar de forma
visível) — confirmado via `getComputedStyle`/medição de retângulo que a regra está correta e
vai centralizar de verdade em monitores mais largos (1920px+, o caso real que motivou o
pedido).

**Rede de segurança adicionada**: `html { overflow-x: hidden }` (par do `overflow-y: scroll`
já existente) — proteção contra qualquer causa de overflow ainda não identificada, não uma
solução em si (documentado na skill pra não virar desculpa de não investigar a causa raiz da
próxima vez).

**Testado em todas as telas principais** (via o iframe de 390px, sessão real): login,
recepção (com busca, abas, modal de encaixe com select de grade real), sala de espera (gate
de guichê, 3 blocos), dashboard/grades/usuários/importar/painéis do gestor, dashboard/
usuários/prefeituras do admin, configuração de grade (guichês expandidos), configuração de
prefeitura, painel de TV público — **zero vazamentos reais restantes** em todas elas.
`tsc --noEmit -p tsconfig.app.json` e `npm run build` limpos.

Skill `identidade-visual` ganhou 3 seções novas documentando essas regras (tabela sempre com
`.tabela-cartao` + `overflow-x:auto`, `input/select/button` sempre `max-width:100%;
min-width:0`, cabeçalho em 2 barras nunca 3 colunas, `.pagina-gestor` sempre com `margin:0
auto`) — pra qualquer tela nova não reintroduzir os mesmos bugs.

## Trigésima rodada (23/09): multi-grade, exclusão, kiosk de TV público, admin em 4 abas por prefeitura

A maior lista de pedidos numa mensagem só até agora — 15 itens distintos, do dono do produto,
misturando bugs pontuais, features novas e duas auditorias explícitas (aninhamento de tenant e
propagação ao vivo de SLA). Tudo implementado e testado nesta rodada; resumo por item:

**1. Scroll duplo/desnecessário**: mitigado com `scrollbar-width: thin` + scrollbar fina do
Chrome em `.tabela-cartao` — a barra clássica do Windows (~17px) aparecia por 1px de
sub-pixel mesmo sem overflow de verdade em alguns casos; a fina nunca mais chama atenção
sozinha. Não foi possível reproduzir um "scroll duplo" literal (2 scrollbars simultâneas) de
verdade nesta sessão — se aparecer de novo, é bom pedir a URL/tela exata pra investigar
melhor.

**2. Botão "copiar link" com estilo secundário**: nova classe `.botao-icone-secundario`
(contorno na cor de ação, sem fundo sólido) — usada nos dois lugares que têm essa ação
(`Grades.tsx`, `Paineis.tsx`). Ações primárias de verdade (confirmar chegada, marcar
presente, engrenagem de configurações) continuam sólidas.

**3. Multi-grade (mudança de schema real)**: `usuarios.grade_id` (FK única) virou uma tabela
N:N, `usuario_grades (usuario_id, grade_id)` — migration `000008`. Atendente exige pelo menos
1 grade; recepcionista pode ter 0 (sem restrição, vê a unidade inteira, como sempre) ou mais
(aí só cadastra encaixe nas grades marcadas). Sessão devolve `grades: [{id,nome}][]` em vez de
`gradeId`/`gradeNome` — o front escolhe em qual trabalhar em runtime, não mais um valor fixo
da sessão.

**4. Abas Ativas/Inativas na Grade de horário** (`Grades.tsx`) — antes tudo numa lista só com
um chip "inativa" solto.

**5. Nome editável na criação/edição de usuário**: já funcionava (confirmado, não era bug) —
tanto no gestor quanto no admin, criação e edição sempre aceitaram/exigiram `nome`.

**6. Ordem da sidebar do gestor**: Dashboard, Usuários internos, Grade de horário, Painéis
(Importar continua separado por respiro, no fim).

**7. Excluir grade/usuário/guichê** — todos soft-delete (nunca hard-delete, os três podem ter
histórico em `chamadas`/`agendamentos`, logs append-only). Grade e guichê ganharam uma coluna
nova `excluido_em` (migration `000008`); usuário reaproveita `usuarios.ativo` (já existia,
já bloqueava login, nunca tinha botão de UI pra isso — não criei uma coluna redundante).
Gestor não pode excluir a própria conta (403).

**8. Grade nasce com 1 guichê padrão + mínimo 1 obrigatório + exclusão de guichê**:
`ResolverOuCriarGrade` agora cria "Guichê 1" automaticamente dentro da mesma transação.
`ExcluirGuiche` bloqueia (409 `ultimo_guiche`) se for o único guichê/sala não-excluído da
grade — checagem e exclusão na MESMA query SQL (atômico, sem condição de corrida).

**9. Copiar predefinições de outra grade + botão de salvar único**: `GradeConfiguracao.tsx`
ganhou um `<select>` no topo ("Copiar predefinições de…") que busca a config de outra grade e
preenche os campos LOCAIS (não salva sozinho — só depois de clicar Salvar). Os 3 blocos
(Recepção/SLA/Painel) perderam cada um seu próprio botão "Salvar" — viraram `<div>`s sem
form — e agora tem UM ícone de salvar no topo (ao lado do select) e UM botão "Salvar
configurações" no final da página, os dois chamando a mesma função.

**10. Acesso aos painéis sem login de gestor (kiosk de TV)**: rota nova, pública, por
UNIDADE: `/tv/:unidadeId` (`telas/painel-tv/PaineisPublico.tsx`) + endpoint público
`GET /unidades/{unidadeId}/paineis-publico` (só id/nome das grades ativas — nenhum dado de
agendamento/cidadão, isso continua só no `GetPainel` de cada grade, já público desde
sempre). A tela "Painéis" do gestor ganhou uma seção "Acesso pela TV do local" com um botão
de copiar esse link por unidade, pra deixar fixo na TV física do CRAS.

**11. Admin reestruturado em 4 abas por prefeitura**: "Dashboard" e "Usuários internos" eram
itens GLOBAIS da sidebar do admin (misturavam dado de todas as prefeituras numa lista só) —
viraram abas DENTRO da configuração de cada prefeitura (`/admin/prefeituras/:id/{dashboard,
usuarios,settings,paineis}`, `telas/admin/prefeitura/PrefeituraLayout.tsx` + 4 telas
filhas). "Settings" é a antiga tela de configuração (nome/cores/secretarias, sem mudança de
conteúdo). "Painéis" é novo — todas as grades ativas de todas as secretarias da prefeitura.
Backend ganhou 3 endpoints espelhando os do gestor mas agregados por PREFEITURA em vez de
secretaria: `GET /prefeituras/{id}/dashboard` (`ResumoDoDiaPrefeitura`, mesma forma do
dashboard do gestor), `GET /prefeituras/{id}/usuarios` (`ListarUsuariosInternosPorPrefeitura`),
`GET /prefeituras/{id}/paineis` (`ListarGradesAtivasPorPrefeitura`, junta unidade→secretaria→
prefeitura). Sidebar do admin ficou só com "Prefeituras".

**12. Bug real corrigido: admin herdava a cor de quem logou por último na aba** — pedido:
"a cor de Jaboatão só interfere apenas Jaboatão, e assim com cada cidade" — mas
`aplicarTemaPrefeitura` grava a cor como estilo INLINE em `:root` e nada revertia isso ao
trocar de contexto numa SPA (login→admin sem reload de página real deixava a cor "grudada").
`tema.ts` ganhou `resetarTema()`, chamado no mount de `Login.tsx` e `AdminLayout.tsx` — as
duas telas que nunca têm (e nunca devem ter) a cor de uma prefeitura específica. Confirmado
ao vivo: sidebar do admin continua na cor neutra da plataforma mesmo dentro da configuração
de uma prefeitura específica (a cor dela aparece só como DADO, no formulário da aba
Settings, nunca pintando o resto da tela).

**13. Auditoria de aninhamento de tenant**: criada uma prefeitura+secretaria+gestor de teste
isolados — confirmado que `/prefeituras/{id}/dashboard|usuarios|paineis` da prefeitura nova
vêm 100% vazios/zerados (nenhum dado de Jaboatão vaza), e que o gestor novo recebe 403 ao
tentar acessar secretaria/grades de Jaboatão. Nada precisou ser corrigido — o isolamento por
IDs (não por nome) já garantia isso desde a rodada do admin (22/09); só reconfirmado com os
endpoints novos desta rodada também. Dado de teste removido ao final.

**14. Etiquetas de grade nas telas de usuário**: `UsuarioInterno`/`UsuarioAdmin` agora trazem
`grades: [{id,nome}][]` em vez de um par `gradeId`/`gradeNome`; as tabelas (gestor e as duas
do admin — a nova por-prefeitura e, indiretamente, qualquer lugar que reusar o tipo) mostram
uma etiqueta arredondada por grade alocada (`.etiqueta-grade`), no lugar de um texto solto.
Formulário de criar/editar usuário também virou multi-seleção (`componentes/
SelecaoGrades.tsx`, checkboxes — melhor UX que um `<select multiple>` nativo).

**15. Confirmado ao vivo: mudança de SLA/cooldown se aplica na hora, sem resetar quem já
estava em atendimento**: testado via curl — chamei alguém com cooldown=3min, anotei
`podeRechamarEm`, mudei o cooldown da grade pra 1min, e o MESMO chamado (mesmo `chamado_em`,
sem rechamar) teve `podeRechamarEm` recalculado imediatamente pro novo valor. Isso já era o
comportamento do código (grade é sempre lida fresca do banco a cada request, nunca cacheada;
o sweeper de ausência automática usa a mesma lógica — `chamado_em` fixo, duração lida em
tempo real) — não precisou de mudança nenhuma, só a verificação pedida.

**Erro real cometido e corrigido na hora**: testando o PATCH de configuração da grade real via
curl, digitei "Único"/"CadÚnico" com acento direto num `-d` inline — mesma armadilha de
encoding já documentada 2x nesta sessão antes — corrompeu `grades.nome` momentaneamente.
Achado na hora (screenshot mostrou "Cad�nico"), corrigido escrevendo o JSON num arquivo e
usando `--data-binary @arquivo`. Terceira vez que esse exato erro acontece nesta sessão —
reforçando de vez: **nunca digitar acento direto num `-d` de curl**, sempre arquivo.

**Migration nova**: `000008_multi_grade_e_exclusao` — tabela `usuario_grades`, backfill a
partir do `usuarios.grade_id` antigo (6 linhas migradas), coluna `grade_id` removida de
`usuarios`; colunas `excluido_em` novas em `grades` e `guiches` (não em `usuarios`, que
reaproveita `ativo`).

**Testado**: `go build/vet/gofmt` e `tsc --noEmit`/`npm run build` limpos o tempo todo;
multi-grade testado via curl (permissão, criação, sessão) E ao vivo no Chrome (gate de
seleção de grade quando há mais de uma, troca de grade+guichê junto); exclusão testada nos 3
tipos (grade, usuário, guichê com a regra do último); kiosk público testado sem cookie
nenhum; admin em 4 abas testado ao vivo com dado real de Jaboatão (571 agendamentos do dia,
usuários reais, painéis reais) E com a prefeitura de teste isolada (tudo vazio, confirmando
isolamento). Dado de teste (prefeitura/secretaria/gestor de isolamento, 2 guichês de teste
numa grade real) revertido/removido ao final.

Skills a atualizar na próxima sessão que mexer nessas áreas: `modelo-dados` (usuario_grades,
excluido_em), `papeis-e-telas` (fluxo de seleção de grade do atendente, estrutura nova do
admin), `regras-negocio-fila` (nada mudou na regra em si, só a fonte da grade ativa).
**Atualizado na rodada seguinte, abaixo.**

## Trigésima primeira rodada (23/09): auditoria completa de multi-grade — bug real, gate removido, agrupamento por local

Auditoria grande pedida pelo dono do produto depois de um bug relatado: "alterei a grade de
um atendente/recepcionista pelo perfil do usuário, mas a alteração não foi refletida
corretamente no perfil dele." Pedido explícito de não presumir que era só frontend, investigar
a cadeia inteira (banco → backend → API → frontend → painel), e aproveitar pra revisar a regra
de negócio de multi-grade por completo, não só o sintoma.

### 1. Causa do bug relatado

**Não era um bug de persistência.** Reproduzido de ponta a ponta via curl (gestor PATCH → SELECT
direto no banco → GET da listagem → login fresco do atendente) — os quatro pontos retornaram o
vínculo atualizado imediatamente e de forma consistente. A causa real era de **regra de
produto, não bug técnico**: a rodada anterior (30, mesma data) tinha acabado de introduzir
`usuario_grades` (N:N) no banco/backend, mas o FRONTEND ainda forçava o atendente a escolher
UMA grade pra trabalhar, num gate obrigatório de tela cheia ("Qual grade você vai atender?").
Alocar uma segunda grade a alguém não tinha efeito visível nenhum, porque a tela continuava só
mostrando/operando a grade escolhida antes — exatamente o sintoma relatado. Corrigido removendo
esse gate por completo (ver item 2).

### 2. Regra de múltiplas grades — como ficou

Confirmado com o dono do produto (exemplo do "Atendente João", com grades em duas unidades
físicas diferentes): **o atendente/recepcionista nunca escolhe uma grade pra trabalhar** — vê e
opera TODAS as que estiver alocado, consolidadas, o tempo todo. Não existe mais nenhuma tela ou
modal de seleção de grade.

- **Backend**: `internal/repository/usuarios.go` — `ListarGradesDoUsuario`/`GradesPorUsuario`
  passaram a devolver `GradeComUnidade` (grade + `unidadeId`/`unidadeNome`), não mais
  `domain.Grade` puro — necessário porque a unidade única do usuário (`usuario.unidadeId`) não
  dá conta de um atendente com grades em unidades diferentes. Endpoints novos, todos derivando
  a lista de grades da PRÓPRIA sessão (nunca recebem IDs do cliente):
  `GET /atendente/grades`, `/atendente/grades/guiches`, `/atendente/grades/sala-espera`,
  `/atendente/grades/chamados`, `/atendente/grades/atendidos-hoje`,
  `/atendente/grades/ausentes-hoje` (`internal/handlers/fila.go`, função `minhasGrades`).
  Repository ganhou variantes `*PorGrades(gradeIDs []string)` (SQL com `= any($1)`) pra cada
  consulta de fila, evitando N requisições separadas — os endpoints antigos `/grades/{gradeId}/
  ...` continuam existindo como wrappers de uma grade só (usados pelas ações que são
  inerentemente de uma grade específica: chamar-próximo, ocupar guichê etc.).
  `GET /sessao` devolve `usuario.grades: [{id,nome,unidadeId,unidadeNome}][]`.
- **Frontend**: `SalaDeEspera.tsx` reescrito — sem gate de grade. "Sala de espera" (aguardando)
  aparece agrupada por unidade → grade, cada grupo com seu próprio guichê (pílula clicável,
  "Selecionar guichê" se ainda não resolvido — não bloqueia as outras grades já resolvidas) e
  seu próprio "Chamar próximo". "Pendentes"/"Chamando agora" e "Meus atendimentos hoje"
  continuam como tabela única consolidada, com uma coluna "Grade" a mais.
- **Regra de bloqueio confirmada ESCOPADA POR GRADE, nunca pelo atendente como um todo**
  (verificado no SQL — `TemChamadaPendenteNaoRechamada`/`ExistemChamadosOrfaos` filtram por
  `grade_id`): uma chamada pendente ou um chamado órfão numa grade nunca bloqueia as outras
  grades do mesmo atendente, mesmo as da mesma unidade física. Reproduzido via curl: atendente
  com pendente na grade A chamou normalmente na grade B, sem esperar.
- **Guichê é por grade, resolvido sob demanda**: ao entrar, o sistema tenta reocupar
  automaticamente o guichê lembrado (`localStorage`, chave por grade) de cada grade; só quem
  não resolver aparece com a pílula pra escolher — nunca trava a tela inteira.
- **Realocação ao vivo, sem logout/reload** (requisito explícito da auditoria): a lista de
  grades é revalidada a cada 8s (`GET /sessao`) enquanto a tela está aberta. Grade nova: ganha
  sua seção e tenta resolver o guichê lembrado dela. Grade removida: some da tela e o guichê
  que estava ocupado nela é liberado no backend automaticamente (efeito de limpeza reagindo à
  mudança do estado `grades`).

### 3. Agrupamento por local no painel

Implementado nos 3 lugares que listam grades como cartões: `telas/gestor/Paineis.tsx`,
`telas/admin/prefeitura/PrefeituraPaineis.tsx` (agrupados por unidade, com cabeçalho do nome da
unidade só quando há mais de uma) e a "Sala de espera" do atendente (mesmo componente de
agrupamento, `agruparPorUnidade`). `telas/painel-tv/PaineisPublico.tsx` (kiosk por unidade,
`/tv/{unidadeId}`) não precisou de mudança — já é inerentemente um grupo só, por definição.

### 4-9. Auditoria de fluxo de dados, CRUD, integridade, concorrência, eficiência

- **Integridade real corrigida**: `ExcluirGrade` (soft-delete, só grava `excluido_em`) não
  limpava `usuario_grades` — o `on delete cascade` do schema só dispara numa exclusão de LINHA
  de verdade, nunca num soft-delete. Corrigido rodando `ExcluirGrade` numa transação que também
  apaga essas linhas explicitamente. Verificado criando uma grade descartável, alocando um
  atendente, excluindo, e confirmando `usuario_grades` zerado e a sessão sem mais aquela grade.
- **Integridade real corrigida**: `PatchUsuario` validava que cada grade pertence à secretaria
  do editor, mas não que ela pertence à UNIDADE fixa de uma recepcionista alvo — um vínculo de
  outra unidade gravava sem erro mas nunca teria efeito (recepcionista só vê grades da própria
  unidade em `GetGradesDaUnidade`). Bloqueado agora com 400 antes de gravar. Atendente não tem
  essa restrição (pode ter grades de unidades diferentes, de propósito).
- **Concorrência revisada, sem mudança necessária**: guichê ocupado por um atendente que perde
  a grade (realocado por um admin) não precisa de limpeza explícita — a ocupação já é lida com
  uma janela de frescor de 15s (`case when ocupado_em > now() - janela then ... end`), então
  qualquer inconsistência visual (guichê aparecendo ocupado por alguém que já não tem mais
  acesso) se autocorrige sozinha dentro desse limite, sem precisar de mais código (mesmo
  desenho já usado pra aba fechada/crash, documentado desde 22/09).
- **Eficiência**: endpoints consolidados evitam N requisições por grade (uma soma via `= any()`
  em vez de uma consulta por grade); nenhuma otimização especulativa adicionada além disso.
- Dado de teste real usado em todos os testes (Atendente Teste com 2→3→2 grades, uma delas
  cross-unidade temporária; um atendente/recepcionista descartáveis criados e desativados ao
  final; uma grade descartável criada e excluída) — tudo revertido ao estado original.
  **Achado incidental, não um bug desta rodada**: duas grades legadas (`Cadastro Único Sede
  Prazeres (ENCAIXE)` e `CRAS - Cavaleiro`) tinham ZERO guichês cadastrados — dado anterior à
  regra "grade nasce com 1 guichê padrão" da rodada 30. Criado um guichê pra a primeira (usada
  no teste); a segunda não foi tocada, fica como nota pra quando alguém for usar aquela grade
  de verdade.

### Validação

`go build/vet/gofmt` e `npx tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos o tempo
todo. Testado via curl de ponta a ponta (não só visualmente): usuário de grade única (link →
alterar → remover → sessão reflete cada etapa), usuário multi-grade mesma unidade (as 2 grades
reais de `atendente.teste`), usuário multi-grade unidades DIFERENTES (grade temporária de CRAS
- Cavaleiro), bloqueio por grade independente entre 2 chamadas simultâneas em grades distintas,
exclusão de grade com limpeza de vínculo, e validação de recepcionista fora da própria unidade.
**Não testado ao vivo no navegador Chrome nesta rodada** (extensão não usada) — a lógica do
agrupamento visual e do polling de 8s foi verificada por leitura de código + build limpo, mas
não por clique real; recomendado confirmar visualmente antes de considerar 100% fechado.

Skills atualizadas: `modelo-dados` (armadilha do cascade em soft-delete, validação de
recepcionista), `papeis-e-telas` (seção Atendente reescrita, seção Painéis nova, Usuários
internos com multi-seleção), `regras-negocio-fila` (escopo por grade confirmado com
multi-grade).

## Trigésima segunda rodada (23/09): reativação, CORS restritivo, testes automatizados, validação ao vivo no Chrome

O dono do produto pediu explicitamente pra resolver a lista de pontos em aberto sinalizados no
fim da rodada anterior (validação visual pendente, sem caminho de reativar exclusão,
zero testes automatizados, CORS permissivo, specs desatualizados).

### 1. Reativação de usuário/grade/guichê (lacuna real fechada)

Até esta rodada, excluir um usuário (`ativo=false`), uma grade ou um guichê (`excluido_em`)
não tinha NENHUM caminho de volta pela UI — o registro simplesmente sumia de toda listagem
(`ListarUsuariosPorSecretaria`/`ListarTodosUsuariosInternos` filtravam `ativo=true`;
`ListarGradesPorSecretaria`/`ListarTodosGuiches` filtravam `excluido_em is null`), só dava pra
desfazer mexendo direto no banco. Corrigido:

- **Backend**: `Repo.ReativarUsuario`/`ReativarGrade`/`ReativarGuiche` (novos) +
  `POST /usuarios/{id}/reativar`, `POST /grades/{id}/reativar`,
  `POST /grades/{id}/guiches/{id}/reativar` (gestor/admin, mesma checagem de acesso das rotas
  de exclusão). As 4 listagens acima passaram a incluir os registros inativos/excluídos —
  `domain.Grade`/`domain.Guiche` ganharam um campo `Excluida bool` (calculado via
  `excluido_em is not null` na query, nunca no Go) pra o frontend saber diferenciar. Importante:
  **todo outro consumidor de grade/guichê continua filtrando excluído** (recepção, painel,
  importação, `ResolverOuCriarGrade`) — só as duas listagens de gestão precisavam mudar.
- **Reativar grade não recupera os vínculos de `usuario_grades`** (esses já foram limpos na
  hora da exclusão, ver rodada anterior) — o gestor realoca quem precisar, igual faria numa
  grade nova. Reativar usuário/guichê não tem essa pegadinha (nada foi limpo neles).
- **Frontend**: `Usuarios.tsx` (gestor) e `PrefeituraUsuarios.tsx` (admin) ganharam abas
  Ativos/Inativos com botão reativar (ícone `RotateCcw`); `Grades.tsx` ganhou uma terceira aba
  "Excluídas"; `GradeConfiguracao.tsx` mostra guichês excluídos com um resumo mínimo + botão
  reativar, em vez do formulário de edição completo. Efeito colateral corrigido: o toggle de
  `ativo` (que já existia, sempre teve volta) usava a palavra "Reativar" também — renomeado pra
  "Ativar" nos dois lugares (guichê e grade) pra não confundir com a exclusão de verdade.
- **Todos os 3 fluxos testados de ponta a ponta via curl** (excluir → confirma que continua
  aparecendo com o status certo → reativa → confirma volta ao normal) **e confirmados ao vivo
  no Chrome**: a aba "Inativos" de Usuários internos já continha 5 contas de teste
  historicamente "perdidas" de rodadas anteriores (documentadas como "pendência de limpeza não
  resolvida") — reativei uma, confirmei que voltou pra "Ativos", desativei de novo pra manter o
  estado original. Mesma confirmação visual na aba "Excluídas" de Grade de horário (mostrou a
  grade "Geral", excluída desde a rodada 30, com botão de reativar funcionando).

### 2. CORS deixou de refletir qualquer origem

`corsSimples` (desde 17/09) ecoava de volta QUALQUER `Origin` recebido junto com
`Access-Control-Allow-Credentials: true` — a combinação clássica que expõe a sessão (cookie
HttpOnly) a um site malicioso que faça uma requisição contra a API a partir do navegador de um
usuário logado. Substituído por `cors(appEnv, origensPermitidas)`
(`cmd/api/main.go`): em desenvolvimento (`APP_ENV=development`, o default), só aceita
`http://(localhost|127.0.0.1|<ip>):(5173|4173)` — continua funcionando por IP da rede sem
reconfigurar nada (mesmo motivo do fix de 20/09), mas nunca um domínio arbitrário; fora de
desenvolvimento, só as origens explicitamente listadas em `CORS_ALLOWED_ORIGINS` (env var nova,
separada por vírgula) — vazio por padrão, ou seja, **nenhuma origem passa até alguém configurar
isso no deploy real** (falha segura). Testado via curl: origem local permitida recebe os
headers de CORS normalmente; `http://evil.com` não recebe nenhum header (o navegador do
cliente bloqueia sozinho); IP da rede na porta do Vite funciona igual antes.

### 3. Testes automatizados — zero para uma suíte real cobrindo a lógica mais frágil

`backend/internal/repository/fila_regras_test.go` (novo, 7 testes) — testes de integração de
verdade contra o Postgres de dev (mesma filosofia de "testar com dado real" já seguida
manualmente a sessão inteira via curl/navegador, sem mocks), cada um criando sua própria
prefeitura/secretaria/unidade/grade/usuários descartáveis (prefixo `TESTE_AUTOMATIZADO_`) e
limpando tudo via `t.Cleanup` — confirmado zero resíduo no banco depois de rodar. Cobre
exatamente as áreas sinalizadas como sem cobertura nenhuma:
- Prioridade na sala de espera (no_horário sempre primeiro, atrasado/encaixe competem só por
  ordem de chegada).
- Cooldown bloqueia chamar alguém novo até expirar (não até "já rechamou uma vez" — regra
  corrigida na rodada de 23/09 anterior a esta).
- Chamado órfão bloqueia a grade inteira pra outros atendentes até ser assumido.
- Ausência automática nos dois estágios (chegada e atendimento) via `SweepAusenciaAutomatica`.
- Independência entre grades (chamada pendente numa grade não bloqueia outra do mesmo
  atendente) — regressão direta da auditoria de multi-grade desta sessão.
- `ExcluirGrade` limpa `usuario_grades` (regressão do fix da rodada anterior).
- Os 3 fluxos de reativação (item 1 acima).

Roda com `go test ./internal/repository/...` (precisa do Postgres do `docker-compose` no ar);
se o banco não responder, os testes pulam (`t.Skip`) em vez de falhar, pra não quebrar quem
rodar sem o ambiente de dev configurado. Não é cobertura completa do sistema — é a suíte que
protege especificamente a lógica que já foi confundida/relatada como bug pelo menos duas vezes
nesta sessão (cooldown/órfão).

### 4. Validação ao vivo no Chrome (pendência da rodada anterior, fechada)

Logado como `atendente.teste` (2 grades, mesma unidade) — confirmado visualmente: **sem
nenhum gate de seleção de grade**, as duas grades aparecem lado a lado na "Sala de espera" já
prontas pra trabalhar; clicar em "Selecionar guichê" abre um modal escopado só àquela grade;
escolher um guichê numa grade não afeta a outra (a segunda continuou mostrando "Selecionar
guichê" normalmente). Criado um encaixe de teste real via curl, chamado pelo botão da própria
tela — apareceu em "Pendentes" com prazo/cooldown corretos, a mensagem de bloqueio
("Você tem uma chamada pendente...") apareceu SÓ na grade onde a chamada estava pendente, a
outra grade continuou com "Chamar próximo" habilitado normalmente. Marcado como presente pelo
botão real — moveu pra "Meus atendimentos hoje" com a coluna "Grade" certa. Dado de teste
resolvido ao final, guichê liberado.

**Achado incidental da automação, não um bug do produto**: um clique em "Excluir" (que dispara
`window.confirm(...)`) travou a aba do navegador automatizado — dialogs nativos do navegador
bloqueiam a comunicação com a ferramenta de automação (comportamento documentado da própria
ferramenta, não um bug do app). Recuperado fechando e reabrindo a aba; a ação em si nunca
chegou a disparar (confirmado via API que o usuário continuava ativo). Não mexi no padrão
`window.confirm` do app — é usado consistentemente em todas as exclusões e funciona bem pra um
humano de verdade, só atrapalha automação de clique.

### 5. Specs desatualizados — sinalizados, não reescritos

Os 3 specs formais (`specs/001-fila-atendimento-presencial`, `002-importacao-multi-local`,
`003-painel-tv`) foram escritos em 19/09 e nunca revisados — hoje descrevem um sistema sem a
entidade Grade, sem multi-grade, sem a hierarquia Admin/Prefeitura. Decidido NÃO reescrever os
3 specs formais nesta rodada (seria um trabalho grande e separado, de escrever spec, não de
corrigir bug) — em vez disso, adicionado um aviso no topo de cada um apontando pra `CLAUDE.md`
e a skill relevante como fonte de verdade atual, mesmo padrão já usado pra sinalizar que o
Notion pode estar desatualizado. `speckit-converge` (a skill de spec-kit mais parecida com
"sincronizar spec e código") foi avaliada e descartada pra esse fim — ela assume que o SPEC é
a verdade e só lista trabalho que falta implementar; aqui o problema é o oposto (código já
implementou muito mais do que o spec descreve).

### Validação final

`go build/vet/gofmt`, `go test ./internal/repository/...` (7/7 passando) e
`npx tsc --noEmit -p tsconfig.app.json`/`npm run build` limpos. Testado ao vivo no Chrome (item
4). Nenhum dado de teste ficou pendente no banco.

## Trigésima terceira rodada (24/09): backend reorganizado pra MVC

Pedido direto: "altere a arquitetura do sistema para o padrao mvc e dps valide cada ponto para
ver se o fluxo está funcionando". Como o backend é uma API só-JSON (sem views server-side), o
significado de "MVC" foi confirmado antes de mexer em ~130 arquivos (`AskUserQuestion`) — opção
escolhida: separar de verdade a camada de apresentação das funções que já existiam espalhadas
dentro dos handlers, sem inventar uma camada de Service nova (a constituição do projeto já
diz "sem camadas genéricas... resolver o problema real do jeito mais direto").

**Model** (sem mudança de comportamento, só de nome onde fazia sentido): continua
`internal/domain` (entidades) + `internal/repository` (toda query SQL, incluindo a regra de
negócio pesada — cooldown/órfão/prioridade/sweeper).

**View, novo** (`internal/views/`, ~10 arquivos): as funções que antes viviam soltas dentro dos
handlers e montavam a resposta (`gradeParaJSON`, `agendamentoParaJSON`,
`usuarioInternoParaJSON`, `chamadoCompartilhadoParaJSON` etc. — 12 no total) foram extraídas,
exportadas, e viraram funções PURAS (recebem dado já resolvido, nunca tocam o banco). O caso
mais complexo foi `GetPainel` (o handler do painel de TV público) — antes montava uns 90 linhas
de `map[string]any` misturadas com a orquestração de busca; virou `views.PainelResponse(...)`,
recebendo grade/unidade/chamada-atual/aguardando/últimas-chamadas/cores já resolvidos.
`responderJSON`/`responderErro` (o ponto que efetivamente escreve no `http.ResponseWriter`)
também migraram pra cá, exportadas (`views.ResponderJSON`/`ResponderErro`) — são literalmente
a operação de "renderizar" a resposta.

**Controller** (`internal/controllers/`, era `internal/handlers/`): renomeado o pacote, a
struct (`Handlers` → `Controllers`) e o receptor (`h` → `c`) em todo lugar. Cada controller
agora só orquestra — resolve permissão, busca no repository, entrega o resultado pronto pra
uma função de `views` decidir o formato; nunca monta `map[string]any` de resposta direto.

**Execução**: renomes mecânicos (pacote, imports, receptor) feitos via `sed` no lote inteiro —
mais rápido e mais seguro que editar arquivo por arquivo pra uma mudança puramente textual; as
extrações de função (mover corpo pra `views`, trocar chamada) feitas uma a uma, com leitura do
código real antes de cada uma. **Bug real introduzido pelo sed e corrigido na hora**: o
`h.` → `c.` global colidiu com 3 lugares em `fila.go` (`PostChamarProximo`, `PostRechamar`,
`PostAssumirChamada`) onde uma variável local já se chamava `c` (de "chamada",
`domain.Chamada`) — o Go recusou compilar ("no new variables on left side of :="), pego na
hora pelo `go build` e corrigido renomeando a variável local pra `chamada`. Não foi um erro
silencioso: o compilador Go pegou os 3 casos de uma vez, nenhum passou despercebido.

**Validação de ponta a ponta, item por item, contra o backend já reorganizado** (não só
build/vet limpos — dado real, como de costume):
- Login dos 4 papéis (`views.UsuarioParaJSON`) — shape idêntico ao de antes do refactor.
- `GET /sessao`, recepção, dashboard do gestor, grade de horário (`views.GradesParaJSON`, campo
  `excluida` presente).
- Endpoints consolidados `/atendente/grades*` (multi-grade).
- Ciclo completo de chamada: criar encaixe → ocupar guichê → chamar próximo
  (`views.AgendamentoParaJSON`, a variável `chamada` renomeada) → `GetChamados`
  (`views.ChamadoCompartilhadoParaJSON`, `podeRechamarEm`/`prazoAusenciaEm` calculados certo) →
  rechamar bloqueado por cooldown (409) → marcar atendido → liberar guichê.
- Painel público (`views.PainelResponse`, a extração mais arriscada) e central de painéis por
  unidade (`views.PaineisPublicoDaUnidadeResponse`).
- Admin: prefeituras (`views.PrefeituraParaJSON`) e usuários internos
  (`views.UsuarioComContextoParaJSON`).
- CORS e reativação de grade (funcionalidades da rodada anterior) — confirmadas intactas.
- **Ao vivo no Chrome**: dashboard do gestor, "Grade de horário" com as 3 abas, e a tela do
  atendente inteira (multi-grade consolidada, "Meus atendimentos hoje" com a linha de teste
  recém-criada) e o painel de TV público (tabela de últimas chamadas completa, ícones de
  status corretos) — todos renderizando exatamente como antes do refactor, sem nenhuma
  diferença visual ou funcional.

**Escopo explicitamente fora desta rodada**: nenhuma regra de negócio foi alterada — é
reorganização pura de onde o código mora, não uma reescrita de comportamento. Os inline
`responderJSON(w, status, map[string]any{"ok": true})` simples (ex: confirmações de
exclusão/reativação) NÃO foram extraídos pra `views` como funções nomeadas — só a família
`xParaJSON`/`xResponse` de serialização de entidade, que é o que de fato caracteriza uma
camada de View (extrair cada literal JSON de uma linha só seria reescrever cada handler sem
ganho real). Skill `stack-go-react-postgres` atualizada com a estrutura de pastas nova e a
regra prática de "onde vai cada coisa".

## Decisões em aberto (consolidado — atualizado 19/09)

Lista única de tudo que ainda falta decidir/confirmar antes do MVP estar completo. A maioria das
pendências levantadas ao longo do dia 19/09 foi resolvida no mesmo dia — o que sobra é bem menor
agora:

**Fora do nosso controle, adiado pro pós-MVP (não bloqueia o piloto):**
- SSO real (JWT/`iss`/`aud`/mapeamento de `roles`) — decisão explícita de não usar no MVP, ver
  "Autenticação" e skill `auth-conecta-cidades`. Só relevante quando a integração real for
  retomada, depois do piloto.
- Endpoint interno/admin de listagem em massa de agendamentos — issue aberta no GitLab (17/09),
  resposta "não temos atualmente", aguardando análise de viabilidade (skill `api-conecta-cidades`).
  Até lá, upload manual de planilha é o caminho definitivo, não provisório.

**Escopo explicitamente adiado pelo dono do produto, com plano de quando resolver:**
- Gestão de atendentes (alocar/trocar unidade, restringir acesso, trocar papel
  atendente↔recepcionista) — **será resolvido durante o MVP, junto com o gestor** (confirmado
  19/09). Sem spec, sem skill dedicada ainda. O modelo de dados já suporta tecnicamente (ver
  `modelo-dados`), mas não implementar tela/endpoint até a regra vir.

**Simplificações assumidas pro MVP, sinalizadas pra revisão futura (não são incertezas, são
escopo reduzido de propósito):**
- `Grade de horários` ignorada pro roteamento, só `Local no mapa` importa — confirmado
  explicitamente como decisão só-do-MVP ("depois isso vai escalar"), revisar quando existirem
  várias grades reais dentro do mesmo local (skill `importacao-planilha`).
- Reagendamento disfarçado de cancelamento na planilha (texto "Reagendado: ...") — entra igual a
  cancelamento por enquanto, decisão de não tratar diferente ainda ("deixa pra discutir"), sem
  prazo definido pra retomar (skill `importacao-planilha`).

**Não é decisão de produto, é dívida de implementação:**
- Migration Postgres desatualizada em 3 rodadas de mudança — ver item 5 de "Próximos passos
  sugeridos". Mais fácil reescrever do zero a partir do `modelo-dados` atual.
- Nenhuma linha de código dos endpoints (login, recepção, atendente, gestor, painel de TV) ainda
  — só specs e skills prontos.

**Resolvido em 19/09, não é mais pendência**: mapeamento `roles`→papel (irrelevante agora que o
SSO real foi adiado), máscara de CPF/telefone (removida — atendentes já veem esse dado na outra
plataforma).
