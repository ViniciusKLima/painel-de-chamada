package controllers

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/xuri/excelize/v2"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/views"
)

type entidadeNova struct {
	Unidade string `json:"unidade"`
	Servico string `json:"servico,omitempty"`
}

type porDiaPreview struct {
	Data  string `json:"data"`
	Total int    `json:"total"`
}

type respostaPreview struct {
	TotalLinhas     int             `json:"totalLinhas"`
	TotalValidas    int             `json:"totalValidas"`
	TotalCancelados int             `json:"totalCancelados"`
	TotalErros      int             `json:"totalErros"`
	Erros           []erroLinha     `json:"erros"`
	PorDia          []porDiaPreview `json:"porDia"`
	UnidadesNovas   []string        `json:"unidadesNovas"`
	GradesNovas     []entidadeNova  `json:"gradesNovas"`
}

// PostPreviewPlanilha: POST /api/v1/secretarias/{secretariaId}/lotes/preview — só gestor.
// Primeiro passo do fluxo de importação em duas etapas (22/09, pedido do dono do produto):
// lê e valida a planilha SEM gravar nada no banco (nem unidade/grade nova) — só pra o
// gestor conferir "tantos agendamentos pro dia tal, confirma?" antes de comprometer.
// Reaproveita as mesmas regras de validação de PostImportarPlanilha, mas em modo
// só-leitura: unidade/grade que não existem ainda aparecem como "seria criada", sem
// escrever no banco (a criação de verdade só acontece na confirmação, via
// PostImportarPlanilha com o mesmo arquivo).
func (c *Controllers) PostPreviewPlanilha(w http.ResponseWriter, r *http.Request) {
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
	file, _, err := r.FormFile("file")
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

	// Cache de "essa unidade já existe?" — igual ResolverOuCriarUnidade, mas sem criar.
	unidadesExistentes, err := c.Repo.ListarUnidades(ctx, secretariaID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao ler unidades existentes")
		return
	}
	unidadeExiste := map[string]bool{}
	for _, u := range unidadesExistentes {
		unidadeExiste[normalizarTextoColuna(u.Nome)] = true
	}
	// Cache de grades já existentes, carregado sob demanda por unidade (só as que já existem).
	gradesExistentesPorUnidade := map[string]map[string]bool{} // nome normalizado da unidade -> set de serviços normalizados

	// Inicializados como slice vazio, não nil (22/09, bug real encontrado ao testar ao
	// vivo): um slice nil vira `null` no JSON, e o frontend quebra tentando ler `.length`
	// de `null` quando não há nenhum erro/unidade nova/grade nova — o caso mais comum.
	erros := []erroLinha{}
	protocolosNoArquivo := map[string]bool{}
	contagemPorDia := map[string]int{}
	totalCancelados := 0
	totalValidas := 0
	unidadesNovasSet := map[string]bool{}
	unidadesNovas := []string{}
	gradesNovasSet := map[string]bool{}
	gradesNovas := []entidadeNova{}

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

		localChave := normalizarTextoColuna(localTexto)
		servicoTexto := celula(linha, indices, "servico")
		servicoChave := normalizarTextoColuna(servicoTexto)
		if servicoChave == "" {
			servicoChave = "geral"
		}

		if !unidadeExiste[localChave] {
			if !unidadesNovasSet[localChave] {
				unidadesNovasSet[localChave] = true
				unidadesNovas = append(unidadesNovas, localTexto)
			}
			chaveGrade := localChave + "|" + servicoChave
			if !gradesNovasSet[chaveGrade] {
				gradesNovasSet[chaveGrade] = true
				gradesNovas = append(gradesNovas, entidadeNova{Unidade: localTexto, Servico: servicoTexto})
			}
		} else {
			gradesDaUnidade, carregado := gradesExistentesPorUnidade[localChave]
			if !carregado {
				// Encontra a unidade real pelo nome normalizado pra listar as grades dela.
				gradesDaUnidade = map[string]bool{}
				for _, u := range unidadesExistentes {
					if normalizarTextoColuna(u.Nome) == localChave {
						grades, err := c.Repo.ListarGradesPorUnidade(ctx, u.ID)
						if err != nil {
							views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", "falha ao ler grades existentes")
							return
						}
						for _, g := range grades {
							gradesDaUnidade[normalizarTextoColuna(g.Servico)] = true
						}
						break
					}
				}
				gradesExistentesPorUnidade[localChave] = gradesDaUnidade
			}
			if !gradesDaUnidade[servicoChave] {
				chaveGrade := localChave + "|" + servicoChave
				if !gradesNovasSet[chaveGrade] {
					gradesNovasSet[chaveGrade] = true
					gradesNovas = append(gradesNovas, entidadeNova{Unidade: localTexto, Servico: servicoTexto})
				}
			}
		}

		statusTexto := normalizarTextoColuna(celula(linha, indices, "status"))
		if statusTexto == "" || len(statusTexto) < 9 || statusTexto[:9] != "cancelado" {
			// segue como agendado — conta pro resumo por dia
		}
		cancelado := len(statusTexto) >= 9 && statusTexto[:9] == "cancelado"
		if cancelado {
			totalCancelados++
		}
		contagemPorDia[horarioPrevisto.Format("2006-01-02")]++

		protocolosNoArquivo[protocolo] = true
		totalValidas++
	}

	porDia := []porDiaPreview{}
	for data, total := range contagemPorDia {
		porDia = append(porDia, porDiaPreview{Data: data, Total: total})
	}
	sort.Slice(porDia, func(i, j int) bool { return porDia[i].Data < porDia[j].Data })

	views.ResponderJSON(w, http.StatusOK, respostaPreview{
		TotalLinhas:     len(dadosLinhas),
		TotalValidas:    totalValidas,
		TotalCancelados: totalCancelados,
		TotalErros:      len(erros),
		Erros:           erros,
		PorDia:          porDia,
		UnidadesNovas:   unidadesNovas,
		GradesNovas:     gradesNovas,
	})
}
