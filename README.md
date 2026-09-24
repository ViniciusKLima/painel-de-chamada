# Painel de Chamada — Jaboatão mais fácil

Ver `CLAUDE.md` pro contexto completo do projeto e `.claude/skills/` pras decisões detalhadas
de stack, modelo de dados, papéis e regras de negócio. **Stack atual: Go (backend) + React/Vite
(frontend) + Postgres** — backend Go ainda não escrito, é o próximo passo.

## Passo 1: subir banco de dados local

Precisa do Docker Desktop aberto e rodando antes.

```bash
docker compose up -d
```

Isso sobe Postgres e Redis localmente. **Atenção**: o Postgres expõe a porta **5433** (não 5432)
— a 5432 padrão colide com um PostgreSQL nativo já instalado nesta máquina de dev.

## Passo 2: rodar as migrations e o seed

Ainda não existem migrations Go (`golang-migrate`, ver skill `stack-go-react-postgres`). O seed
de dados de teste (unidade CRAS Piloto + 3 usuários fictícios) está em `/seed/usuarios_teste.sql`
e pode ser aplicado direto via `psql` assim que as tabelas existirem.

## Implementação anterior (Node/Fastify/Prisma) — descontinuada, mantida como referência

As pastas `backend/` e `frontend/` na raiz contêm uma implementação anterior, funcional e
testada com dado real, mas **descontinuada** depois da decisão de trocar o backend pra Go (ver
`CLAUDE.md`). Não é o ponto de partida do novo backend. Pra rodar essa versão antiga só de
consulta:

```bash
cd backend
cp .env.example .env   # já aponta pra porta 5433
npm install
npx prisma migrate dev
npm run dev
curl http://localhost:3333/health
```
