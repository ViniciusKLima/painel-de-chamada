---
name: api-conecta-cidades
description: Use ao integrar com a API externa da plataforma "Conecta Cidades" (mesma plataforma por trás do "Jaboatão mais fácil"/"Jornada" — o nome comercial muda por prefeitura) pra consultar, criar, reagendar ou cancelar agendamentos, ou consultar locais/dependentes. Ler ANTES de assumir que essa API resolve o pull automático da agenda do dia — ela tem uma limitação estrutural importante (ver seção "Limitação crítica").
---

# API externa "Conecta Cidades" — Agendamentos

Documentação obtida em 17/09 a partir de `https://rede.ipojuca.pe.gov.br/admin/api-reference`
(instância de **Ipojuca-PE**, não de Jaboatão — é a mesma plataforma white-label, mas o host real
de Jaboatão ainda não foi confirmado; ver "Pendências" no fim). O spec OpenAPI completo (todos
os recursos, não só agendamento) está salvo em `_docs-externas/conecta-cidades-openapi.json` —
consultar esse arquivo pra qualquer recurso fora de agendamento/local/dependente.

## ⚠️ Limitação crítica — leia antes de planejar a integração

Essa API é **desenhada pra agir em nome de UM cidadão por vez** (bots de WhatsApp, atendimento
via IA) — **toda** rota de agendamento exige `citizen_cpf` como parâmetro obrigatório. **Não
existe** um endpoint tipo "listar todos os agendamentos de hoje num local/unidade" que não exija
já saber o CPF de cada cidadão individualmente.

Ou seja: **essa API não substitui o upload manual de planilha** pro caso de uso do painel
("receber os agendamentos automaticamente pra alimentar a fila") — pra isso seria preciso ou
(a) a prefeitura/Conecta Cidades expor um endpoint administrativo/bulk separado (não documentado
aqui, pode existir numa "API Interna" distinta desta "API Externa"), ou (b) continuar com
export manual (Excel/CSV) por enquanto.

**Importante**: o filtro por local/setor é responsabilidade do **nosso** sistema, não da API
externa nem de uma futura API interna — o que falta é só a listagem em massa (do dia, ou geral),
sem exigir CPF por agendamento; filtrar/direcionar por local é trabalho nosso em cima do que
vier.

**Status (17/09)**: issue aberta no GitLab da plataforma perguntando sobre isso. Resposta
inicial do Mateus (contato da plataforma): "não temos atualmente", vai analisar viabilidade.
Até sair resposta, upload manual continua sendo o caminho.

O que essa API **serve bem** pro projeto, mesmo sem resolver o pull em massa:
- Buscar/confirmar um agendamento específico por protocolo (`GET .../by_protocol/{protocolo}`) — útil se a recepção/atendente digitar o protocolo pra conferir dados.
- Cancelar ou reagendar um agendamento específico a partir do painel (se algum papel precisar disso).
- Consultar dependentes ativos de um cidadão.

## Autenticação

Bearer token no header `Authorization: Bearer SEU_TOKEN`. Token emitido pelo administrador da
plataforma, com **permissões granulares por recurso** (ex: `dependents_api`, `scheduling_api`)
— um token pode não ter acesso a todos os endpoints.

Base URL: `https://{host}/api/ext/v1` — `{host}` é o domínio da prefeitura (ex:
`rede.ipojuca.pe.gov.br` nesse exemplo; pra Jaboatão será outro host, ainda não confirmado).

## Rate limiting

Todo endpoint tem limite **por token**, em duas janelas simultâneas:
- **Por minuto** (burst) — headers `X-RateLimit-Limit`, `X-RateLimit-Remaining`, `X-RateLimit-Reset` (epoch).
- **Por dia** (volume) — mesmos nomes com sufixo `-Day`. Zera às 00:00 UTC (21h em Brasília, **não** meia-noite local).
- Sem token válido: teto por IP de 60/min.
- Estourou qualquer janela → `429` com corpo `RateLimitError`, header `Retry-After` (segundos) e campo `limit_window` (`minute`/`day`/`ip`) dizendo qual teto foi violado.
- **Implementar backoff respeitando `Retry-After`** em qualquer client que chamar essa API — não usar só `X-RateLimit-Remaining` isoladamente (ignora o teto diário).

## Formato de erro

Toda resposta de erro: `{ error, message, error_code, request_id }`. `error_code` é o campo
estável pra tratamento automático (ex: `unauthorized`, `service_not_accepting_submissions`).
Informar `request_id` ao reportar problema pra equipe da plataforma.

## Endpoints relevantes pro painel

### `GET /scheduling_locations` — locais de agendamento disponíveis
Query: `citizen_cpf` (obrig.), `service_id` (obrig.), `date` (opcional, filtra por disponibilidade).
Só retorna locais `is_active` **e** `user_visible` — local só-administrativo não aparece aqui
(e um `POST` de agendamento pra um local fora dessa lista responde `422`).

### `GET /scheduling_locations/{id}/available_dates` e `.../available_slots`
Datas/horários livres pra um local+serviço (+ data, no segundo). Precisa de `citizen_cpf`.

### `GET /scheduling_appointments` — lista agendamentos **do cidadão**
Query: `citizen_cpf` (obrig.), `status` (`scheduled|attended|cancelled|completed|all`), `per_page`.
Note que `cancelled` aqui agrega `cancelled_by_citizen` **e** `cancelled_by_user` — se precisar
diferenciar quem cancelou, usar o `status` exato do objeto `Appointment`, não o filtro.

### `POST /scheduling_appointments` — cria agendamento em nome do cidadão
Body: `citizen_cpf` (obrig.), `dependent_cpf` (opcional — agenda em nome de um dependente ativo
do cidadão em vez do titular), `scheduling_appointment: { service_id, scheduling_location_id, scheduled_at, form_submission_id? }`.
`422` cobre local inativo/oculto, slot ocupado, ou cidadão já com agendamento ativo pro serviço
(ver `has_active_for_service` abaixo pra checar antes).

### `GET /scheduling_appointments/by_protocol/{protocolo}`
Busca por protocolo (aceita `A.2026.42.XYZ` ou `A202642XYZ`) + `citizen_cpf` na query.

### `GET /scheduling_appointments/has_active_for_service`
Verifica se já existe agendamento ativo (`scheduled` + data futura) pro cidadão (ou um
dependente específico, via `dependent_cpf`) num serviço — usar antes de tentar criar, pra evitar
o 422 de "já tem agendamento ativo". Resposta inclui `blocking_overdue`/`blocking_overdue_appointment`
quando um agendamento **vencido** (passou da data, ainda `scheduled`) está ocupando a vaga —
nesse caso a única saída é cancelar o vencido.

### `POST /scheduling_appointments/{id}/reschedule`
Cancela o atual e cria um novo, atômico. `id` aceita ID numérico ou protocolo. Body:
`citizen_cpf`, `scheduled_at` (obrig.), `scheduling_location_id` (opcional — mantém o local
atual se omitido, mas **o destino** precisa estar ativo/visível mesmo assim).

### `PATCH /scheduling_appointments/{id}/cancel`
Body: `citizen_cpf` (obrig.), `cancellation_reason` (opcional, padrão "Cancelado pelo cidadão").
Só funciona se o agendamento estiver `scheduled` (422 caso contrário).

### `GET /dependents` (recomendado) — lista dependentes ativos do cidadão
Query: `citizen_cpf`. Retorna `[]` se não tiver dependentes (não é erro). Existe uma versão
legada em `GET /scheduling_appointments/dependents` (mesmo payload, permissão diferente,
"candidata a deprecação futura") — preferir `/dependents` em código novo.

## Schemas

### `Appointment`
```
id, protocol (ex: "A.2026.42.XYZ"), scheduled_at (datetime), status, service {id, name},
scheduling_location {id, name}, citizen {id, name, phone?, email?},
appointee {type: "Citizen"|"Dependent", id, name, relationship?, birth_date?, cpf?} | null,
cancellation_reason?, cancelled_at?, form_submission_protocol?
```
**`status` (enum real da fonte — usar esses nomes ao mapear pra status interno):**
`scheduled`, `attended`, `cancelled_by_citizen`, `cancelled_by_user`, `no_show`.

Comparar com o enum que a skill `modelo-dados` usa (`aguardando_chegada/sala_espera/chamado/
atendido/ausente`) — não bate 1:1. `attended`≈`atendido`, `no_show`≈`ausente`,
`cancelled_by_citizen`/`cancelled_by_user` não têm equivalente direto no enum atual (mais um
motivo pra revisar esse enum quando a integração de verdade for desenhada).

### `Dependent`
```
id, name, cpf (nullable), social_name (nullable), birth_date (YYYY-MM-DD),
relationship: "child"|"grandchild"|"ward"|"other"
```
Confirma o achado da planilha real: "Agendamento para (dependente)" existe de verdade no
domínio, com CPF próprio quando cadastrado.

## Pendências

- **Confirmar o host real de Jaboatão** (esse exemplo é de Ipojuca) antes de codar contra essa API de verdade.
- **Issue já aberta no GitLab da plataforma (17/09)** pedindo um endpoint interno/admin de listagem em massa dos agendamentos — resposta inicial do Mateus: "não temos atualmente", vai analisar viabilidade. Acompanhar retorno antes de descartar de vez o upload manual.
- Conseguir um token de API de teste pra validar os endpoints de verdade (nada aqui foi testado contra a API real, só documentado a partir do OpenAPI spec).
