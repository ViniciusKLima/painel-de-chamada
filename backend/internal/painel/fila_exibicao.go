// Pacote painel implementa a fila de EXIBIÇÃO do painel de TV — decisão de arquitetura
// de 19/09 (ver skill painel-tv): efêmera, em memória, sem tabela no Postgres. A tabela
// `chamadas` continua sendo a fonte de verdade; isto aqui é só "o que mostrar agora, e
// por quanto tempo", reconstruível a qualquer momento a partir do histórico de chamadas.
// Se o processo reiniciar, essa fila se perde — aceitável, o pior caso é uma chamada
// aparecer por menos tempo que o normal.
package painel

import (
	"sync"
	"time"
)

type ItemExibicao struct {
	ChamadaID  string
	Protocolo  string
	Nome       string
	GuicheNome string
}

type filaDaUnidade struct {
	pendentes []ItemExibicao
	atual     *ItemExibicao
	expiraEm  time.Time
	duracao   time.Duration
}

type GerenciadorDeFilas struct {
	mu    sync.Mutex
	filas map[string]*filaDaUnidade
}

func NovoGerenciador() *GerenciadorDeFilas {
	return &GerenciadorDeFilas{filas: map[string]*filaDaUnidade{}}
}

// Empurrar registra uma nova chamada/rechamada pra entrar na fila de exibição daquela
// unidade — chamar isso é a única integração que os handlers de chamar/rechamar
// precisam fazer com este pacote.
func (g *GerenciadorDeFilas) Empurrar(unidadeID string, item ItemExibicao) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f := g.fila(unidadeID)
	f.pendentes = append(f.pendentes, item)
}

// Atual retorna o que deve estar em destaque agora (avançando a fila se o item anterior
// já expirou), quando essa exibição expira e por quanto tempo total ela dura — os dois
// últimos servem só pro painel desenhar a barra de contagem regressiva (22/09), a decisão
// de "trocar ou não" continua inteiramente daqui, no backend. duracaoNormal é usada quando
// a fila fica vazia depois de tirar o próximo; duracaoFila (mais curta) é usada enquanto
// ainda sobra backlog — drena mais rápido até esvaziar.
func (g *GerenciadorDeFilas) Atual(unidadeID string, duracaoNormal, duracaoFila time.Duration) (item *ItemExibicao, expiraEm time.Time, duracaoTotal time.Duration) {
	g.mu.Lock()
	defer g.mu.Unlock()
	f := g.fila(unidadeID)
	agora := time.Now()

	if f.atual != nil && agora.Before(f.expiraEm) {
		return f.atual, f.expiraEm, f.duracao
	}

	if len(f.pendentes) == 0 {
		f.atual = nil
		return nil, time.Time{}, 0
	}

	proximo := f.pendentes[0]
	f.pendentes = f.pendentes[1:]
	f.atual = &proximo

	duracao := duracaoNormal
	if len(f.pendentes) > 0 {
		duracao = duracaoFila
	}
	f.expiraEm = agora.Add(duracao)
	f.duracao = duracao
	return f.atual, f.expiraEm, f.duracao
}

func (g *GerenciadorDeFilas) fila(unidadeID string) *filaDaUnidade {
	f, ok := g.filas[unidadeID]
	if !ok {
		f = &filaDaUnidade{}
		g.filas[unidadeID] = f
	}
	return f
}
