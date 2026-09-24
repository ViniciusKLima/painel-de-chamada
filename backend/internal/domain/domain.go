package domain

import "time"

type Papel string

const (
	PapelAdmin         Papel = "admin"
	PapelGestor        Papel = "gestor"
	PapelAtendente     Papel = "atendente"
	PapelRecepcionista Papel = "recepcionista"
	// PapelPainelChamada não é gravado em nenhuma linha de `usuarios` — o painel de TV
	// continua público, sem login (ver skill painel-tv). Existe aqui só pra dar um nome de
	// primeira classe a esse contexto de acesso dentro do esquema central de permissões
	// (internal/auth/permissoes.go), caso um dia o painel precise de autenticação própria
	// (ex. um dispositivo/kiosk com credencial) sem precisar inventar o conceito do zero.
	PapelPainelChamada Papel = "painel_chamada"
)

type StatusAgendamento string

const (
	StatusAguardandoChegada StatusAgendamento = "aguardando_chegada"
	StatusSalaEspera        StatusAgendamento = "sala_espera"
	StatusChamado           StatusAgendamento = "chamado"
	StatusAtendido          StatusAgendamento = "atendido"
	StatusAusente           StatusAgendamento = "ausente"
	StatusCancelado         StatusAgendamento = "cancelado"
)

type TipoAgendamento string

const (
	TipoAgendado TipoAgendamento = "agendado"
	TipoEncaixe  TipoAgendamento = "encaixe"
)

type Prioridade string

const (
	PrioridadeNoHorario Prioridade = "no_horario"
	PrioridadeAtrasado  Prioridade = "atrasado"
)

type Prefeitura struct {
	ID        string
	Nome      string
	Slug      string
	IssuerJWT *string
	// CorDestaque/CorClara são a identidade visual da prefeitura, configurável pelo admin
	// (22/09) — aplicadas em runtime nos tokens --cor-primaria/--cor-info-fundo do
	// frontend. Hex (#rrggbb), sempre preenchidas (default = paleta atual do sistema).
	CorDestaque string
	CorClara    string
}

type Secretaria struct {
	ID           string
	PrefeituraID string
	Nome         string
	Sigla        string
}

// Unidade é o local físico (ex. "Cadastro Único Sede Prazeres"). Desde a introdução de
// Grade (22/09), a unidade não carrega mais SLA/guichês/painel diretamente — isso agora
// vive em cada Grade, porque uma unidade pode oferecer vários serviços/filas ao mesmo tempo.
type Unidade struct {
	ID           string
	SecretariaID string
	Nome         string
}

// Grade é a fila de um serviço específico dentro de uma unidade (22/09, ver skill
// modelo-dados) — ex. "CadÚnico" e "Bolsa Família" na mesma unidade seriam duas grades
// distintas, cada uma com seu próprio SLA, guichês e link de painel público.
type Grade struct {
	ID                               string
	UnidadeID                        string
	Servico                          string
	Nome                             string
	SLAChegadaMinutos                int
	SLAAtendimentoMinutos            int
	DuracaoChamadaPainelSegundos     int
	DuracaoChamadaPainelFilaSegundos int
	RepeticoesChamada                int
	IntervaloRepeticaoSegundos       int
	CooldownRechamadaMinutos         int
	// PermiteEncaixeRecepcao (23/09) — se a recepção pode cadastrar encaixe (walk-in) nesta
	// grade. Gestor/admin SEMPRE podem, independente deste campo (checado nos handlers, não
	// aqui — este campo só vale pra recepcionista).
	PermiteEncaixeRecepcao bool
	Ativo                  bool
	CriadoEm               time.Time
	// Excluida (23/09, auditoria de reativação) — true quando `excluido_em` está preenchido.
	// Distinto de `Ativo` (desativação reversível, sempre teve botão de ligar/desligar):
	// exclusão era soft-delete SEM volta na UI até esta rodada — agora tem endpoint de
	// reativar, mas o valor continua sendo exposto pra a tela saber mostrar o botão certo.
	Excluida bool
}

// Usuario carrega vínculos DIFERENTES por papel (22/09, ver skill modelo-dados) — nem todo
// campo de escopo é preenchido pra todo mundo:
//   - admin: nenhum vínculo (SecretariaID e UnidadeID nulos) — acesso master à plataforma.
//   - gestor: SecretariaID preenchido (gerencia TODAS as unidades/grades daquela
//     secretaria); UnidadeID nulo.
//   - recepcionista: UnidadeID preenchido (trabalha o local físico inteiro) + 0 ou mais
//     grades alocadas (`usuario_grades`) — sem nenhuma, continua vendo/cadastrando em
//     qualquer grade da unidade (comportamento de sempre); com uma ou mais, só nelas.
//   - atendente: UnidadeID preenchido + 1 ou mais grades alocadas (`usuario_grades`,
//     23/09 — antes era uma FK única `grade_id`, "atendente ou recepção podem ser alocados
//     em mais de uma grade").
//
// As grades alocadas NÃO vêm embutidas neste struct (não são coluna de `usuarios` mais) —
// buscar via Repo.ListarGradesDoUsuario quando precisar. A sessão devolve a lista pro
// frontend escolher em qual grade trabalhar no momento (guichê/sala), não um valor fixo.
//
// Ver internal/auth/permissoes.go pra centralização das regras que usam esses campos —
// nenhum handler deve comparar esses IDs diretamente, sempre passar pelas funções de lá.
type Usuario struct {
	ID             string
	SecretariaID   *string
	UnidadeID      *string
	Nome           string
	Email          string
	SenhaHash      *string
	Papel          Papel
	ExternalSub    *string
	Ativo          bool
	UltimoAcessoEm *time.Time
}

type TipoGuiche string

const (
	TipoGuicheGuiche TipoGuiche = "guiche"
	TipoGuicheSala   TipoGuiche = "sala"
)

// Guiche pertence a uma Grade agora, não mais direto à unidade (22/09) — cada grade tem seu
// próprio conjunto de guichês/salas.
//
// OcupadoPorUsuarioID/OcupadoPorNome/OcupadoEm (22/09) — ocupação em tempo real: quando um
// atendente escolhe um guichê no modal de seleção, esses campos ficam preenchidos, e o
// frontend reafirma a ocupação periodicamente (heartbeat, ver skill regras-negocio-fila).
// A consulta que preenche esses campos (repository) já filtra por heartbeat recente — um
// guichê "esquecido" (aba fechada sem logout) aparece livre de novo sozinho depois de um
// tempo sem heartbeat, sem precisar de nenhuma limpeza manual.
type Guiche struct {
	ID                  string
	GradeID             string
	Nome                string
	Ativo               bool
	Tipo                TipoGuiche
	Andar               *string
	Capacidade          int
	OcupadoPorUsuarioID *string
	OcupadoPorNome      *string
	OcupadoEm           *time.Time
	// Excluida (23/09) — mesmo sentido de Grade.Excluida, ver comentário lá.
	Excluida bool
}

type LoteImportacao struct {
	ID           string
	SecretariaID string
	UsuarioID    string
	ArquivoNome  string
	DataUpload   time.Time
	Status       string
	TotalLinhas  *int
	TotalErros   *int
}

type Agendamento struct {
	ID          string
	UnidadeID   string
	GradeID     string
	LoteID      *string
	Protocolo   string
	NomeCidadao string
	CPF         *string
	Telefone    *string
	Servico     *string
	// GradeHorario é o texto solto que vinha da coluna "Grade de horários" da planilha —
	// puramente informativo, não confundir com GradeID (a fila de verdade, ver skill
	// modelo-dados). Mantido só pra exibição/histórico.
	GradeHorario       *string
	Tipo               TipoAgendamento
	HorarioPrevisto    *time.Time
	ChegadaEm          *time.Time
	Prioridade         *Prioridade
	GuicheID           *string
	Status             StatusAgendamento
	AtendidoEm         *time.Time
	AusenteEm          *time.Time
	CanceladoEm        *time.Time
	MotivoCancelamento *string
	CriadoEm           time.Time
}

type Chamada struct {
	ID            string
	AgendamentoID string
	GuicheID      string
	UsuarioID     string
	ChamadoEm     time.Time
}
