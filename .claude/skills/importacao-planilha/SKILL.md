---
name: importacao-planilha
description: Use ao implementar o parser de upload de planilha (CSV/Excel) do gestor, ou qualquer lógica que leia colunas de um export real da plataforma Jornada/Conecta Cidades. Fonte da verdade do mapeamento de colunas real → schema — baseado em exports reais analisados, não invenção.
---

# Importação de planilha — mapeamento de colunas real

Baseado em 3 exports reais analisados (14/09: dois arquivos de ~570 e ~45 linhas; 19/09: um
terceiro de 176 linhas, já com colunas novas pedidas pelo usuário aos devs da plataforma). Os
arquivos originais não ficam versionados no repo (contêm CPF/telefone reais — cobertos pelo
`.gitignore`).

## Colunas confirmadas (export de 19/09)

```
Protocolo, Data, Horário, Cidadão (responsável), CPF, Agendamento para (dependente), Serviço,
Grade de horários, Local no mapa, Status, Confirmado em, Telefone, Cancelado em,
Motivo do cancelamento, Observações, Notas do atendimento, Atendido por
```

Em relação ao export de 14/09, ganhou 3 colunas: **`Grade de horários`**, **`Local no mapa`**,
**`Atendido por`**.

## Mapeamento pra `agendamentos`

| Coluna da planilha | Campo | Observação |
| --- | --- | --- |
| `Protocolo` | `protocolo` | identificador único por secretaria/prefeitura |
| `Data` + `Horário` | `horario_previsto` | "Data" vem em português por extenso **sem ano** (ex: "21 de Setembro") — assumir o ano do momento do upload; "Horário" é `HH:mm`. Ver seção de parsing abaixo. |
| `Cidadão (responsável)` | `nome_cidadao` (default) | usado quando `Agendamento para (dependente)` está vazio |
| `Agendamento para (dependente)` | `nome_cidadao` (override) | quando preenchido, é esse nome que vira `nome_cidadao` — o agendamento foi feito por um responsável em nome de um dependente |
| `CPF` | `cpf` | guardado, mascarado na API pra quem não é gestor/admin (ver `modelo-dados`) |
| `Telefone` | `telefone` | mesma regra de máscara do CPF |
| `Serviço` | `servico` + `grade_id` (via lookup ou criação) | texto livre, tipo de atendimento — **e também**, desde 22/09, a chave de roteamento pra `grade` (junto com `Local no mapa`). Ver seção abaixo. |
| `Grade de horários` | `grade_horario` | texto livre, só informativo — **não** é a fonte da entidade `grades` do sistema, apesar do nome parecido. Ver seção abaixo. |
| `Local no mapa` | `unidade_id` (via lookup ou criação) | casar por nome (normalizado: sem acento, case-insensitive, sem espaços extras) contra `unidades.nome` já cadastradas na secretaria. **Confirmado 19/09: não encontrado → cria a unidade automaticamente** com esse nome (não é mais erro de linha). Local **vazio** continua erro de linha — a criação automática só vale pra nome preenchido e não encontrado. |
| `Status` | `status` + `cancelado_em` + `motivo_cancelamento` | ver mapeamento de status abaixo |
| `Confirmado em` | *(não mapeado ainda)* | timestamp de confirmação via SMS/sistema — não usado pelo nosso fluxo hoje; pode interessar pra métrica futura de "taxa de confirmação" |
| `Cancelado em` | `cancelado_em` | só relevante quando `Status` = cancelado |
| `Motivo do cancelamento` | `motivo_cancelamento` | ver ressalva sobre reagendamento disfarçado, abaixo |
| `Observações` / `Notas do atendimento` | *(não mapeados)* | sempre vazios em todos os exports vistos até agora; schema não tem campo pra isso ainda — adicionar se algum dia vierem preenchidos |
| `Atendido por` | *(não mapeado ainda)* | nova coluna (19/09), sempre vazia nos agendamentos futuros/pendentes do export visto — provavelmente só populada retroativamente em atendimentos já concluídos pela plataforma. Não confundir com `chamadas.usuario_id` (que é o log **do nosso** atendente chamando **no nosso** painel) — essa coluna é sobre quem atendeu **na plataforma de origem**, um conceito histórico separado. Não implementar nada com isso ainda, só reconhecer a coluna pra não gerar erro de parsing. |

## Grade de horários vs. Local no mapa vs. Serviço — ✅ resolvido (22/09)

Ambiguidade que ficou em aberto desde 19/09 (ver histórico abaixo) — resolvida com uma
pergunta direta de esclarecimento ao dono do produto. Resposta literal: **"unidade = local,
grade de horário = agenda para um serviço dentro daquele local"**.

Isso separa três conceitos que antes pareciam se sobrepor:

- **`Local no mapa`** → continua sendo a fonte de verdade pra `agendamentos.unidade_id`, sem
  mudança. É o local físico.
- **`Serviço`** → agora é a chave de roteamento pra `agendamentos.grade_id` (nova entidade,
  ver skill `modelo-dados`) — cada combinação (`Local no mapa`, `Serviço`) vira/casa com uma
  `grade`, auto-criada na importação do mesmo jeito que unidade nova já era (`Resolver
  OuCriarGrade`, análogo a `ResolverOuCriarUnidade`). Serviço vazio cai num bucket "Geral"
  por unidade.
- **`Grade de horários`** (a coluna da planilha) → **continua sendo só o texto informativo**
  de sempre, guardado em `agendamentos.grade_horario` — **não é** a fonte da nova entidade
  `grades` do sistema, apesar do nome parecido. Motivo: nos exports reais vistos até agora,
  o valor dessa coluna sempre foi idêntico a `Local no mapa` (não ao `Serviço`), então não é
  confiável como critério de separação de fila — o texto "Grade de horários" da plataforma de
  origem e o conceito de "grade" deste sistema **não são a mesma coisa**, apesar do nome
  coincidir. Não confundir os dois ao mexer no parser.

**Histórico da dúvida original (19/09, mantido por registro)**: nos 3 exports vistos até
então, toda linha tinha `Grade de horários` idêntico a `Local no mapa` (o usuário tinha
filtrado a planilha pra uma unidade só de propósito), o que não permitia confirmar se grade e
local eram 1:1, se um local podia ter várias grades, ou se uma grade podia abranger vários
locais. A simplificação de MVP na época foi "ignorar Grade de horários pro roteamento,
só Local importa" — sinalizada explicitamente como provisória ("isso vai escalar"). A
resolução de 22/09 não é bem essa simplificação sendo revertida; é a introdução de um
critério de roteamento adicional (`Serviço`) que a dúvida original nem tinha considerado
como candidato.

## Mapeamento de status (planilha → nosso enum)

A coluna `Status` da planilha é sobre o agendamento em si (visão da plataforma de origem), não
sobre onde a pessoa está no nosso fluxo físico — são conceitos diferentes.

| `Status` da planilha | `status` em `agendamentos` | Observação |
| --- | --- | --- |
| `Agendado` | `aguardando_chegada` | entra no fluxo normal |
| `Cancelado pelo cidadão` | `cancelado` | grava `cancelado_em` e `motivo_cancelamento`; **não** entra na sala de espera, nunca é chamado |

⚠️ **Reagendamento disfarçado de cancelamento**: em exports anteriores (14/09), vários registros
com `Status = Cancelado pelo cidadão` tinham `Motivo do cancelamento` no formato "Reagendado:
Local alterado de 'X' para 'Y'; Data/hora alterada para..." — ou seja, a pessoa não desistiu, só
mudou de dia/local, e um novo agendamento (novo protocolo) deve existir na mesma planilha ou numa
futura.

**Decisão (19/09)**: por enquanto, importar esses registros **igual a qualquer outro
cancelamento** (status `cancelado`, sem distinção) — mas fica **explicitamente registrado que
isso não é a mesma coisa** que uma desistência de verdade, e **não deve** ser contado junto numa
métrica futura de "não comparecimento"/cancelamento no dashboard do gestor sem antes discutir
como separar os dois casos (ex: por regex no motivo, procurando o prefixo "Reagendado:"). Ver
skill `regras-negocio-fila` antes de reportar qualquer métrica desse tipo.

## Parsing de data/hora

- `"21 de Setembro"` (dia + mês por extenso em português, **sem ano**) + `"12:40"` → combinar
  assumindo o ano do momento do upload. Risco documentado: quebra em uploads que atravessem
  virada de ano — aceitável pro piloto, revisar se isso virar operação real de longo prazo.
- `"01 de Setembro de 2026, 06:08"` (formato usado em `Confirmado em`/`Cancelado em`, quando
  presentes) — **esse formato já inclui o ano**, parsing diferente do campo `Data` principal.

## Protocolo — duplicidade

`protocolo` deve ser único (constraint recomendada por secretaria, não por unidade — o mesmo
protocolo nunca deve aparecer em duas linhas, mesmo em unidades diferentes). Linha com protocolo
já existente no banco, ou duplicado dentro do próprio arquivo: erro de linha, não sobrescrever.

## Fluxo do upload (visão geral, ver spec formal em `specs/002-importacao-multi-local`)

0. **(22/09) Fluxo em duas etapas, pedido do dono do produto**: antes de gravar qualquer coisa,
   o gestor vê uma **análise prévia** (endpoint separado, `.../lotes/preview`, roda a mesma
   validação mas não escreve no banco) — total de agendamentos válidos, por dia, quantas
   unidades/grades novas seriam criadas, e a lista de erros. Só depois de "Confirmar" o
   upload de verdade é reenviado pro endpoint que grava (`.../lotes`, o de sempre).
1. Gestor sobe o arquivo (associado à **secretaria**, não a uma unidade específica — ver correção
   em `modelo-dados`).
2. Sistema lê cada linha, resolve `unidade_id` via `Local no mapa` (**cria a unidade
   automaticamente se o nome não existir ainda**, confirmado 19/09 — ver `specs/002-importacao-multi-local`)
   e resolve `grade_id` via (`Local no mapa`, `Serviço`) do mesmo jeito (**cria a grade
   automaticamente**, confirmado 22/09 — ver skill `modelo-dados`), valida campos obrigatórios
   (protocolo, nome, data/horário, local **preenchido**).
3. Linha inválida (campo obrigatório ausente — incluindo local vazio — ou protocolo duplicado) →
   erro reportado por linha (número da linha + motivo), sem descartar as válidas do resto do
   arquivo (mesmo padrão já usado na versão Node). Local preenchido mas não encontrado **não** é
   mais erro — vira criação automática de unidade.
4. Linhas válidas viram `agendamentos` (`status = aguardando_chegada` ou `cancelado`, conforme
   `Status` da planilha).
5. Um `lote_importacao` por upload, cobrindo todas as unidades tocadas.

## Não usar `xlsx`/SheetJS do npm

Repetindo o que já está registrado no `CLAUDE.md` — 2 CVEs de alta severidade sem correção
(prototype pollution + ReDoS), exploráveis via arquivo malicioso, que é exatamente o vetor de um
upload de planilha. No ecossistema Go, `qax-os/excelize` é a opção mais usada e mantida.
