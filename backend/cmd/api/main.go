package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"painel-chamada-backend/internal/auth"
	"painel-chamada-backend/internal/controllers"
	"painel-chamada-backend/internal/painel"
	"painel-chamada-backend/internal/repository"
)

func main() {
	ctx := context.Background()

	databaseURL := envOu("DATABASE_URL", "postgres://painel:painel_dev_local@localhost:5433/painel_chamada")
	sessaoSegredo := []byte(envOu("SESSAO_SEGREDO", "troque-por-um-segredo-local-qualquer-32-chars"))
	cookieSeguro := envOu("APP_ENV", "development") != "development"

	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		log.Fatalf("falha ao conectar no banco: %v", err)
	}
	defer pool.Close()

	repo := repository.New(pool)
	gerenciadorPainel := painel.NovoGerenciador()

	c := &controllers.Controllers{
		Repo:          repo,
		SessaoSegredo: sessaoSegredo,
		SessaoDuracao: 8 * time.Hour,
		CookieSeguro:  cookieSeguro,
		Painel:        gerenciadorPainel,
	}

	iniciarSweeperAusencia(ctx, repo)

	router := chi.NewRouter()

	router.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		var ok int
		err := pool.QueryRow(r.Context(), "SELECT 1").Scan(&ok)
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"status": "ok", "db": err == nil && ok == 1})
	})

	router.Route("/api/v1", func(api chi.Router) {
		api.Post("/login", c.PostLogin)
		api.Get("/painel/{gradeId}", c.GetPainel)                                      // público, sem sessão — ver skill painel-tv
		api.Get("/unidades/{unidadeId}/paineis-publico", c.GetPaineisPublicoDaUnidade) // público, kiosk de TV do local (23/09)

		api.Group(func(protegido chi.Router) {
			protegido.Use(auth.Middleware(sessaoSegredo, repo))

			protegido.Get("/sessao", c.GetSessao)
			protegido.Delete("/sessao", c.DeleteSessao)

			protegido.Post("/secretarias/{secretariaId}/lotes/preview", c.PostPreviewPlanilha)
			protegido.Post("/secretarias/{secretariaId}/lotes", c.PostImportarPlanilha)

			// Gestor deixou de ser vinculado a uma unidade só — dashboard, grade de horário e
			// usuários internos são escopados pela SECRETARIA inteira agora (22/09, ver skill
			// modelo-dados: "Admin -> Prefeitura -> Secretaria -> Usuarios").
			protegido.Get("/secretarias/{secretariaId}/dashboard", c.GetDashboard)
			protegido.Get("/secretarias/{secretariaId}/grades", c.GetGrades)
			protegido.Get("/secretarias/{secretariaId}/usuarios", c.GetUsuarios)
			protegido.Get("/secretarias/{secretariaId}/unidades", c.GetUnidadesDaSecretaria)

			// Recepção trabalha na UNIDADE inteira (todas as grades/serviços daquele local
			// físico) — ver skill modelo-dados, decisão 22/09.
			protegido.Get("/unidades/{unidadeId}/recepcao", c.GetRecepcao)
			protegido.Post("/unidades/{unidadeId}/encaixe", c.PostEncaixe)
			protegido.Get("/unidades/{unidadeId}/grades", c.GetGradesDaUnidade)

			// Consolidado de TODAS as grades do atendente logado (23/09, auditoria de
			// multi-grade — "não quero uma interface em que o atendente precise selecionar
			// uma grade"). Registradas ANTES de "/grades/{gradeId}/..." de propósito — não
			// que colida (chi resolve rota estática antes de parametrizada de qualquer
			// forma), só pra ficarem juntas no arquivo por contexto.
			protegido.Get("/atendente/grades", c.GetMinhasGrades)
			protegido.Get("/atendente/grades/guiches", c.GetMinhasGradesGuiches)
			protegido.Get("/atendente/grades/sala-espera", c.GetMinhasGradesSalaEspera)
			protegido.Get("/atendente/grades/chamados", c.GetMinhasGradesChamados)
			protegido.Get("/atendente/grades/atendidos-hoje", c.GetMinhasGradesAtendidosHoje)
			protegido.Get("/atendente/grades/ausentes-hoje", c.GetMinhasGradesAusentesHoje)

			// Atendente e configuração de fila trabalham por GRADE (um serviço específico
			// dentro da unidade), não mais pela unidade inteira.
			protegido.Get("/grades/{gradeId}/guiches", c.GetGuiches)
			protegido.Post("/grades/{gradeId}/guiches/{guicheId}/ocupar", c.PostOcuparGuiche)
			protegido.Post("/grades/{gradeId}/guiches/{guicheId}/liberar", c.PostLiberarGuiche)
			protegido.Post("/grades/{gradeId}/guiches", c.PostGuiche)
			protegido.Patch("/grades/{gradeId}/guiches/{guicheId}", c.PatchGuiche)
			protegido.Put("/grades/{gradeId}/guiches/{guicheId}", c.PutGuiche)
			protegido.Delete("/grades/{gradeId}/guiches/{guicheId}", c.DeleteGuiche)
			// Reativação (23/09, auditoria de reativação) — desfaz uma exclusão que antes não
			// tinha volta nenhuma na UI, só mexendo direto no banco.
			protegido.Post("/grades/{gradeId}/guiches/{guicheId}/reativar", c.PostReativarGuiche)
			protegido.Get("/grades/{gradeId}/configuracao", c.GetConfiguracaoGrade)
			protegido.Patch("/grades/{gradeId}/configuracao", c.PatchConfiguracaoGrade)
			protegido.Delete("/grades/{gradeId}", c.DeleteGrade)
			protegido.Post("/grades/{gradeId}/reativar", c.PostReativarGrade)
			protegido.Get("/grades/{gradeId}/sala-espera", c.GetSalaEspera)
			protegido.Get("/grades/{gradeId}/chamados", c.GetChamados)
			protegido.Get("/grades/{gradeId}/atendidos-hoje", c.GetAtendidosHoje)
			protegido.Get("/grades/{gradeId}/ausentes-hoje", c.GetAusentesHoje)
			protegido.Post("/grades/{gradeId}/guiches/{guicheId}/chamar-proximo", c.PostChamarProximo)

			protegido.Post("/agendamentos/{id}/confirmar-chegada", c.PostConfirmarChegada)
			protegido.Post("/agendamentos/{id}/desfazer-chegada", c.PostDesfazerChegada)
			protegido.Post("/agendamentos/{id}/rechamar", c.PostRechamar)
			protegido.Post("/agendamentos/{id}/atendido", c.PostAtendido)
			protegido.Post("/agendamentos/{id}/ausencia", c.PostAusenciaManual)
			protegido.Post("/agendamentos/{id}/assumir", c.PostAssumirChamada)
			protegido.Get("/agendamentos/{id}/historico", c.GetHistorico)

			protegido.Patch("/usuarios/{usuarioId}", c.PatchUsuario)
			protegido.Delete("/usuarios/{usuarioId}", c.DeleteUsuario)
			protegido.Post("/usuarios/{usuarioId}/reativar", c.PostReativarUsuario)

			// Admin (22/09): master da plataforma inteira, sem vinculo de tenant. Hierarquia
			// Admin -> Prefeitura -> Secretaria -> Usuarios, ver skill modelo-dados.
			protegido.Get("/admin/resumo", c.GetResumoAdmin)
			protegido.Get("/admin/usuarios", c.GetUsuariosInternosAdmin)
			protegido.Get("/prefeituras", c.GetPrefeituras)
			protegido.Post("/prefeituras", c.PostPrefeitura)
			protegido.Get("/prefeituras/{prefeituraId}", c.GetPrefeitura)
			protegido.Patch("/prefeituras/{prefeituraId}", c.PatchPrefeitura)
			protegido.Post("/prefeituras/{prefeituraId}/secretarias", c.PostSecretaria)
			// Abas da configuração da prefeitura, no admin (23/09) — dashboard/usuários/
			// painéis escopados a UMA prefeitura só, ver skill papeis-e-telas.
			protegido.Get("/prefeituras/{prefeituraId}/dashboard", c.GetDashboardPrefeitura)
			protegido.Get("/prefeituras/{prefeituraId}/usuarios", c.GetUsuariosPrefeitura)
			protegido.Get("/prefeituras/{prefeituraId}/paineis", c.GetPaineisPrefeitura)
			protegido.Post("/secretarias/{secretariaId}/gestores", c.PostGestor)
			protegido.Post("/unidades/{unidadeId}/funcionarios", c.PostFuncionario)
		})
	})

	port := envOu("PORT", "3333")
	appEnv := envOu("APP_ENV", "development")
	origensPermitidas := envOu("CORS_ALLOWED_ORIGINS", "")
	log.Printf("Servidor rodando na porta %s", port)
	if err := http.ListenAndServe(":"+port, cors(appEnv, origensPermitidas)(router)); err != nil {
		log.Fatal(err)
	}
}

// iniciarSweeperAusencia roda em background, varrendo os dois estágios de timeout
// (chegada e atendimento) periodicamente — decisão confirmada 19/09: ambos viram
// ausente automático, sem ação manual. Ver skill regras-negocio-fila.
func iniciarSweeperAusencia(ctx context.Context, repo *repository.Repo) {
	go func() {
		ticker := time.NewTicker(20 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			chegada, atendimento, err := repo.SweepAusenciaAutomatica(ctx, time.Now())
			if err != nil {
				log.Printf("sweeper de ausencia: erro: %v", err)
				continue
			}
			if chegada > 0 || atendimento > 0 {
				log.Printf("sweeper de ausencia: %d por falta de chegada, %d por falta de atendimento", chegada, atendimento)
			}
		}
	}()
}

func envOu(chave, padrao string) string {
	if v := os.Getenv(chave); v != "" {
		return v
	}
	return padrao
}

// regexOrigemDevLocal (23/09, auditoria de segurança — CORS estava refletindo QUALQUER
// Origin de volta junto com `Allow-Credentials: true`, o que deixa a sessão (cookie
// HttpOnly) exposta a qualquer site que fizesse uma requisição contra a API a partir do
// navegador de um usuário logado — é exatamente a combinação perigosa "origem refletida +
// credenciais permitidas"). Em desenvolvimento continuamos aceitando qualquer host
// (localhost, 127.0.0.1 ou o IP da rede local) nas portas padrão do Vite — é o que já
// permitia acessar o front por `localhost` ou pelo IP da máquina sem reconfigurar nada (ver
// "Bug real: login voltava sozinho" em CLAUDE.md, 20/09) — mas nunca um domínio arbitrário.
var regexOrigemDevLocal = regexp.MustCompile(`^https?://(localhost|127\.0\.0\.1|\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}):(5173|4173)$`)

// cors substitui o antigo `corsSimples`, que aceitava e refletia qualquer `Origin` de volta.
// Fora de desenvolvimento (`appEnv != "development"`), só origens explicitamente listadas em
// `CORS_ALLOWED_ORIGINS` (separadas por vírgula) são aceitas — vazio por padrão, ou seja,
// nenhuma origem passa até alguém configurar essa variável no deploy real (falha segura, não
// permissiva). Uma origem fora da lista simplesmente não recebe os headers de CORS — o
// navegador do cliente bloqueia a leitura da resposta sozinho, sem precisar de um 403 manual.
func cors(appEnv, origensPermitidasCSV string) func(http.Handler) http.Handler {
	permitidas := map[string]bool{}
	for _, o := range strings.Split(origensPermitidasCSV, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			permitidas[o] = true
		}
	}
	emDev := appEnv == "development"
	origemPermitida := func(origem string) bool {
		if origem == "" {
			return false
		}
		if permitidas[origem] {
			return true
		}
		return emDev && regexOrigemDevLocal.MatchString(origem)
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origem := r.Header.Get("Origin")
			if origemPermitida(origem) {
				w.Header().Set("Access-Control-Allow-Origin", origem)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
				w.Header().Set("Access-Control-Allow-Methods", "GET,POST,PUT,PATCH,DELETE,OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
			}
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
