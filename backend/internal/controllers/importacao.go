package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/xuri/excelize/v2"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/util"
	"painel-chamada-backend/internal/views"
)

// Mapeamento de colunas real, baseado em exports analisados — ver skill importacao-planilha.
var aliasesColuna = map[string][]string{
	"protocolo":   {"protocolo"},
	"data":        {"data"},
	"horario":     {"horario", "hora"},
	"nome":        {"cidadao responsavel", "nome", "nome cidadao", "cidadao"},
	"dependente":  {"agendamento para dependente", "dependente"},
	"cpf":         {"cpf"},
	"telefone":    {"telefone"},
	"servico":     {"servico"},
	"grade":       {"grade de horarios", "grade horarios", "grade"},
	"local":       {"local no mapa", "local", "unidade"},
	"status":      {"status"},
	"canceladoEm": {"cancelado em"},
	"motivo":      {"motivo do cancelamento", "motivo cancelamento"},
}

var mesesPortugues = map[string]int{
	"janeiro": 1, "fevereiro": 2, "marco": 3, "abril": 4, "maio": 5, "junho": 6,
	"julho": 7, "agosto": 8, "setembro": 9, "outubro": 10, "novembro": 11, "dezembro": 12,
}

var reDataSimples = regexp.MustCompile(`(?i)^(\d{1,2})\s+de\s+([a-zçã]+)$`)
var reHorario = regexp.MustCompile(`^(\d{1,2}):(\d{2})$`)

func normalizarTextoColuna(s string) string {
	return util.NormalizarCabecalho(s)
}

func mapearColunas(cabecalhos []string) map[string]int {
	normalizados := make([]string, len(cabecalhos))
	for i, c := range cabecalhos {
		normalizados[i] = normalizarTextoColuna(c)
	}
	indices := map[string]int{}
	for campo, aliases := range aliasesColuna {
		for i, n := range normalizados {
			achou := false
			for _, a := range aliases {
				if n == a {
					achou = true
					break
				}
			}
			if achou {
				indices[campo] = i
				break
			}
		}
	}
	return indices
}

func celula(linha []string, indices map[string]int, campo string) string {
	idx, ok := indices[campo]
	if !ok || idx >= len(linha) {
		return ""
	}
	return strings.TrimSpace(linha[idx])
}

// parseDataHorario combina "21 de Setembro" (sem ano — assume o ano do momento do
// upload) com "12:40". Ver skill importacao-planilha.
func parseDataHorario(dataTexto, horarioTexto string, agora time.Time) (time.Time, bool) {
	md := reDataSimples.FindStringSubmatch(strings.TrimSpace(dataTexto))
	mh := reHorario.FindStringSubmatch(strings.TrimSpace(horarioTexto))
	if md == nil || mh == nil {
		return time.Time{}, false
	}
	mes, ok := mesesPortugues[normalizarTextoColuna(md[2])]
	if !ok {
		return time.Time{}, false
	}
	dia, _ := strconv.Atoi(md[1])
	hora, _ := strconv.Atoi(mh[1])
	min, _ := strconv.Atoi(mh[2])
	return time.Date(agora.Year(), time.Month(mes), dia, hora, min, 0, 0, agora.Location()), true
}

type erroLinha struct {
	Linha  int    `json:"linha"`
	Motivo string `json:"motivo"`
}

type respostaImportacao struct {
	LoteID          string      `json:"loteId"`
	Status          string      `json:"status"`
	TotalLinhas     int         `json:"totalLinhas"`
	TotalCriados    int         `json:"totalCriados"`
	TotalCancelados int         `json:"totalCancelados"`
	TotalErros      int         `json:"totalErros"`
	Erros           []erroLinha `json:"erros"`
}

// PostImportarPlanilha: POST /api/v1/secretarias/{secretariaId}/lotes — só gestor/admin.
func (c *Controllers) PostImportarPlanilha(w http.ResponseWriter, r *http.Request) {
	usuario, secretariaID, ok := c.exigirSecretaria(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin pode importar planilha")
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "arquivo_invalido", "nao foi possivel ler o formulario enviado")
		return
	}
	file, header, err := r.FormFile("file")
	if err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "arquivo_ausente", "nenhum arquivo enviado (campo 'file' esperado)")
		return
	}
	defer file.Close()

	f, err := excelize.OpenReader(file)
	if err != nil {
		views.ResponderErro(w, http.StatusBadRequest, "arquivo_invalido", "nao foi possivel ler o arquivo — confira se e um .xlsx valido")
		return
	}
	defer f.Close()

	sheets := f.GetSheetList()
	if len(sheets) == 0 {
		views.ResponderErro(w, http.StatusBadRequest, "arquivo_invalido", "planilha sem nenhuma aba")
		return
	}
	linhas, err := f.GetRows(sheets[0])
	if err != nil || len(linhas) < 2 {
		views.ResponderErro(w, http.StatusBadRequest, "arquivo_invalido", "planilha vazia ou sem linhas de dados")
		return
	}

	indices := mapearColunas(linhas[0])
	dadosLinhas := linhas[1:]

	ctx := r.Context()

	agora := time.Now()
	unidadesCache := map[string]*domain.Unidade{}
	gradesCache := map[string]*domain.Grade{}
	protocolosNoArquivo := map[string]bool{}
	var validas []repository.NovoAgendamento
	// Slice vazio, não nil (22/09) — evita virar `null` no JSON quando não há erro nenhum
	// (ver o mesmo achado em importacao_preview.go).
	erros := []erroLinha{}
	totalCancelados := 0

	for i, linha := range dadosLinhas {
		numeroLinha := i + 2
		protocolo := celula(linha, indices, "protocolo")
		responsavel := celula(linha, indices, "nome")
		dependente := celula(linha, indices, "dependente")
		nomeCidadao := responsavel
		if dependente != "" {
			nomeCidadao = dependente
		}
		localTexto := celula(linha, indices, "local")

		if protocolo == "" {
			erros = append(erros, erroLinha{numeroLinha, "Protocolo ausente."})
			continue
		}
		if nomeCidadao == "" {
			erros = append(erros, erroLinha{numeroLinha, "Nome do cidadao (ou dependente) ausente."})
			continue
		}
		horarioPrevisto, ok := parseDataHorario(celula(linha, indices, "data"), celula(linha, indices, "horario"), agora)
		if !ok {
			erros = append(erros, erroLinha{numeroLinha, "Data/horario ausente ou em formato nao reconhecido."})
			continue
		}
		if protocolosNoArquivo[protocolo] {
			erros = append(erros, erroLinha{numeroLinha, fmt.Sprintf("Protocolo '%s' duplicado no arquivo.", protocolo)})
			continue
		}
		existe, err := c.Repo.ProtocoloExisteNaSecretaria(ctx, secretariaID, protocolo)
		if err != nil {
			views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao validar protocolo")
			return
		}
		if existe {
			erros = append(erros, erroLinha{numeroLinha, fmt.Sprintf("Protocolo '%s' ja existe.", protocolo)})
			continue
		}
		if localTexto == "" {
			erros = append(erros, erroLinha{numeroLinha, "Local ausente."})
			continue
		}

		unidade, achou := unidadesCache[normalizarTextoColuna(localTexto)]
		if !achou {
			u, err := c.Repo.ResolverOuCriarUnidade(ctx, secretariaID, localTexto)
			if err != nil {
				views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao resolver unidade")
				return
			}
			unidade = u
			unidadesCache[normalizarTextoColuna(localTexto)] = unidade
		}

		// Grade = fila de um serviço específico dentro da unidade (22/09, ver skill
		// modelo-dados) — resolvida/criada automaticamente pela combinação unidade+serviço,
		// mesmo padrão já usado pra unidade via "Local no mapa".
		servicoTexto := celula(linha, indices, "servico")
		nomeGradeTexto := celula(linha, indices, "grade")
		chaveGrade := unidade.ID + "|" + normalizarTextoColuna(servicoTexto)
		grade, achouGrade := gradesCache[chaveGrade]
		if !achouGrade {
			g, err := c.Repo.ResolverOuCriarGrade(ctx, unidade.ID, servicoTexto, nomeGradeTexto)
			if err != nil {
				views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao resolver grade")
				return
			}
			grade = g
			gradesCache[chaveGrade] = grade
		}

		statusTexto := normalizarTextoColuna(celula(linha, indices, "status"))
		cancelado := strings.HasPrefix(statusTexto, "cancelado")

		n := repository.NovoAgendamento{
			UnidadeID:       unidade.ID,
			GradeID:         grade.ID,
			Protocolo:       protocolo,
			NomeCidadao:     nomeCidadao,
			Tipo:            domain.TipoAgendado,
			HorarioPrevisto: &horarioPrevisto,
		}
		if v := celula(linha, indices, "cpf"); v != "" {
			n.CPF = &v
		}
		if v := celula(linha, indices, "telefone"); v != "" {
			n.Telefone = &v
		}
		if servicoTexto != "" {
			n.Servico = &servicoTexto
		}
		if v := celula(linha, indices, "grade"); v != "" {
			n.GradeHorario = &v
		}
		if cancelado {
			n.Status = domain.StatusCancelado
			if v := celula(linha, indices, "motivo"); v != "" {
				n.MotivoCancelamento = &v
			}
			totalCancelados++
		} else {
			n.Status = domain.StatusAguardandoChegada
		}

		protocolosNoArquivo[protocolo] = true
		validas = append(validas, n)
	}

	status := "concluido"
	if len(validas) == 0 {
		status = "erro"
	}

	var loteID string
	err = pgx.BeginFunc(ctx, c.Repo.Pool, func(tx pgx.Tx) error {
		id, err := c.Repo.CriarLote(ctx, tx, secretariaID, usuario.ID, header.Filename)
		if err != nil {
			return err
		}
		loteID = id
		for i := range validas {
			validas[i].LoteID = &loteID
			if err := c.Repo.CriarAgendamento(ctx, tx, validas[i]); err != nil {
				return err
			}
		}
		return c.Repo.FinalizarLote(ctx, tx, loteID, status, len(dadosLinhas), len(erros))
	})
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao gravar a importacao: "+err.Error())
		return
	}

	views.ResponderJSON(w, http.StatusCreated, respostaImportacao{
		LoteID:          loteID,
		Status:          status,
		TotalLinhas:     len(dadosLinhas),
		TotalCriados:    len(validas),
		TotalCancelados: totalCancelados,
		TotalErros:      len(erros),
		Erros:           erros,
	})
}
