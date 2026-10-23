package repository

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"painel-chamada-backend/internal/domain"
)

// Testes de integração dos 3 achados de revisão de segurança (22/10, mesmo dia da
// implementação do SSO — ver skill auth-conecta-cidades, "Implementação real"): isolamento
// de tenant em usuarios.external_sub, isolamento de tenant em mapeamento_externo, e a trava
// contra rebind silencioso de vínculo externo. Mesma filosofia de fila_regras_test.go: banco
// real, sem mocks, prefixo TESTE_AUTOMATIZADO_, limpeza total via t.Cleanup.

func TestBuscarUsuarioPorExternalSub_EscopadoPorEmissor(t *testing.T) {
	pool := abrirPoolDeTeste(t)
	r := New(pool)
	ctx := context.Background()
	sufixo := fmt.Sprintf("%d", time.Now().UnixNano())

	// Dois usuários com o MESMO external_sub ("42"), mas emissores DIFERENTES — simula duas
	// prefeituras diferentes cujos servidores coincidem no ID interno da Conecta Cidades.
	u1, err := r.UpsertUsuarioPortal(ctx, DadosPortal{
		ExternalIssuer: "https://emissor-a.teste", ExternalSub: "42",
		Nome: "TESTE_AUTOMATIZADO_SUB_A_" + sufixo, Email: "sub.a." + sufixo + "@local.dev",
		EmailVerificado: true, Papel: domain.PapelAdmin,
	})
	if err != nil {
		t.Fatalf("criar usuario A: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from usuarios where id = $1`, u1.ID) })

	u2, err := r.UpsertUsuarioPortal(ctx, DadosPortal{
		ExternalIssuer: "https://emissor-b.teste", ExternalSub: "42",
		Nome: "TESTE_AUTOMATIZADO_SUB_B_" + sufixo, Email: "sub.b." + sufixo + "@local.dev",
		EmailVerificado: true, Papel: domain.PapelAdmin,
	})
	if err != nil {
		t.Fatalf("criar usuario B: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from usuarios where id = $1`, u2.ID) })

	if u1.ID == u2.ID {
		t.Fatal("usuarios de emissores diferentes com o mesmo sub NUNCA deveriam colidir no mesmo registro")
	}

	achadoA, err := r.BuscarUsuarioPorExternalSub(ctx, "https://emissor-a.teste", "42")
	if err != nil || achadoA.ID != u1.ID {
		t.Errorf("busca por (emissor-a, 42) deveria achar u1, achou %v (err=%v)", achadoA, err)
	}
	achadoB, err := r.BuscarUsuarioPorExternalSub(ctx, "https://emissor-b.teste", "42")
	if err != nil || achadoB.ID != u2.ID {
		t.Errorf("busca por (emissor-b, 42) deveria achar u2, achou %v (err=%v)", achadoB, err)
	}
}

func TestUpsertUsuarioPortal_NaoRebindaVinculoExistente(t *testing.T) {
	pool := abrirPoolDeTeste(t)
	r := New(pool)
	ctx := context.Background()
	sufixo := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "rebind." + sufixo + "@local.dev"

	original, err := r.UpsertUsuarioPortal(ctx, DadosPortal{
		ExternalIssuer: "https://emissor.teste", ExternalSub: "sub-original",
		Nome: "TESTE_AUTOMATIZADO_REBIND_" + sufixo, Email: email,
		EmailVerificado: true, Papel: domain.PapelAdmin,
	})
	if err != nil {
		t.Fatalf("criar usuario original: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from usuarios where id = $1`, original.ID) })

	// Mesmo e-mail, `sub` DIFERENTE — simula um token (legitimamente assinado, mas com outro
	// `sub`) carregando o e-mail de alguém que já tem vínculo externo estabelecido. Nunca
	// deve rebindar silenciosamente.
	_, err = r.UpsertUsuarioPortal(ctx, DadosPortal{
		ExternalIssuer: "https://emissor.teste", ExternalSub: "sub-impostor",
		Nome: "Impostor", Email: email,
		EmailVerificado: true, Papel: domain.PapelAdmin,
	})
	if !errors.Is(err, ErrVinculoExternoConflitante) {
		t.Fatalf("esperava ErrVinculoExternoConflitante, veio: %v", err)
	}

	// Confirma que o registro original não foi alterado.
	recarregado, err := r.BuscarUsuarioPorID(ctx, original.ID)
	if err != nil {
		t.Fatalf("recarregar original: %v", err)
	}
	if recarregado.ExternalSub == nil || *recarregado.ExternalSub != "sub-original" {
		t.Errorf("external_sub do usuario original foi alterado: %v", recarregado.ExternalSub)
	}
}

func TestUpsertUsuarioPortal_FallbackEmailSoComVerificado(t *testing.T) {
	pool := abrirPoolDeTeste(t)
	r := New(pool)
	ctx := context.Background()
	sufixo := fmt.Sprintf("%d", time.Now().UnixNano())
	email := "naoverificado." + sufixo + "@local.dev"

	local, err := r.CriarUsuario(ctx, NovoUsuario{
		Nome: "TESTE_AUTOMATIZADO_LOCAL_" + sufixo, Email: email,
		Papel: domain.PapelAdmin,
	})
	if err != nil {
		t.Fatalf("criar usuario local: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from usuarios where id = $1`, local.ID) })

	// E-mail NÃO verificado no token: o fallback por e-mail nunca deve rodar — a função
	// tenta CRIAR um usuário novo com o mesmo e-mail (não vincular ao local existente), que
	// a constraint unique de `usuarios.email` recusa. Resultado esperado: erro, NUNCA um
	// login bem-sucedido herdando a conta local por um e-mail não confirmado.
	_, err = r.UpsertUsuarioPortal(ctx, DadosPortal{
		ExternalIssuer: "https://emissor.teste", ExternalSub: "sub-nao-verificado",
		Nome: "Tentativa", Email: email,
		EmailVerificado: false, Papel: domain.PapelAdmin,
	})
	if err == nil {
		t.Fatal("esperava erro (constraint de email duplicado) — e-mail nao verificado nunca deveria conseguir vincular nem criar com sucesso")
	}
}

func TestResolverSecretariaPorExternalIDs_EscopadoPorPrefeitura(t *testing.T) {
	pool := abrirPoolDeTeste(t)
	r := New(pool)
	ctx := context.Background()
	sufixo := fmt.Sprintf("%d", time.Now().UnixNano())

	prefA, err := r.CriarPrefeitura(ctx, "TESTE_AUTOMATIZADO_PREFA_"+sufixo)
	if err != nil {
		t.Fatalf("CriarPrefeitura A: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from prefeituras where id = $1`, prefA.ID) })
	secA, err := r.CriarSecretaria(ctx, prefA.ID, "TESTE_AUTOMATIZADO_SECA_"+sufixo, "TA1")
	if err != nil {
		t.Fatalf("CriarSecretaria A: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from secretarias where id = $1`, secA.ID) })

	prefB, err := r.CriarPrefeitura(ctx, "TESTE_AUTOMATIZADO_PREFB_"+sufixo)
	if err != nil {
		t.Fatalf("CriarPrefeitura B: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from prefeituras where id = $1`, prefB.ID) })
	secB, err := r.CriarSecretaria(ctx, prefB.ID, "TESTE_AUTOMATIZADO_SECB_"+sufixo, "TA2")
	if err != nil {
		t.Fatalf("CriarSecretaria B: %v", err)
	}
	t.Cleanup(func() { pool.Exec(ctx, `delete from secretarias where id = $1`, secB.ID) })

	// MESMO external_id ("3") mapeado em CADA prefeitura pra uma secretaria DIFERENTE —
	// exatamente o cenário que motivou o fix (department_id é um ID interno por-município).
	if _, err := pool.Exec(ctx, `insert into mapeamento_externo (tipo, external_id, secretaria_id, prefeitura_id) values ('department', '3', $1, $2)`, secA.ID, prefA.ID); err != nil {
		t.Fatalf("inserir mapeamento A: %v", err)
	}
	t.Cleanup(func() {
		pool.Exec(ctx, `delete from mapeamento_externo where secretaria_id = any($1)`, []string{secA.ID, secB.ID})
	})
	if _, err := pool.Exec(ctx, `insert into mapeamento_externo (tipo, external_id, secretaria_id, prefeitura_id) values ('department', '3', $1, $2)`, secB.ID, prefB.ID); err != nil {
		t.Fatalf("inserir mapeamento B: %v", err)
	}

	resolvidoA, err := r.ResolverSecretariaPorExternalIDs(ctx, prefA.ID, []int{3})
	if err != nil || resolvidoA == nil || *resolvidoA != secA.ID {
		t.Errorf("resolver department_id=3 sob prefeitura A deveria achar secA (%s), achou %v (err=%v)", secA.ID, resolvidoA, err)
	}
	resolvidoB, err := r.ResolverSecretariaPorExternalIDs(ctx, prefB.ID, []int{3})
	if err != nil || resolvidoB == nil || *resolvidoB != secB.ID {
		t.Errorf("resolver department_id=3 sob prefeitura B deveria achar secB (%s), achou %v (err=%v)", secB.ID, resolvidoB, err)
	}
}
