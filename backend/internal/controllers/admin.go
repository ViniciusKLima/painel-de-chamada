package controllers

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strings"

	"github.com/go-chi/chi/v5"
	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/domain"
	"painel-chamada-backend/internal/repository"
	"painel-chamada-backend/internal/views"
)

func mustCompileHex() *regexp.Regexp {
	return regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)
}

// Controllers do Admin (22/09) — hierarquia Admin → Prefeitura → Secretaria → Usuários. O
// admin é o único papel sem vínculo de tenant nenhum (sem secretaria_id/unidade_id), então
// TODO controller aqui exige `usuario.Papel == domain.PapelAdmin` explicitamente — não faz
// sentido reaproveitar exigirSecretaria/exigirUnidade (que existem justamente pra checar um
// vínculo que o admin não tem). Ver skill papeis-e-telas / internal/auth/permissoes.go.

func exigirAdmin(w http.ResponseWriter, r *http.Request) (usuario *domain.Usuario, ok bool) {
	usuario = auth.UsuarioDoContexto(r.Context())
	if !auth.EhAdmin(usuario) {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas admin")
		return nil, false
	}
	return usuario, true
}

// GetPrefeituras: GET /api/v1/prefeituras — só admin.
func (c *Controllers) GetPrefeituras(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	lista, err := c.Repo.ListarPrefeituras(r.Context())
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, p := range lista {
		pp := p
		out[i] = views.PrefeituraParaJSON(&pp)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

type corpoNovaPrefeitura struct {
	Nome string `json:"nome"`
}

// PostPrefeitura: POST /api/v1/prefeituras — só admin.
func (c *Controllers) PostPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	var corpo corpoNovaPrefeitura
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || strings.TrimSpace(corpo.Nome) == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e obrigatorio")
		return
	}
	p, err := c.Repo.CriarPrefeitura(r.Context(), corpo.Nome)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, views.PrefeituraParaJSON(p))
}

// GetPrefeitura: GET /api/v1/prefeituras/{prefeituraId} — só admin. Prefeitura + secretarias
// dela, base da tela "Configurações da Prefeitura".
func (c *Controllers) GetPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	p, err := c.Repo.BuscarPrefeituraPorID(r.Context(), prefeituraID)
	if err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "prefeitura nao encontrada")
		return
	}
	secretarias, err := c.Repo.ListarSecretariasPorPrefeitura(r.Context(), prefeituraID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	secretariasJSON := make([]map[string]any, len(secretarias))
	for i, s := range secretarias {
		ss := s
		secretariasJSON[i] = views.SecretariaParaJSON(&ss)
	}
	views.ResponderJSON(w, http.StatusOK, map[string]any{
		"prefeitura":  views.PrefeituraParaJSON(p),
		"secretarias": secretariasJSON,
	})
}

// GetDashboardPrefeitura: GET /api/v1/prefeituras/{prefeituraId}/dashboard — só admin (23/09,
// aba "Dashboard" dentro da configuração de uma prefeitura — pedido: "dashboard e usuarios
// internos... dentro da configuração de cada prefeitura"). Mesmo formato de resposta do
// dashboard do gestor, só que agregado por TODAS as secretarias da prefeitura de uma vez.
func (c *Controllers) GetDashboardPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	if _, err := c.Repo.BuscarPrefeituraPorID(r.Context(), prefeituraID); err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "prefeitura nao encontrada")
		return
	}
	resumo, err := c.Repo.ResumoDoDiaPrefeitura(r.Context(), prefeituraID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, resumo)
}

// GetUsuariosPrefeitura: GET /api/v1/prefeituras/{prefeituraId}/usuarios — só admin (23/09,
// aba "Usuários" da configuração da prefeitura) — todo gestor/atendente/recepcionista
// daquela prefeitura específica, com as grades de cada um já resolvidas (etiquetas).
func (c *Controllers) GetUsuariosPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	if _, err := c.Repo.BuscarPrefeituraPorID(r.Context(), prefeituraID); err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "prefeitura nao encontrada")
		return
	}
	lista, err := c.Repo.ListarUsuariosInternosPorPrefeitura(r.Context(), prefeituraID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, u := range lista {
		out[i] = views.UsuarioComContextoParaJSON(u)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// GetPaineisPrefeitura: GET /api/v1/prefeituras/{prefeituraId}/paineis — só admin (23/09,
// aba "Painéis" da configuração da prefeitura) — todas as grades ativas de todas as unidades
// de todas as secretarias dessa prefeitura, com o nome da unidade junto (a central de
// painéis do gestor mostra isso pra UMA secretaria; esta é a versão pra prefeitura inteira).
func (c *Controllers) GetPaineisPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	if _, err := c.Repo.BuscarPrefeituraPorID(r.Context(), prefeituraID); err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "prefeitura nao encontrada")
		return
	}
	grades, err := c.Repo.ListarGradesAtivasPorPrefeitura(r.Context(), prefeituraID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(grades))
	for i, g := range grades {
		out[i] = map[string]any{"id": g.ID, "nome": g.Nome, "unidadeId": g.UnidadeID, "unidadeNome": g.UnidadeNome}
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

type corpoAtualizacaoPrefeitura struct {
	Nome        string `json:"nome"`
	CorDestaque string `json:"corDestaque"`
	CorClara    string `json:"corClara"`
}

var regexCorHex = mustCompileHex()

// PatchPrefeitura: PATCH /api/v1/prefeituras/{prefeituraId} — só admin. Nome e as duas
// cores de identidade (destaque/clara — ver skill identidade-visual).
func (c *Controllers) PatchPrefeitura(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	var corpo corpoAtualizacaoPrefeitura
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || strings.TrimSpace(corpo.Nome) == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e obrigatorio")
		return
	}
	if !regexCorHex.MatchString(corpo.CorDestaque) || !regexCorHex.MatchString(corpo.CorClara) {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "corDestaque e corClara devem ser hex validos (#rrggbb)")
		return
	}
	p, err := c.Repo.AtualizarPrefeitura(r.Context(), prefeituraID, corpo.Nome, corpo.CorDestaque, corpo.CorClara)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, views.PrefeituraParaJSON(p))
}

type corpoNovaSecretaria struct {
	Nome  string `json:"nome"`
	Sigla string `json:"sigla"`
}

// PostSecretaria: POST /api/v1/prefeituras/{prefeituraId}/secretarias — só admin.
func (c *Controllers) PostSecretaria(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	prefeituraID := chi.URLParam(r, "prefeituraId")
	var corpo corpoNovaSecretaria
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" || corpo.Sigla == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome e sigla sao obrigatorios")
		return
	}
	s, err := c.Repo.CriarSecretaria(r.Context(), prefeituraID, corpo.Nome, corpo.Sigla)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, views.SecretariaParaJSON(s))
}

// GetUnidadesDaSecretaria: GET /api/v1/secretarias/{secretariaId}/unidades — admin/gestor,
// base do formulário de criação de funcionário (escolher em qual unidade ele trabalha).
func (c *Controllers) GetUnidadesDaSecretaria(w http.ResponseWriter, r *http.Request) {
	usuario, secretariaID, ok := c.exigirSecretaria(w, r)
	if !ok {
		return
	}
	if usuario.Papel != domain.PapelGestor && usuario.Papel != domain.PapelAdmin {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas gestor ou admin")
		return
	}
	unidades, err := c.Repo.ListarUnidades(r.Context(), secretariaID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(unidades))
	for i, u := range unidades {
		out[i] = map[string]any{"id": u.ID, "secretariaId": u.SecretariaID, "nome": u.Nome}
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

type corpoNovoGestor struct {
	Nome  string `json:"nome"`
	Email string `json:"email"`
	Senha string `json:"senha"`
}

// PostGestor: POST /api/v1/secretarias/{secretariaId}/gestores — só admin. Cria um gestor
// vinculado à secretaria inteira (não a uma unidade — ver skill modelo-dados).
func (c *Controllers) PostGestor(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	secretariaID := chi.URLParam(r, "secretariaId")
	if _, err := c.Repo.BuscarSecretariaPorID(r.Context(), secretariaID); err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "secretaria nao encontrada")
		return
	}
	var corpo corpoNovoGestor
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" || corpo.Email == "" || corpo.Senha == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome, email e senha sao obrigatorios")
		return
	}
	hash, err := auth.HashSenha(corpo.Senha)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	usuario, err := c.Repo.CriarUsuario(r.Context(), repository.NovoUsuario{
		Nome: corpo.Nome, Email: corpo.Email, SenhaHash: hash, Papel: domain.PapelGestor, SecretariaID: &secretariaID,
	})
	if err != nil {
		if err == repository.ErrEmailJaExiste {
			views.ResponderErro(w, http.StatusConflict, "email_existente", "ja existe um usuario com este email")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, views.UsuarioInternoParaJSON(*usuario, nil))
}

type corpoNovoFuncionario struct {
	Nome     string   `json:"nome"`
	Email    string   `json:"email"`
	Senha    string   `json:"senha"`
	Papel    string   `json:"papel"` // "atendente" ou "recepcionista"
	GradeIDs []string `json:"gradeIds"`
}

// PostFuncionario: POST /api/v1/unidades/{unidadeId}/funcionarios — admin (qualquer
// unidade) ou gestor (só dentro da própria secretaria — pedido explícito do dono do
// produto: "gestor só cria atendente e recepcionista dentro da sua secretaria"). Cria
// atendente (exige gradeId — a fila específica que ele vai chamar) ou recepcionista (sem
// grade — trabalha a unidade inteira, ver skill modelo-dados).
func (c *Controllers) PostFuncionario(w http.ResponseWriter, r *http.Request) {
	usuario := auth.UsuarioDoContexto(r.Context())
	if usuario.Papel != domain.PapelAdmin && usuario.Papel != domain.PapelGestor {
		views.ResponderErro(w, http.StatusForbidden, "papel_sem_acesso", "apenas admin ou gestor")
		return
	}
	unidadeID := chi.URLParam(r, "unidadeId")
	if _, err := c.Repo.BuscarUnidadePorID(r.Context(), unidadeID); err != nil {
		views.ResponderErro(w, http.StatusNotFound, "nao_encontrado", "unidade nao encontrada")
		return
	}
	if !c.usuarioPodeAcessarUnidade(r.Context(), usuario, unidadeID) {
		views.ResponderErro(w, http.StatusForbidden, "unidade_incorreta", "unidade fora da sua secretaria")
		return
	}
	var corpo corpoNovoFuncionario
	if err := json.NewDecoder(r.Body).Decode(&corpo); err != nil || corpo.Nome == "" || corpo.Email == "" || corpo.Senha == "" {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "nome, email e senha sao obrigatorios")
		return
	}
	var papel domain.Papel
	switch corpo.Papel {
	case "atendente":
		papel = domain.PapelAtendente
	case "recepcionista":
		papel = domain.PapelRecepcionista
	default:
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "papel deve ser 'atendente' ou 'recepcionista'")
		return
	}
	// Atendente exige pelo menos 1 grade alocada (a fila que ele vai chamar); recepcionista
	// pode ter 0+ (23/09, "atendente ou recepção podem ser alocados em mais de uma grade" —
	// sem nenhuma, a recepcionista continua vendo/cadastrando em qualquer grade da unidade).
	if papel == domain.PapelAtendente && len(corpo.GradeIDs) == 0 {
		views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "pelo menos uma grade e obrigatoria para atendente")
		return
	}
	for _, gradeID := range corpo.GradeIDs {
		grade, err := c.Repo.BuscarGradePorID(r.Context(), gradeID)
		if err != nil || grade.UnidadeID != unidadeID {
			views.ResponderErro(w, http.StatusBadRequest, "corpo_invalido", "grade nao encontrada nesta unidade")
			return
		}
	}
	hash, err := auth.HashSenha(corpo.Senha)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	novoUsuario, err := c.Repo.CriarUsuario(r.Context(), repository.NovoUsuario{
		Nome: corpo.Nome, Email: corpo.Email, SenhaHash: hash, Papel: papel, UnidadeID: &unidadeID, GradeIDs: corpo.GradeIDs,
	})
	if err != nil {
		if err == repository.ErrEmailJaExiste {
			views.ResponderErro(w, http.StatusConflict, "email_existente", "ja existe um usuario com este email")
			return
		}
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	grades, err := c.Repo.ListarGradesDoUsuario(r.Context(), novoUsuario.ID)
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusCreated, views.UsuarioInternoParaJSON(*novoUsuario, grades))
}

// GetUsuariosInternosAdmin: GET /api/v1/admin/usuarios — só admin. Tela "Usuários internos"
// consolidada (22/09) — gestor, atendente e recepcionista da plataforma inteira numa lista
// só, já com prefeitura/secretaria/unidade/grade resolvidos (sem N+1 no frontend).
func (c *Controllers) GetUsuariosInternosAdmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	lista, err := c.Repo.ListarTodosUsuariosInternos(r.Context())
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	out := make([]map[string]any, len(lista))
	for i, u := range lista {
		out[i] = views.UsuarioComContextoParaJSON(u)
	}
	views.ResponderJSON(w, http.StatusOK, out)
}

// GetResumoAdmin: GET /api/v1/admin/resumo — só admin. Contadores simples pro dashboard do
// admin — deliberadamente simples nesta rodada (a prioridade era estrutura de acesso, não
// gráficos de negócio; ver CLAUDE.md).
func (c *Controllers) GetResumoAdmin(w http.ResponseWriter, r *http.Request) {
	if _, ok := exigirAdmin(w, r); !ok {
		return
	}
	resumo, err := c.Repo.ResumoAdmin(r.Context())
	if err != nil {
		views.ResponderErro(w, http.StatusInternalServerError, "erro_interno", err.Error())
		return
	}
	views.ResponderJSON(w, http.StatusOK, resumo)
}
