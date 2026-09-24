---
name: auth-login-mvp
description: Use ao implementar o login do MVP (email + senha, sem integração real com o SSO da plataforma ainda). Decisão do dono do produto (19/09) — pro piloto, login simples é suficiente; o fluxo real de SSO (skill `auth-conecta-cidades`) fica pra depois do MVP.
---

# Login do MVP — email + senha, sem SSO real ainda

**Decisão (19/09)**: o piloto **não** vai integrar o JWT real da plataforma Conecta Cidades por
enquanto — "essas questões do JWT ainda serão discutidas, mas pro MVP não vamos usar, vamos
passar com login mesmo". O desenho completo do SSO real continua documentado na skill
`auth-conecta-cidades` (é trabalho de pesquisa válido, só adiado) — retomar quando a integração
formal com o portal da prefeitura for negociada.

Pro MVP, login é convencional: usuário e senha, verificados contra o banco, trocados por um
cookie de sessão próprio — **mesmo mecanismo de sessão** já desenhado em `auth-conecta-cidades`
(seção 5: cookie `HttpOnly`, `Secure` fora de dev, `SameSite=Lax`; papel e situação do usuário
relidos do banco a cada requisição, não gravados no token; sem tabela de sessão no servidor). A
única coisa que muda é **como a sessão nasce** — login direto em vez de troca de token vindo de
outro sistema.

## Schema

`usuarios` precisa de mais um campo pra isso funcionar (ver skill `modelo-dados`):

```sql
usuarios (
  ...,
  senha_hash text nullable  -- null enquanto o usuário não tem senha definida; hash (ex: bcrypt), nunca texto puro
)
```

`external_sub` (pensado pro `sub` do JWT real) fica null durante todo o MVP — não é usado
enquanto o SSO real não for implementado.

## Rotas

```
POST /api/v1/login
body: { "email": "...", "senha": "..." }
resposta: Set-Cookie da sessão (mesmo formato de `auth-conecta-cidades`, seção 5) — 401 se
          credencial inválida, sem detalhar se foi o email ou a senha que errou.

GET /api/v1/sessao       -- diz quem está do outro lado do cookie (mesmo contrato do SSO real)
DELETE /api/v1/sessao    -- remove o cookie
```

## Cadastro de usuário e senha — resolvido (22/09)

**Não existe cadastro nenhum pra quem usa o sistema** — pedido explícito do dono do produto:
"não tem como se cadastrar, login criado apenas pela gestão". A senha inicial de todo mundo
(gestor, atendente, recepcionista) é definida na hora da criação, pelo Admin (gestor e
funcionário) — ver skill `papeis-e-telas`, seção Admin, e `internal/handlers/admin.go`
(`PostGestor`/`PostFuncionario`). Não existe um "definir senha no primeiro acesso" nem
"esqueci minha senha" — trocar a senha de alguém é uma edição feita pelo gestor (tela
"Usuários internos", campo opcional) ou pelo admin.

## Tela de login — real, sem atalho de acesso rápido (22/09)

A tela de login (`frontend/src/telas/login/Login.tsx`) é email+senha de verdade, sem link de
cadastro nenhum — reflete a decisão acima. Isso **reverte** uma decisão anterior (20/09,
"esse login não vai ser usado", quando a tela tinha virado 4 botões de acesso rápido por
papel): agora que o Admin cria contas reais de verdade, o login real voltou a fazer sentido,
e os botões de atalho foram removidos. Redireciona por papel depois de logar
(`admin→/admin`, `gestor→/gestor`, `recepcionista→/recepcao`, `atendente→/sala-espera`). O
link de atalho pro Painel de TV também saiu da tela de login — o link público de cada grade
agora se copia direto na tela "Grade de horário" do gestor (botão de copiar link).

## Usuários fictícios para teste

- `admin.teste@local.dev` — papel `admin` (criado direto na migration `000005_admin_
  hierarquia.up.sql`, sem vínculo de unidade/secretaria — acesso master, ver skill
  `modelo-dados`)
- `gestor.teste@local.dev` — papel `gestor` (ver `/seed/usuarios_teste.sql`)
- `atendente.teste@local.dev` — papel `atendente` (ver `/seed/usuarios_teste.sql`)
- `recepcao.teste@local.dev` — papel `recepcionista` (ver `/seed/usuarios_teste.sql`)

Senha de todos: `teste123`. Os três primeiros (não-admin) estão vinculados à unidade
"Cadastro Único Sede Prazeres" criada no seed.

## Quando o SSO real (skill `auth-conecta-cidades`) for retomado

1. Adicionar a troca `?token=` → sessão como um segundo caminho de login, mantendo o login por
   senha funcionando em paralelo (ou substituindo, a decidir na hora).
2. Confirmar `iss`/`aud`/mapeamento de `roles` com a equipe da plataforma (pendências já
   registradas em `auth-conecta-cidades`).
3. Popular `external_sub` a partir do `sub` do JWT real pros usuários que migrarem pro SSO.
