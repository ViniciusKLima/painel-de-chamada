---
name: stack-go-react-postgres
description: Use ao escrever código de backend (Go), frontend (React) ou migrations (Postgres) neste projeto. Define bibliotecas escolhidas e convenções — não introduzir framework/lib alternativa sem justificar.
---

# Stack e convenções

Projeto tocado por uma pessoa só (sem time grande ainda), então prioriza bibliotecas padrão da linguagem e pouca "mágica" — quanto menos abstração escondida, mais fácil de debugar sozinho.

## Backend: Go

- **Router**: `chi` (`github.com/go-chi/chi/v5`) — leve, idiomático, sem excesso de convenção.
- **Acesso a banco**: `pgx` (`github.com/jackc/pgx/v5`) direto, ou `sqlc` se o projeto crescer e valer a pena gerar código a partir de SQL. Não usar ORM pesado tipo GORM — a hierarquia de tabelas é simples o suficiente pra SQL direto ser mais previsível.
- **Migrations**: `golang-migrate` (`github.com/golang-migrate/migrate`) — arquivos `.sql` versionados em `/migrations`.
- **Auth**: MVP usa login simples (email+senha, ver skill `auth-login-mvp`) — `golang.org/x/crypto/bcrypt` pro hash de senha. `github.com/golang-jwt/jwt/v5` + JWKS fica reservado pro SSO real (skill `auth-conecta-cidades`), adiado pro pós-MVP.
- **Estrutura de pastas — MVC (reorganizado 24/09)**:
  ```
  /cmd/api/main.go
  /internal/controllers/   -- o "C": um arquivo por recurso (agendamentos, usuarios, dashboard),
                               struct `Controllers`. Só orquestra: parse do request, checagem de
                               acesso, chama repository, entrega o resultado já resolvido pra
                               uma função de internal/views/ montar a resposta. Nunca monta
                               map[string]any de resposta direto (isso é responsabilidade da view).
  /internal/views/         -- o "V": funções puras `XxxParaJSON`/`XxxResponse` que decidem o
                               formato de uma resposta a partir de dado já resolvido — nunca
                               tocam o banco. Também é dono de `ResponderJSON`/`ResponderErro`
                               (o ponto único que escreve no `http.ResponseWriter`).
  /internal/domain/        -- o "M" (parte 1): structs e enums das entidades, sem I/O.
  /internal/repository/    -- o "M" (parte 2): toda query SQL via pgx puro — é aqui que a regra
                               de negócio "pesada" (cooldown, órfão, prioridade, sweeper) mora,
                               ver skill regras-negocio-fila.
  /internal/auth/          -- middleware de sessão + regras centrais de permissão
                               (internal/auth/permissoes.go).
  /internal/painel/        -- estado em memória da fila de exibição do painel de TV (não é
                               Model de banco, é estado de aplicação — ver skill painel-tv).
  /migrations/
  ```
  Regra prática de "onde vai cada coisa nova": se a função só decide COMO uma entidade já
  resolvida vira JSON (sem consulta própria), vai em `views`; se ela BUSCA/decide COM QUEM
  fazer a operação (autorização, parse de request, orquestração de múltiplas chamadas ao
  repository), vai em `controllers`; se ela é regra de negócio sobre o dado em si (uma query,
  uma transação, um cálculo de prioridade), vai em `repository`.
- **Testes**: pacote `testing` padrão, de integração de verdade contra o Postgres de dev (sem
  mocks — mesma filosofia de "testar com dado real" do resto do projeto), ver
  `internal/repository/fila_regras_test.go`. Cobre cooldown/órfão/prioridade/sweeper — a lógica
  mais sensível do sistema, e o ponto certo pra testar já que é onde ela mora (ver acima).

## Frontend: React + Vite + TypeScript

- Mantém a decisão anterior: Vite (não Next.js — painel interno, sem necessidade de SSR).
- Uma pasta por papel: `/src/telas/gestor`, `/src/telas/recepcionista`, `/src/telas/atendente` — evita uma tela genérica tentando servir os três casos de uso com `if` espalhado.
- Estado de servidor via `@tanstack/react-query` — a sala de espera do atendente e a lista do gestor vão precisar de refetch periódico (polling), e react-query já resolve isso com `refetchInterval` sem reinventar.
- Painel de TV pode continuar como uma tela separada, sem autenticação, só consumindo o endpoint público de "últimos chamados".

## Postgres

- Sempre `uuid` como chave primária (`gen_random_uuid()`, extensão `pgcrypto`).
- Timestamps sempre `timestamptz`, nunca `timestamp` sem timezone.
- Ver skill `modelo-dados` para o schema completo.

## O que NÃO usar (decisão já tomada, não reabrir sem motivo forte)

- Sem GraphQL — REST simples é suficiente pro tamanho do domínio.
- Sem microserviços — é um monólito Go pequeno, o domínio não justifica separação de serviços ainda.
- Sem Redis por enquanto — fila em tempo real via polling do frontend (ver decisão de hospedagem gratuita no CLAUDE.md raiz).
