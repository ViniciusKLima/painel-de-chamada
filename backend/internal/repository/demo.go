package repository

import (
	"context"
	"fmt"
	"time"
)

// PrefixoProtocoloDemo marca toda linha gerada pelo gerador de dados de demonstração —
// nenhum protocolo real usa esse prefixo (importação sempre vem da planilha, encaixe sempre
// "ENC-..."), então LimparDadosDemo nunca tem risco de apagar dado de verdade por engano,
// mesmo que um dia essa função seja chamada contra uma unidade com agendamentos reais.
const PrefixoProtocoloDemo = "DEMO-"

// nomesDemo — nomes brasileiros plausíveis, nunca de pessoa real (mesma regra que o design
// system do Conecta Cidades já exige pra mockup: "usar dados brasileiros plausíveis, nunca
// Lorem ipsum"). CPF/telefone gerados à parte, sempre com prefixo óbvio de teste.
var nomesDemo = []string{
	"Maria Eduarda Santos", "João Pedro Oliveira", "Ana Clara Souza", "Carlos Eduardo Lima",
	"Fernanda Costa Almeida", "Rafael Henrique Silva", "Juliana Pereira Rocha", "Lucas Gabriel Nunes",
	"Beatriz Fernandes Dias", "Pedro Henrique Araujo", "Camila Ribeiro Barbosa", "Gustavo Martins Carvalho",
	"Larissa Mendes Teixeira", "Bruno Alves Cardoso", "Patrícia Gomes Moreira",
}

// cpfDemo gera um CPF no formato certo (11 dígitos), mas sempre começando em "000" — nunca
// coincide com um CPF real de verdade (que nunca começa com 3 zeros na prática) e deixa
// óbvio, pra quem olhar o banco, que é dado sintético.
func cpfDemo(indice int) string {
	return fmt.Sprintf("000%08d", 10000000+indice*137)
}

func telefoneDemo(indice int) string {
	return fmt.Sprintf("(81) 9%04d-%04d", 1000+indice*7, 2000+indice*11)
}

// LimparDadosDemo remove qualquer agendamento gerado anteriormente (protocolo com o prefixo
// DEMO-) daquela unidade — rodado sempre antes de gerar um lote novo, pra repetir a
// demonstração não acumular lixo de execuções passadas.
func (r *Repo) LimparDadosDemo(ctx context.Context, unidadeID string) error {
	_, err := r.Pool.Exec(ctx, `
		delete from agendamentos where unidade_id = $1 and protocolo like $2
	`, unidadeID, PrefixoProtocoloDemo+"%")
	return err
}

// GerarDadosDemo cria um lote de agendamentos sintéticos, mas com mecânica 100% real (fila
// de verdade, chips de prioridade corretos, dashboard com números corretos) — pedido direto
// do dono do produto: "dado fake, mas que funciona de verdade", pra demonstrar o sistema sem
// depender de uma planilha real importada nem expor dado de cidadão de verdade. Restrito à
// prefeitura "Smartândia" na camada de controller (ver PostGerarDadosDemo) — esta função em
// si é genérica (recebe a unidade/grade prontas), não teria como aplicar a mesma trava aqui
// sem acoplar a camada de dados à regra de autorização.
//
// Mistura de estados gerada (pensada pra cobrir visualmente recepção + sala de espera +
// dashboard numa demonstração só): aguardando chegada (no horário e atrasado), já na sala de
// espera (no horário e atrasado, mais um encaixe), atendido, e ausente. Os "atendido" aqui
// NÃO ganham linha em `chamadas` (não existe nenhum atendente de demonstração pra atribuir a
// chamada) — aparecem certos nos totais do dashboard, mas não na lista "atendido por
// atendente", que depende do histórico de chamadas.
func (r *Repo) GerarDadosDemo(ctx context.Context, unidadeID, gradeID string) (int, error) {
	if err := r.LimparDadosDemo(ctx, unidadeID); err != nil {
		return 0, err
	}

	agora := time.Now()
	hoje := time.Date(agora.Year(), agora.Month(), agora.Day(), 0, 0, 0, 0, agora.Location())
	sufixo := agora.Format("150405")

	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, err
	}
	defer tx.Rollback(ctx)

	type linhaDemo struct {
		horas          int // horário previsto, em horas a partir de 00h
		status         string
		tipo           string
		prioridade     *string
		chegadaAtraso  *int // minutos depois do horário previsto, se já chegou
		marcarAtendido bool
		marcarAusente  bool
	}
	noHorario := "no_horario"
	atrasado := "atrasado"
	linhas := []linhaDemo{
		{horas: 14, status: "aguardando_chegada", tipo: "agendado"},
		{horas: 15, status: "aguardando_chegada", tipo: "agendado"},
		{horas: 16, status: "aguardando_chegada", tipo: "agendado"},
		{horas: 9, status: "sala_espera", tipo: "agendado", prioridade: &noHorario, chegadaAtraso: ptrInt(2)},
		{horas: 9, status: "sala_espera", tipo: "agendado", prioridade: &atrasado, chegadaAtraso: ptrInt(40)},
		{horas: 10, status: "sala_espera", tipo: "encaixe", chegadaAtraso: ptrInt(0)},
		{horas: 8, status: "atendido", tipo: "agendado", marcarAtendido: true},
		{horas: 8, status: "atendido", tipo: "agendado", marcarAtendido: true},
		{horas: 8, status: "ausente", tipo: "agendado", marcarAusente: true},
	}

	for i, l := range linhas {
		nome := nomesDemo[i%len(nomesDemo)]
		protocolo := fmt.Sprintf("%s%s-%02d", PrefixoProtocoloDemo, sufixo, i+1)
		cpf := cpfDemo(i)
		telefone := telefoneDemo(i)
		servico := "Atendimento Geral"

		var horarioPrevisto *time.Time
		if l.tipo != "encaixe" {
			h := hoje.Add(time.Duration(l.horas) * time.Hour)
			horarioPrevisto = &h
		}

		var chegadaEm *time.Time
		if l.chegadaAtraso != nil {
			base := hoje.Add(time.Duration(l.horas) * time.Hour)
			if l.tipo == "encaixe" {
				base = agora
			}
			c := base.Add(time.Duration(*l.chegadaAtraso) * time.Minute)
			chegadaEm = &c
		}

		var atendidoEm, ausenteEm *time.Time
		status := l.status
		if l.marcarAtendido {
			base := hoje.Add(time.Duration(l.horas) * time.Hour)
			c := base.Add(5 * time.Minute)
			a := base.Add(25 * time.Minute)
			chegadaEm = &c
			atendidoEm = &a
			np := "no_horario"
			l.prioridade = &np
		}
		if l.marcarAusente {
			base := hoje.Add(time.Duration(l.horas) * time.Hour)
			a := base.Add(80 * time.Minute)
			ausenteEm = &a
			np := "atrasado"
			l.prioridade = &np
		}

		_, err := tx.Exec(ctx, `
			insert into agendamentos (
				unidade_id, grade_id, protocolo, nome_cidadao, cpf, telefone, servico,
				tipo, horario_previsto, chegada_em, prioridade, status, atendido_em, ausente_em
			) values ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		`, unidadeID, gradeID, protocolo, nome, cpf, telefone, servico,
			l.tipo, horarioPrevisto, chegadaEm, l.prioridade, status, atendidoEm, ausenteEm)
		if err != nil {
			return 0, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, err
	}
	return len(linhas), nil
}

func ptrInt(v int) *int { return &v }
