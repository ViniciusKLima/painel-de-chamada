package repository

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"painel-chamada-backend/internal/domain"
)

// Testes de integração das regras de negócio mais frágeis da fila (cooldown/órfão, prioridade
// na sala de espera, ausência automática) e da reativação (23/09) — rodam contra um Postgres
// real (mesmo banco de dev usado no resto do projeto, sem mocks, mesma filosofia de "testar
// com dado real" já seguida manualmente via curl/navegador ao longo da sessão). Cada teste cria
// sua própria unidade/grade/usuários com nomes prefixados "TESTE_AUTOMATIZADO_" e remove tudo
// via t.Cleanup, pra nunca deixar lixo no banco de dev nem colidir com dado real.
//
// Rodar: vá pra backend/ e rode `go test ./internal/repository/...` com o Postgres do
// docker-compose no ar (DATABASE_URL aponta pro mesmo default do cmd/api/main.go).

func abrirPoolDeTeste(t *testing.T) *pgxpool.Pool {
	t.Helper()
	url := os.Getenv("DATABASE_URL")
	if url == "" {
		url = "postgres://painel:painel_dev_local@localhost:5433/painel_chamada"
	}
	pool, err := pgxpool.New(context.Background(), url)
	if err != nil {
		t.Fatalf("falha ao conectar no banco de teste: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		t.Skipf("Postgres de dev indisponível em %s (suba com `docker-compose up -d`): %v", url, err)
	}
	t.Cleanup(pool.Close)
	return pool
}

// fixture agrupa uma unidade/grade/dois atendentes descartáveis, prontos pra testar a fila.
type fixture struct {
	r            *Repo
	pool         *pgxpool.Pool
	unidadeID    string
	gradeID      string
	guicheID     string
	usuarioA     string
	usuarioB     string
	prefeituraID string
	secretariaID string
}

func criarFixture(t *testing.T) *fixture {
	t.Helper()
	pool := abrirPoolDeTeste(t)
	r := New(pool)
	ctx := context.Background()
	sufixo := fmt.Sprintf("%d", time.Now().UnixNano())

	pref, err := r.CriarPrefeitura(ctx, "TESTE_AUTOMATIZADO_PREF_"+sufixo)
	if err != nil {
		t.Fatalf("CriarPrefeitura: %v", err)
	}
	sec, err := r.CriarSecretaria(ctx, pref.ID, "TESTE_AUTOMATIZADO_SEC_"+sufixo, "TA"+sufixo[len(sufixo)-4:])
	if err != nil {
		t.Fatalf("CriarSecretaria: %v", err)
	}
	unidade, err := r.ResolverOuCriarUnidade(ctx, sec.ID, "TESTE_AUTOMATIZADO_UNI_"+sufixo)
	if err != nil {
		t.Fatalf("ResolverOuCriarUnidade: %v", err)
	}
	grade, err := r.ResolverOuCriarGrade(ctx, unidade.ID, "TESTE_AUTOMATIZADO_SERVICO_"+sufixo, "TESTE_AUTOMATIZADO_GRADE_"+sufixo)
	if err != nil {
		t.Fatalf("ResolverOuCriarGrade: %v", err)
	}
	guiches, err := r.ListarGuichesAtivos(ctx, grade.ID)
	if err != nil || len(guiches) == 0 {
		t.Fatalf("grade nova deveria ter nascido com 1 guichê padrão: %v", err)
	}

	hash := "$2a$10$fakehashsoparatestenaoehusadoparalogin........................"
	usuarioA, err := r.CriarUsuario(ctx, NovoUsuario{
		Nome: "TESTE_AUTOMATIZADO_ATENDENTE_A", Email: "teste.auto.a." + sufixo + "@local.dev",
		SenhaHash: hash, Papel: domain.PapelAtendente, UnidadeID: &unidade.ID, GradeIDs: []string{grade.ID},
	})
	if err != nil {
		t.Fatalf("CriarUsuario A: %v", err)
	}
	usuarioB, err := r.CriarUsuario(ctx, NovoUsuario{
		Nome: "TESTE_AUTOMATIZADO_ATENDENTE_B", Email: "teste.auto.b." + sufixo + "@local.dev",
		SenhaHash: hash, Papel: domain.PapelAtendente, UnidadeID: &unidade.ID, GradeIDs: []string{grade.ID},
	})
	if err != nil {
		t.Fatalf("CriarUsuario B: %v", err)
	}

	f := &fixture{
		r: r, pool: pool, unidadeID: unidade.ID, gradeID: grade.ID, guicheID: guiches[0].ID,
		usuarioA: usuarioA.ID, usuarioB: usuarioB.ID, prefeituraID: pref.ID, secretariaID: sec.ID,
	}
	t.Cleanup(func() { f.limpar(t) })
	return f
}

func (f *fixture) limpar(t *testing.T) {
	t.Helper()
	ctx := context.Background()
	// Ordem importa: filho antes do pai (chamadas -> agendamentos -> usuario_grades ->
	// guiches -> grades -> usuarios -> unidades -> secretarias -> prefeituras).
	exec := func(sql string, args ...any) {
		if _, err := f.pool.Exec(ctx, sql, args...); err != nil {
			t.Logf("limpeza de fixture: %q falhou: %v (não fatal, só lixo de teste sobrando)", sql, err)
		}
	}
	exec(`delete from chamadas where agendamento_id in (select id from agendamentos where unidade_id = $1)`, f.unidadeID)
	exec(`delete from agendamentos where unidade_id = $1`, f.unidadeID)
	exec(`delete from usuario_grades where usuario_id = any($1)`, []string{f.usuarioA, f.usuarioB})
	exec(`delete from guiches where grade_id = $1`, f.gradeID)
	exec(`delete from grades where unidade_id = $1`, f.unidadeID)
	exec(`delete from usuarios where id = any($1)`, []string{f.usuarioA, f.usuarioB})
	exec(`delete from unidades where id = $1`, f.unidadeID)
	exec(`delete from secretarias where id = $1`, f.secretariaID)
	exec(`delete from prefeituras where id = $1`, f.prefeituraID)
}

// inserirEncaixe cria um agendamento tipo encaixe direto em sala_espera, com chegada_em e
// prioridade controlados manualmente — `CriarAgendamento` não expõe esses dois campos (não
// precisa, no fluxo real eles vêm de ConfirmarChegada/PostEncaixe), mas os testes de
// prioridade/ausência precisam de controle fino sobre eles.
func (f *fixture) inserirAgendamento(t *testing.T, protocolo string, status domain.StatusAgendamento, prioridade *string, chegadaEm, horarioPrevisto *time.Time) string {
	t.Helper()
	var id string
	err := f.pool.QueryRow(context.Background(), `
		insert into agendamentos (unidade_id, grade_id, protocolo, nome_cidadao, tipo, status, prioridade, chegada_em, horario_previsto)
		values ($1, $2, $3, $4, 'encaixe', $5, $6, $7, $8)
		returning id
	`, f.unidadeID, f.gradeID, protocolo, "TESTE_AUTOMATIZADO_CIDADAO_"+protocolo, status, prioridade, chegadaEm, horarioPrevisto).Scan(&id)
	if err != nil {
		t.Fatalf("inserirAgendamento(%s): %v", protocolo, err)
	}
	return id
}

// --- Prioridade na sala de espera ---

func TestSalaDeEsperaPrioridadeNoHorarioPrimeiro(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()
	agora := time.Now()
	noHorario := "no_horario"
	atrasado := "atrasado"

	// Chegou primeiro (mais cedo), mas é atrasado — deve ficar DEPOIS do no_horario mesmo
	// assim (regra: no_horario sempre primeiro, independente de quem chegou antes).
	idAtrasado := f.inserirAgendamento(t, "ATR1", domain.StatusSalaEspera, &atrasado, ptr(agora.Add(-10*time.Minute)), nil)
	idNoHorario := f.inserirAgendamento(t, "NOH1", domain.StatusSalaEspera, &noHorario, ptr(agora.Add(-1*time.Minute)), ptr(agora))
	// Encaixe sem prioridade nenhuma marcada (nil) — trata como "não é no_horario", compete
	// com atrasado só pela ordem de chegada.
	idEncaixe := f.inserirAgendamento(t, "ENC1", domain.StatusSalaEspera, nil, ptr(agora.Add(-5*time.Minute)), nil)

	lista, err := f.r.ListarSalaEsperaPorGrades(ctx, []string{f.gradeID})
	if err != nil {
		t.Fatalf("ListarSalaEsperaPorGrades: %v", err)
	}
	if len(lista) != 3 {
		t.Fatalf("esperava 3 na sala de espera, veio %d", len(lista))
	}
	if lista[0].ID != idNoHorario {
		t.Errorf("no_horario deveria vir primeiro, veio %s (protocolo %s)", lista[0].ID, lista[0].Protocolo)
	}
	// Entre atrasado (chegou há 10min) e encaixe (chegou há 5min), quem chegou antes (o
	// atrasado) vem antes — competem só por ordem de chegada.
	if lista[1].ID != idAtrasado || lista[2].ID != idEncaixe {
		t.Errorf("esperava [atrasado, encaixe] depois do no_horario por ordem de chegada, veio [%s, %s]", lista[1].Protocolo, lista[2].Protocolo)
	}
}

// --- Cooldown e órfãos, escopados por grade (auditoria 23/09) ---

func TestCooldownBloqueiaChamarNovoAteExpirar(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()
	agora := time.Now()
	f.inserirAgendamento(t, "COOL1", domain.StatusSalaEspera, nil, ptr(agora), nil)
	f.inserirAgendamento(t, "COOL2", domain.StatusSalaEspera, nil, ptr(agora.Add(time.Second)), nil)

	cooldown := 3 // minutos
	_, _, err := f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown)
	if err != nil {
		t.Fatalf("primeira chamada deveria funcionar: %v", err)
	}

	// Ainda dentro do cooldown — chamar outro deve ser bloqueado com ErrChamadaPendente.
	_, _, err = f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown)
	if err != ErrChamadaPendente {
		t.Fatalf("esperava ErrChamadaPendente dentro do cooldown, veio: %v", err)
	}

	pendente, err := f.r.TemChamadaPendenteNaoRechamada(ctx, f.gradeID, f.usuarioA, cooldown)
	if err != nil {
		t.Fatalf("TemChamadaPendenteNaoRechamada: %v", err)
	}
	if !pendente {
		t.Fatal("deveria estar pendente ainda dentro do cooldown")
	}

	// Simula o cooldown já ter passado, sem precisar esperar de verdade (backdate direto do
	// chamado_em da linha de `chamadas`, único jeito de "voltar no tempo" num teste síncrono).
	if _, err := f.pool.Exec(ctx, `update chamadas set chamado_em = now() - interval '10 minutes' where usuario_id = $1`, f.usuarioA); err != nil {
		t.Fatalf("backdate do chamado_em: %v", err)
	}

	pendente, err = f.r.TemChamadaPendenteNaoRechamada(ctx, f.gradeID, f.usuarioA, cooldown)
	if err != nil {
		t.Fatalf("TemChamadaPendenteNaoRechamada apos backdate: %v", err)
	}
	if pendente {
		t.Fatal("nao deveria mais estar pendente depois do cooldown passar")
	}

	// Agora deve conseguir chamar outra pessoa — a primeira vira órfã (rechamou zero vezes
	// ainda, então nem é órfã de verdade até rechamar pelo menos uma vez — ver próximo teste).
	_, _, err = f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown)
	if err != nil {
		t.Fatalf("apos cooldown expirar deveria conseguir chamar outra pessoa: %v", err)
	}
}

func TestOrfaoBloqueiaGradeInteiraAteAssumir(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()
	agora := time.Now()
	f.inserirAgendamento(t, "ORF1", domain.StatusSalaEspera, nil, ptr(agora), nil)
	f.inserirAgendamento(t, "ORF2", domain.StatusSalaEspera, nil, ptr(agora.Add(time.Second)), nil)
	f.inserirAgendamento(t, "ORF3", domain.StatusSalaEspera, nil, ptr(agora.Add(2*time.Second)), nil)

	cooldown := 1
	primeiro, _, err := f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown)
	if err != nil {
		t.Fatalf("chamar primeiro: %v", err)
	}

	// Backdate pra poder rechamar (cooldown de 1min já passado) e rechama — agora
	// totalChamadasDoDono = 2, condição de órfão ainda falta (precisa seguir em frente).
	if _, err := f.pool.Exec(ctx, `update chamadas set chamado_em = now() - interval '10 minutes' where agendamento_id = $1`, primeiro.ID); err != nil {
		t.Fatalf("backdate: %v", err)
	}
	if _, err := f.r.Rechamar(ctx, primeiro.ID, f.usuarioA); err != nil {
		t.Fatalf("rechamar: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `update chamadas set chamado_em = now() - interval '10 minutes' where agendamento_id = $1`, primeiro.ID); err != nil {
		t.Fatalf("backdate pos-rechamada: %v", err)
	}

	// Segue em frente pra outra pessoa — o primeiro agendamento vira órfão de verdade agora.
	if _, _, err := f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown); err != nil {
		t.Fatalf("chamar segundo (abandonando o primeiro): %v", err)
	}

	existeOrfao, err := f.r.ExistemChamadosOrfaos(ctx, f.gradeID)
	if err != nil {
		t.Fatalf("ExistemChamadosOrfaos: %v", err)
	}
	if !existeOrfao {
		t.Fatal("deveria existir um órfão depois de rechamar e seguir em frente")
	}

	// Um SEGUNDO atendente, sem nenhuma pendência própria, deve ser bloqueado de chamar
	// gente nova enquanto o órfão não for assumido (regra de grade inteira).
	_, _, err = f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioB, cooldown)
	if err != ErrExistemOrfaos {
		t.Fatalf("esperava ErrExistemOrfaos bloqueando o segundo atendente, veio: %v", err)
	}

	// Assumir o órfão libera a grade de novo.
	if _, _, err := f.r.AssumirChamada(ctx, f.gradeID, primeiro.ID, f.guicheID, f.usuarioB); err != nil {
		t.Fatalf("AssumirChamada: %v", err)
	}
	existeOrfao, err = f.r.ExistemChamadosOrfaos(ctx, f.gradeID)
	if err != nil {
		t.Fatalf("ExistemChamadosOrfaos apos assumir: %v", err)
	}
	if existeOrfao {
		t.Fatal("nao deveria mais existir orfao depois de assumido")
	}
}

// --- Ausência automática (sweeper) ---

func TestSweepAusenciaAutomaticaChegadaEAtendimento(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()
	agora := time.Now()

	// SLA padrão da grade nova: chegada 60min, atendimento 10min (defaults pós rodada 28).
	// Estágio 1 — agendado que nunca confirmou chegada, horário previsto 2h atrás: deve virar
	// ausente pelo estágio de CHEGADA.
	idChegada := f.inserirAgendamento(t, "SWEEP_CHEG", domain.StatusAguardandoChegada, nil, nil, ptr(agora.Add(-2*time.Hour)))
	if _, err := f.pool.Exec(ctx, `update agendamentos set tipo = 'agendado' where id = $1`, idChegada); err != nil {
		t.Fatalf("forcar tipo agendado: %v", err)
	}

	// Estágio 2 — alguém chamado há 30min (SLA de atendimento é 10min): deve virar ausente
	// pelo estágio de ATENDIMENTO.
	idAtendimento := f.inserirAgendamento(t, "SWEEP_ATEND", domain.StatusSalaEspera, nil, ptr(agora), nil)
	cooldown := 1
	if _, _, err := f.r.ChamarProximo(ctx, f.gradeID, f.guicheID, f.usuarioA, cooldown); err != nil {
		t.Fatalf("chamar pra depois forcar ausencia por atendimento: %v", err)
	}
	if _, err := f.pool.Exec(ctx, `update chamadas set chamado_em = now() - interval '30 minutes' where agendamento_id = $1`, idAtendimento); err != nil {
		t.Fatalf("backdate chamado_em: %v", err)
	}

	chegada, atendimento, err := f.r.SweepAusenciaAutomatica(ctx, agora)
	if err != nil {
		t.Fatalf("SweepAusenciaAutomatica: %v", err)
	}
	if chegada < 1 {
		t.Errorf("esperava pelo menos 1 ausência por chegada, veio %d", chegada)
	}
	if atendimento < 1 {
		t.Errorf("esperava pelo menos 1 ausência por atendimento, veio %d", atendimento)
	}

	var statusChegada, statusAtendimento string
	if err := f.pool.QueryRow(ctx, `select status from agendamentos where id = $1`, idChegada).Scan(&statusChegada); err != nil {
		t.Fatalf("select status chegada: %v", err)
	}
	if err := f.pool.QueryRow(ctx, `select status from agendamentos where id = $1`, idAtendimento).Scan(&statusAtendimento); err != nil {
		t.Fatalf("select status atendimento: %v", err)
	}
	if statusChegada != string(domain.StatusAusente) {
		t.Errorf("agendamento de chegada deveria estar ausente, está %q", statusChegada)
	}
	if statusAtendimento != string(domain.StatusAusente) {
		t.Errorf("agendamento de atendimento deveria estar ausente, está %q", statusAtendimento)
	}
}

// --- Multi-grade: independência entre grades (auditoria 23/09) ---

func TestChamadaPendenteEmUmaGradeNaoBloqueiaOutra(t *testing.T) {
	f1 := criarFixture(t)
	f2 := criarFixture(t)
	ctx := context.Background()
	agora := time.Now()
	f1.inserirAgendamento(t, "MG1", domain.StatusSalaEspera, nil, ptr(agora), nil)
	f2.inserirAgendamento(t, "MG2", domain.StatusSalaEspera, nil, ptr(agora), nil)

	cooldown := 5
	// mesmo usuário (usuarioA de f1) não existe na grade de f2 de verdade, mas ChamarProximo
	// não valida alocação (isso é feito na camada de handler via exigirGrade) — o ponto aqui é
	// só confirmar que a checagem de pendência no banco é por grade_id, não usuario_id sozinho.
	if _, _, err := f1.r.ChamarProximo(ctx, f1.gradeID, f1.guicheID, f1.usuarioA, cooldown); err != nil {
		t.Fatalf("chamar na grade 1: %v", err)
	}
	// Mesmo usuário, grade DIFERENTE (f2) — não deve ser bloqueado pela pendência da grade 1.
	if _, _, err := f2.r.ChamarProximo(ctx, f2.gradeID, f2.guicheID, f1.usuarioA, cooldown); err != nil {
		t.Fatalf("chamar na grade 2 com o mesmo usuario da grade 1 nao deveria ser bloqueado: %v", err)
	}
}

// --- Reativação (23/09, auditoria de reativação) ---

func TestExcluirGradeLimpaUsuarioGrades(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()

	var totalAntes int
	if err := f.pool.QueryRow(ctx, `select count(*) from usuario_grades where grade_id = $1`, f.gradeID).Scan(&totalAntes); err != nil {
		t.Fatalf("count antes: %v", err)
	}
	if totalAntes == 0 {
		t.Fatal("fixture deveria ter alocado usuarioA/usuarioB na grade")
	}

	if err := f.r.ExcluirGrade(ctx, f.gradeID); err != nil {
		t.Fatalf("ExcluirGrade: %v", err)
	}

	var totalDepois int
	if err := f.pool.QueryRow(ctx, `select count(*) from usuario_grades where grade_id = $1`, f.gradeID).Scan(&totalDepois); err != nil {
		t.Fatalf("count depois: %v", err)
	}
	if totalDepois != 0 {
		t.Fatalf("ExcluirGrade deveria zerar usuario_grades da grade excluida, restaram %d", totalDepois)
	}

	grades, err := f.r.ListarGradesDoUsuario(ctx, f.usuarioA)
	if err != nil {
		t.Fatalf("ListarGradesDoUsuario: %v", err)
	}
	for _, g := range grades {
		if g.ID == f.gradeID {
			t.Fatal("usuario nao deveria mais listar a grade excluida como alocada")
		}
	}
}

func TestReativarGradeUsuarioEGuiche(t *testing.T) {
	f := criarFixture(t)
	ctx := context.Background()

	// Grade
	if err := f.r.ExcluirGrade(ctx, f.gradeID); err != nil {
		t.Fatalf("ExcluirGrade: %v", err)
	}
	g, err := f.r.ReativarGrade(ctx, f.gradeID)
	if err != nil {
		t.Fatalf("ReativarGrade: %v", err)
	}
	if g.Excluida {
		t.Error("grade reativada nao deveria vir com Excluida=true")
	}

	// Usuário
	if err := f.r.ExcluirUsuario(ctx, f.usuarioA); err != nil {
		t.Fatalf("ExcluirUsuario: %v", err)
	}
	usuarioExcluido, err := f.r.BuscarUsuarioPorID(ctx, f.usuarioA)
	if err != nil {
		t.Fatalf("BuscarUsuarioPorID: %v", err)
	}
	if usuarioExcluido.Ativo {
		t.Fatal("usuario deveria estar inativo apos ExcluirUsuario")
	}
	if err := f.r.ReativarUsuario(ctx, f.usuarioA); err != nil {
		t.Fatalf("ReativarUsuario: %v", err)
	}
	usuarioReativado, err := f.r.BuscarUsuarioPorID(ctx, f.usuarioA)
	if err != nil {
		t.Fatalf("BuscarUsuarioPorID apos reativar: %v", err)
	}
	if !usuarioReativado.Ativo {
		t.Fatal("usuario deveria estar ativo apos ReativarUsuario")
	}

	// Guichê (precisa de um segundo pra poder excluir o primeiro sem bater no ultimo_guiche)
	segundo, err := f.r.CriarGuiche(ctx, f.gradeID, NovoGuicheOuSala{Nome: "TESTE_AUTOMATIZADO_GUICHE_2", Tipo: domain.TipoGuicheGuiche, Capacidade: 1})
	if err != nil {
		t.Fatalf("CriarGuiche: %v", err)
	}
	if err := f.r.ExcluirGuiche(ctx, f.gradeID, f.guicheID); err != nil {
		t.Fatalf("ExcluirGuiche: %v", err)
	}
	guicheReativado, err := f.r.ReativarGuiche(ctx, f.gradeID, f.guicheID)
	if err != nil {
		t.Fatalf("ReativarGuiche: %v", err)
	}
	if guicheReativado.Excluida {
		t.Error("guiche reativado nao deveria vir com Excluida=true")
	}
	_ = segundo
}

func ptr[T any](v T) *T { return &v }
