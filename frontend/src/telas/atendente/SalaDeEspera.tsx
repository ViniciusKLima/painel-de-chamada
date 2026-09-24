import { useCallback, useEffect, useRef, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CheckCircle2, Clock, LogOut, Megaphone, Repeat, Search, UserX, Volume2 } from "lucide-react";
import { api, ApiError, type Agendamento, type ChamadoCompartilhado, type Guiche, type GradeResumo, type Usuario } from "../../api";
import Cabecalho from "../../componentes/Cabecalho";
import { useNotificar } from "../../componentes/Notificacoes";
import { aplicarTemaPrefeitura } from "../../tema";

// Chave por GRADE (23/09) — cada grade tem sua própria ocupação de guichê, já que um
// atendente pode estar alocado em várias ao mesmo tempo (inclusive de unidades físicas
// diferentes). Antes era uma chave só, porque só existia uma grade em uso por vez.
function chaveGuicheLocal(gradeId: string): string {
  return `painel-chamada:guiche-selecionado:${gradeId}`;
}

function lerLocal(chave: string): string | null {
  try {
    return localStorage.getItem(chave);
  } catch {
    return null;
  }
}

function gravarLocal(chave: string, valor: string) {
  try {
    localStorage.setItem(chave, valor);
  } catch {
    // ambiente sem storage disponível — segue só em memória
  }
}

// Reafirma a ocupação dos guichês a cada 5s (heartbeat) — bem menor que a janela de 15s que
// o backend usa pra considerar um guichê "abandonado" (aba fechada sem logout), pra tolerar
// perda ocasional de uma requisição sem derrubar a ocupação por engano (22/09, ver skill
// regras-negocio-fila). Roda por GRADE agora (23/09) — um atendente pode ter um guichê
// ocupado em cada uma de várias grades ao mesmo tempo.
const INTERVALO_HEARTBEAT_GUICHE_MS = 5000;
const INTERVALO_POLL_DADOS_MS = 2000;
const INTERVALO_POLL_GUICHES_MS = 2000;
// Revalida a lista de grades alocadas periodicamente (23/09, auditoria de multi-grade,
// requisito explícito: "não quero situações em que a alteração aparece apenas depois de
// logout/login ou apenas depois de atualizar manualmente a página") — se um gestor/admin
// realoca esse atendente (adiciona ou remove uma grade) enquanto ele está com a tela aberta,
// a mudança chega sozinha, sem precisar de F5. Não precisa ser tão frequente quanto o poll
// operacional (2s) — realocação é um evento raro, não algo que muda a cada segundo.
const INTERVALO_POLL_SESSAO_MS = 8000;

function useContagem(alvoISO: string): number {
  const calcular = () => new Date(alvoISO).getTime() - Date.now();
  const [restanteMs, setRestanteMs] = useState(calcular);
  useEffect(() => {
    const id = setInterval(() => setRestanteMs(calcular()), 1000);
    return () => clearInterval(id);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [alvoISO]);
  return restanteMs;
}

function formatarMMSS(ms: number): string {
  const totalSegundos = Math.max(0, Math.floor(ms / 1000));
  const min = Math.floor(totalSegundos / 60);
  const seg = totalSegundos % 60;
  return `${min}:${seg.toString().padStart(2, "0")}`;
}

// Contagem regressiva até o prazo de ausência automática (primeira chamada + SLA de
// atendimento da grade) — pedido do dono do produto (20/09): visível pra todo mundo no
// bloco compartilhado, não só pro dono (ver skill regras-negocio-fila).
function ContagemAusencia({ prazoISO }: { prazoISO: string }) {
  const restanteMs = useContagem(prazoISO);
  if (restanteMs <= 0) return <span className="contagem-ausencia expirada">prazo esgotado</span>;
  return (
    <span className={`contagem-ausencia ${restanteMs < 60_000 ? "alerta" : ""}`}>
      {formatarMMSS(restanteMs)}
    </span>
  );
}

// Uma linha da tabela "Chamando agora" — compartilhada entre todos os atendentes DA MESMA
// grade (decisão 20/09): todo mundo vê quem está sendo chamado e por qual guichê; só o dono
// ganha os botões de ação (rechamar/marcar presente, em ícone). Um chamado "órfão" (o dono já
// rechamou e seguiu em frente pra outra pessoa) pode ser assumido por qualquer atendente
// ocioso DAQUELA grade — por isso "assumir" precisa de um guichê JÁ selecionado na mesma
// grade do item (23/09, ver `onAssumir` em SalaDeEspera).
function LinhaChamado({
  item,
  nomeGrade,
  onRechamar,
  onMarcarPresente,
  onMarcarAusente,
  onAssumir,
}: {
  item: ChamadoCompartilhado;
  nomeGrade: string;
  onRechamar: () => void;
  onMarcarPresente: () => void;
  onMarcarAusente: () => void;
  onAssumir: () => void;
}) {
  const restanteCooldown = useContagem(item.podeRechamarEm);
  const podeRechamar = restanteCooldown <= 0;

  return (
    <tr>
      <td className="celula-guiche">{item.guicheNome}</td>
      <td>
        <strong>{item.nomeCidadao}</strong>
        <div className="chamado-meta">
          {item.protocolo} · {nomeGrade} · chamado por {item.chamadoPorNome}
          {item.ehOrfao && <span className="chip atrasado" style={{ marginLeft: 8 }}>sem resposta</span>}
        </div>
      </td>
      <td className="celula-prazo"><ContagemAusencia prazoISO={item.prazoAusenciaEm} /></td>
      <td className="coluna-acao">
        {item.souDono ? (
          <div className="acoes-icones">
            {podeRechamar ? (
              <button className="botao-icone" onClick={onRechamar} title="Rechamar">
                <Repeat size={18} />
              </button>
            ) : (
              <button className="botao-cooldown" disabled title="Aguarde pra rechamar">
                <Clock size={14} /> {formatarMMSS(restanteCooldown)}
              </button>
            )}
            <button className="botao-icone botao-icone-sucesso" onClick={onMarcarPresente} title="Marcar presente">
              <CheckCircle2 size={18} />
            </button>
            {/* Ausência manual (23/09, pedido do dono do produto): caso alguém avise que o
                cidadão já foi embora, ou ele mesmo avise, o atendente não precisa esperar o
                sweeper automático rodar até o fim do SLA — marca ausente na hora. */}
            <button className="botao-icone botao-icone-perigo" onClick={onMarcarAusente} title="Marcar ausência">
              <UserX size={18} />
            </button>
          </div>
        ) : item.ehOrfao ? (
          <button className="botao-icone" onClick={onAssumir} title="Assumir chamado">
            <Volume2 size={18} />
          </button>
        ) : null}
      </td>
    </tr>
  );
}

type AbaSalaEspera = "espera" | "ausentes";

// Normaliza pra comparação de busca — mesma lógica da recepção (case-insensitive, ignora
// pontuação, útil se um dia o protocolo tiver separadores).
function normalizarBusca(texto: string): string {
  return texto.toLowerCase().replace(/[.\-/\s]/g, "");
}

// Atendente sempre tem unidadeNome preenchido na sessão (ver skill modelo-dados) — mas
// desde a auditoria de multi-grade (23/09) isso é só a unidade "principal" do cadastro do
// usuário; as grades alocadas (u.grades) é que carregam a unidade DE CADA UMA, já que podem
// ser de locais físicos diferentes (exemplo real confirmado pelo dono do produto: atendente
// com grades na Unidade A e na Unidade B). O agrupamento visual usa sempre `grade.unidadeNome`,
// nunca essa unidade única do usuário.
type UsuarioAtendente = Usuario & { unidadeNome: string };

// Um "grupo de unidade" — todas as grades do atendente que pertencem ao mesmo local físico,
// agrupadas juntas na tela (23/09, pedido explícito do dono do produto: "grades do mesmo
// local devem aparecer agrupadas visualmente, não como filas independentes").
interface GrupoUnidade {
  unidadeId: string;
  unidadeNome: string;
  grades: GradeResumo[];
}

function agruparPorUnidade(grades: GradeResumo[]): GrupoUnidade[] {
  const grupos = new Map<string, GrupoUnidade>();
  for (const g of grades) {
    let grupo = grupos.get(g.unidadeId);
    if (!grupo) {
      grupo = { unidadeId: g.unidadeId, unidadeNome: g.unidadeNome, grades: [] };
      grupos.set(g.unidadeId, grupo);
    }
    grupo.grades.push(g);
  }
  return Array.from(grupos.values()).sort((a, b) => a.unidadeNome.localeCompare(b.unidadeNome));
}

export default function SalaDeEspera() {
  const [usuario, setUsuario] = useState<UsuarioAtendente | null>(null);
  // Todas as grades do atendente, consolidadas (23/09) — não há mais "a grade em uso": ele
  // vê e trabalha em TODAS ao mesmo tempo, sem selecionar nenhuma antes.
  const [grades, setGrades] = useState<GradeResumo[]>([]);
  const [chamados, setChamados] = useState<ChamadoCompartilhado[]>([]);
  const [espera, setEspera] = useState<Agendamento[]>([]);
  const [atendidosHoje, setAtendidosHoje] = useState<Agendamento[]>([]);
  const [ausentesHoje, setAusentesHoje] = useState<Agendamento[]>([]);
  const [abaSalaEspera, setAbaSalaEspera] = useState<AbaSalaEspera>("espera");
  const [buscaAtendimentos, setBuscaAtendimentos] = useState("");
  // Guichês agrupados por grade — um atendente pode ter um guichê ocupado em CADA grade ao
  // mesmo tempo, já que cada grade é uma fila física própria (23/09).
  const [guichesPorGrade, setGuichesPorGrade] = useState<Record<string, Guiche[]>>({});
  const [guicheSelecionadoPorGrade, setGuicheSelecionadoPorGrade] = useState<Record<string, string>>({});
  // Qual grade tem o modal de seleção de guichê aberto agora (null = nenhum) — não é mais um
  // gate obrigatório de tela cheia: cada grade resolve o próprio guichê sob demanda, sem
  // bloquear as outras grades já resolvidas (23/09, "não quero uma interface em que o
  // atendente precise selecionar uma grade" — e, por extensão, sem travar a tela toda
  // esperando uma escolha de guichê também).
  const [modalGuicheGradeId, setModalGuicheGradeId] = useState<string | null>(null);
  const guicheSelecionadoRef = useRef(guicheSelecionadoPorGrade);
  guicheSelecionadoRef.current = guicheSelecionadoPorGrade;
  const notificar = useNotificar();
  const navigate = useNavigate();

  const carregarDados = useCallback(async () => {
    const [c, e, h, au] = await Promise.all([
      api.minhasGradesChamados(),
      api.minhasGradesSalaEspera(),
      api.minhasGradesAtendidosHoje(),
      api.minhasGradesAusentesHoje(),
    ]);
    setChamados(c);
    setEspera(e);
    setAtendidosHoje(h);
    setAusentesHoje(au);
  }, []);

  const carregarGuiches = useCallback(async () => {
    const lista = await api.minhasGradesGuiches();
    const porGrade: Record<string, Guiche[]> = {};
    for (const g of lista) {
      (porGrade[g.gradeId] ??= []).push(g);
    }
    setGuichesPorGrade(porGrade);
    return porGrade;
  }, []);

  // Tenta reocupar automaticamente o guichê lembrado de cada grade em `gradesAlvo` (23/09) —
  // sem pedir confirmação de novo, já que a regra de negócio não quer nenhuma seleção
  // obrigatória bloqueando a tela. Se o guichê lembrado já estiver ocupado por outra pessoa
  // (ex: alguém assumiu enquanto o atendente estava fora, ou a grade acabou de ser alocada
  // pra ele agora mesmo), a ocupação falha silenciosamente e a grade fica sem guichê —
  // aparece como "Selecionar guichê" na seção dela, sem travar as outras grades já
  // resolvidas. Reaproveitada tanto no carregamento inicial quanto quando uma grade nova
  // aparece em runtime (realocação feita por um gestor/admin, ver poll de sessão abaixo).
  const resolverGuichesLembrados = useCallback(async (gradesAlvo: GradeResumo[]) => {
    if (gradesAlvo.length === 0) return;
    const porGrade = await carregarGuiches();
    const selecoes: Record<string, string> = {};
    await Promise.all(
      gradesAlvo.map(async (g) => {
        const lembrado = lerLocal(chaveGuicheLocal(g.id));
        if (!lembrado) return;
        const existe = (porGrade[g.id] ?? []).some((guiche) => guiche.id === lembrado);
        if (!existe) return;
        try {
          await api.ocuparGuiche(g.id, lembrado);
          selecoes[g.id] = lembrado;
        } catch {
          // guichê perdido pra outro atendente — segue sem selecionar, resolve pelo modal
        }
      })
    );
    if (Object.keys(selecoes).length > 0) {
      setGuicheSelecionadoPorGrade((atual) => ({ ...atual, ...selecoes }));
    }
  }, [carregarGuiches]);

  const gradesConhecidasRef = useRef<Set<string>>(new Set());

  useEffect(() => {
    (async () => {
      try {
        const u = await api.sessao();
        // Correção 23/09 (auditoria de autenticação/autorização): antes disso, um papel sem
        // grade fazia esse `return` ficar preso pra sempre em "Carregando…" — nenhum aviso,
        // nenhum redirecionamento, só uma tela travada.
        if (u.papel !== "atendente" || u.grades.length === 0) {
          navigate("/login");
          return;
        }
        setUsuario(u as UsuarioAtendente);
        setGrades(u.grades);
        gradesConhecidasRef.current = new Set(u.grades.map((g) => g.id));
        aplicarTemaPrefeitura(u.corDestaque, u.corClara);
        await resolverGuichesLembrados(u.grades);
        await carregarDados();
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) navigate("/login");
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [navigate]);

  // Poll dos dados operacionais (chamados/sala de espera/atendidos/ausentes) — sempre ativo,
  // não depende mais de nenhuma seleção prévia (23/09).
  useEffect(() => {
    if (grades.length === 0) return;
    const id = setInterval(() => {
      carregarDados().catch(() => {});
    }, INTERVALO_POLL_DADOS_MS);
    return () => clearInterval(id);
  }, [grades, carregarDados]);

  // Revalida a lista de grades alocadas (23/09, ver INTERVALO_POLL_SESSAO_MS acima) — se um
  // gestor/admin adicionar ou remover uma grade deste atendente enquanto a tela está aberta,
  // isso chega sozinho, sem precisar de logout/login nem F5 (requisito explícito da
  // auditoria). Grade nova: tenta resolver o guichê lembrado dela na hora. Grade removida: o
  // efeito de limpeza abaixo (que reage a `grades`) libera o guichê que estava ocupado nela.
  useEffect(() => {
    if (grades.length === 0) return;
    const id = setInterval(async () => {
      try {
        const u = await api.sessao();
        if (u.papel !== "atendente") return; // troca de identidade — o interceptor global cuida disso
        const idsNovos = new Set(u.grades.map((g) => g.id));
        const mudou = idsNovos.size !== gradesConhecidasRef.current.size || [...idsNovos].some((gid) => !gradesConhecidasRef.current.has(gid));
        if (!mudou) return;
        const gradesAdicionadas = u.grades.filter((g) => !gradesConhecidasRef.current.has(g.id));
        gradesConhecidasRef.current = idsNovos;
        setGrades(u.grades);
        if (gradesAdicionadas.length > 0) await resolverGuichesLembrados(gradesAdicionadas);
      } catch {
        // falha de rede numa revalidação periódica não é motivo pra travar a tela
      }
    }, INTERVALO_POLL_SESSAO_MS);
    return () => clearInterval(id);
  }, [grades.length, resolverGuichesLembrados]);

  // Limpeza (23/09): se uma grade sai da lista (realocação feita por um gestor/admin — ver
  // poll de sessão acima), qualquer guichê que o atendente tinha ocupado NELA é liberado no
  // backend e removido do estado local — sem isso, o heartbeat continuaria tentando reafirmar
  // a ocupação de um guichê que ele não tem mais acesso (a rota exige a grade dele, ver
  // exigirGrade), e a UI achando (erradamente) que ele ainda está posicionado ali.
  useEffect(() => {
    const idsValidos = new Set(grades.map((g) => g.id));
    setGuicheSelecionadoPorGrade((atual) => {
      const removidas = Object.entries(atual).filter(([gradeId]) => !idsValidos.has(gradeId));
      if (removidas.length === 0) return atual;
      removidas.forEach(([gradeId, guicheId]) => {
        api.liberarGuiche(gradeId, guicheId).catch(() => {});
      });
      const proximo = { ...atual };
      removidas.forEach(([gradeId]) => delete proximo[gradeId]);
      return proximo;
    });
  }, [grades]);

  // Enquanto o modal de guichê de alguma grade está aberto, mantém a lista atualizada (2s) —
  // é assim que um atendente vê em tempo real um guichê ficando ocupado/liberado por outro
  // (22/09, ver skill regras-negocio-fila).
  useEffect(() => {
    if (!modalGuicheGradeId) return;
    const id = setInterval(() => {
      carregarGuiches().catch(() => {});
    }, INTERVALO_POLL_GUICHES_MS);
    return () => clearInterval(id);
  }, [modalGuicheGradeId, carregarGuiches]);

  // Heartbeat (22/09, agora por grade — 23/09): reafirma a ocupação de TODOS os guichês
  // selecionados periodicamente — sem isso, o backend liberaria cada guichê pra outro
  // atendente depois da janela de 15s mesmo com o atendente ainda trabalhando ali. Se o
  // heartbeat falhar (guichê perdido, ex: outro atendente assumiu depois de uma queda de
  // conexão longa), a seleção daquela grade é limpa — a seção correspondente volta a mostrar
  // "Selecionar guichê" em vez de continuar achando (erradamente) que está ocupando algo.
  useEffect(() => {
    const id = setInterval(() => {
      const selecoes = guicheSelecionadoRef.current;
      Object.entries(selecoes).forEach(([gradeId, guicheId]) => {
        api.ocuparGuiche(gradeId, guicheId).catch(() => {
          setGuicheSelecionadoPorGrade((atual) => {
            if (atual[gradeId] !== guicheId) return atual;
            const proximo = { ...atual };
            delete proximo[gradeId];
            return proximo;
          });
        });
      });
    }, INTERVALO_HEARTBEAT_GUICHE_MS);
    return () => clearInterval(id);
  }, []);

  async function escolherGuiche(gradeId: string, guicheId: string) {
    try {
      await api.ocuparGuiche(gradeId, guicheId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Não foi possível ocupar este guichê.", "erro");
      carregarGuiches().catch(() => {});
      return;
    }
    setGuicheSelecionadoPorGrade((atual) => ({ ...atual, [gradeId]: guicheId }));
    gravarLocal(chaveGuicheLocal(gradeId), guicheId);
    setModalGuicheGradeId(null);
  }

  function abrirModalGuiche(gradeId: string) {
    carregarGuiches().catch(() => {});
    setModalGuicheGradeId(gradeId);
  }

  async function sair() {
    const liberacoes = Object.entries(guicheSelecionadoPorGrade).map(([gradeId, guicheId]) =>
      api.liberarGuiche(gradeId, guicheId).catch(() => {})
    );
    await Promise.all(liberacoes);
    await api.sair();
    navigate("/login");
  }

  async function acao(fn: () => Promise<unknown>, sucesso: string) {
    try {
      await fn();
      notificar(sucesso);
      await carregarDados();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro.", "erro");
    }
  }

  function chamarProximo(gradeId: string) {
    const guicheId = guicheSelecionadoPorGrade[gradeId];
    if (!guicheId) {
      abrirModalGuiche(gradeId);
      return;
    }
    acao(() => api.chamarProximo(gradeId, guicheId), "Próximo chamado.");
  }

  function assumir(item: ChamadoCompartilhado) {
    const guicheId = guicheSelecionadoPorGrade[item.gradeId];
    if (!guicheId) {
      notificar("Selecione um guichê nessa grade antes de assumir.", "erro");
      abrirModalGuiche(item.gradeId);
      return;
    }
    acao(() => api.assumirChamada(item.id, guicheId), "Chamado assumido.");
  }

  if (!usuario) return <div className="tela-central">Carregando…</div>;
  if (grades.length === 0) {
    return (
      <div className="tela-central">
        <div className="cartao">
          <h1>Sem grade alocada</h1>
          <p>Você ainda não está alocado em nenhuma grade (fila de serviço) — peça pro gestor te alocar na tela "Usuários internos".</p>
        </div>
      </div>
    );
  }

  const gradesPorId = new Map(grades.map((g) => [g.id, g]));
  const nomeDaGrade = (gradeId: string) => gradesPorId.get(gradeId)?.nome ?? "—";
  const grupos = agruparPorUnidade(grades);

  // Regra 20/09, escopo por grade confirmado na auditoria 23/09 (ver
  // Repo.TemChamadaPendenteNaoRechamada/ExistemChamadosOrfaos, ambos filtrados por
  // grade_id): não dá pra puxar gente nova da sala de espera de UMA grade se (a) o próprio
  // atendente tem uma chamada pendente NAQUELA grade que ainda não rechamou, ou (b) existe
  // qualquer chamado órfão NAQUELA grade — isso nunca bloqueia as outras grades do mesmo
  // atendente, mesmo as da mesma unidade física.
  function motivoBloqueio(gradeId: string): string | null {
    const chamadosDaGrade = chamados.filter((c) => c.gradeId === gradeId);
    const meuPendente = chamadosDaGrade.find((c) => c.souDono && c.totalChamadasDoDono === 1);
    const existeOrfao = chamadosDaGrade.some((c) => c.ehOrfao);
    if (meuPendente) return "Você tem uma chamada pendente. Rechame antes de chamar outra pessoa.";
    if (existeOrfao) return "Existem chamados sem resposta. Assuma um deles antes de chamar alguém novo.";
    return null;
  }

  const atendidosFiltrados = (() => {
    const alvo = normalizarBusca(buscaAtendimentos.trim());
    if (!alvo) return atendidosHoje;
    return atendidosHoje.filter((a) => normalizarBusca(a.nomeCidadao).includes(alvo) || normalizarBusca(a.protocolo).includes(alvo));
  })();

  const gradeModal = modalGuicheGradeId ? gradesPorId.get(modalGuicheGradeId) : null;
  const guichesDoModal = modalGuicheGradeId ? guichesPorGrade[modalGuicheGradeId] ?? [] : [];

  return (
    <>
      <Cabecalho
        nome={usuario.nome}
        papel="Atendente"
        unidadeNome={usuario.unidadeNome}
        prefeituraNome={usuario.prefeituraNome}
        acoes={
          <button onClick={sair}>
            <LogOut size={16} /> Sair
          </button>
        }
      />

      <div className="pagina pagina-larga">
        {/* Modal de guichê agora é POR GRADE, aberto sob demanda (23/09) — nunca bloqueia a
            tela inteira: as outras grades já resolvidas continuam operando normalmente atrás
            dele. Mesmo padrão visual de sempre (quadrados clicáveis, ocupado por outro fica
            desabilitado com o nome de quem está lá). */}
        {gradeModal && (
          <div className="modal-fundo" onClick={() => setModalGuicheGradeId(null)}>
            <div className="modal-selecao-guiche" onClick={(e) => e.stopPropagation()}>
              <h2>Qual guichê você vai atender?</h2>
              <p className="aviso">{gradeModal.unidadeNome} — {gradeModal.nome}</p>
              {guichesDoModal.length === 0 ? (
                <p>Nenhum guichê cadastrado — peça pro gestor cadastrar um.</p>
              ) : (
                <div className="grade-quadros-guiche">
                  {guichesDoModal.map((g) => {
                    const ocupadoPorOutro = g.ocupadoPorUsuarioId !== null && g.ocupadoPorUsuarioId !== usuario.id;
                    return (
                      <button
                        key={g.id}
                        className="quadro-guiche"
                        onClick={() => escolherGuiche(gradeModal.id, g.id)}
                        disabled={ocupadoPorOutro}
                        title={ocupadoPorOutro ? `Ocupado por ${g.ocupadoPorNome}` : undefined}
                      >
                        {g.nome}
                        {ocupadoPorOutro && <span className="quadro-guiche-ocupado">Ocupado por {g.ocupadoPorNome}</span>}
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        )}

        <div className="grade-atendente">
          {/* Blocos sem cartão em volta (21/09, pedido do dono do produto) — título solto
              direto sobre o fundo da página, a tabela é o próprio "bloco" abaixo dele.
              "Chamando agora" e "Meus atendimentos hoje" continuam como tabelas únicas,
              consolidadas entre TODAS as grades do atendente, com uma coluna "Grade" a mais
              pra identificar de qual fila é cada linha (23/09). */}
          <section className="bloco-chamando">
            <h2>Pendentes ({chamados.length})</h2>
            <div className="tabela-cartao">
              <table>
                <thead>
                  <tr>
                    <th>Guichê</th>
                    <th>Cidadão</th>
                    <th>Prazo</th>
                    <th className="coluna-acao"></th>
                  </tr>
                </thead>
                <tbody>
                  {chamados.map((item) => (
                    <LinhaChamado
                      key={item.id}
                      item={item}
                      nomeGrade={nomeDaGrade(item.gradeId)}
                      onRechamar={() => acao(() => api.rechamar(item.id), "Rechamado.")}
                      onMarcarPresente={() => acao(() => api.marcarAtendido(item.id), "Marcado como presente.")}
                      onMarcarAusente={() => acao(() => api.marcarAusencia(item.id), "Marcado como ausente.")}
                      onAssumir={() => assumir(item)}
                    />
                  ))}
                  {chamados.length === 0 && <tr><td colSpan={4}>Ninguém sendo chamado no momento.</td></tr>}
                </tbody>
              </table>
            </div>
          </section>

          <section className="bloco-espera">
            <div className="secao-cabecalho">
              <h2>Sala de espera ({espera.length})</h2>
            </div>
            <div className="abas">
              <button className={`aba ${abaSalaEspera === "espera" ? "aba-ativa" : ""}`} onClick={() => setAbaSalaEspera("espera")}>
                Aguardando ({espera.length})
              </button>
              <button className={`aba ${abaSalaEspera === "ausentes" ? "aba-ativa" : ""}`} onClick={() => setAbaSalaEspera("ausentes")}>
                Ausentes ({ausentesHoje.length})
              </button>
            </div>

            {abaSalaEspera === "espera" ? (
              // Agrupado por UNIDADE -> GRADE (23/09, pedido explícito do dono do produto:
              // grades do mesmo local físico ficam visualmente juntas). Cada grade tem seu
              // próprio "Chamar próximo" e seu próprio guichê, já que a fila e o bloqueio de
              // chamada são escopados por grade, não pela unidade nem pelo atendente como um
              // todo (ver motivoBloqueio acima).
              <div className="grupos-unidade">
                {grupos.map((grupo) => (
                  <div key={grupo.unidadeId} className="grupo-unidade">
                    {grupos.length > 1 && <h3 className="grupo-unidade-titulo">{grupo.unidadeNome}</h3>}
                    {grupo.grades.map((g) => {
                      const itensGrade = espera.filter((a) => a.gradeId === g.id);
                      const guicheAtual = guicheSelecionadoPorGrade[g.id];
                      const nomeGuicheAtual = guichesPorGrade[g.id]?.find((x) => x.id === guicheAtual)?.nome;
                      const bloqueio = motivoBloqueio(g.id);
                      return (
                        <div key={g.id} className="grupo-grade">
                          <div className="grupo-grade-cabecalho">
                            <div>
                              <strong>{g.nome}</strong>
                              <span className="grupo-grade-contagem"> ({itensGrade.length})</span>
                            </div>
                            <div className="grupo-grade-acoes">
                              <button className="pilula-guiche" onClick={() => abrirModalGuiche(g.id)}>
                                {nomeGuicheAtual ?? "Selecionar guichê"}
                              </button>
                              <button
                                disabled={bloqueio !== null}
                                onClick={() => chamarProximo(g.id)}
                                title={bloqueio ?? (guicheAtual ? "Chamar próximo da fila" : "Escolher guichê e chamar")}
                              >
                                <Megaphone size={16} /> Chamar próximo
                              </button>
                            </div>
                          </div>
                          {bloqueio && <p className="aviso">{bloqueio}</p>}
                          <div className="tabela-cartao">
                            <table>
                              <thead>
                                <tr>
                                  <th>Nome</th>
                                  <th>Protocolo</th>
                                  <th>Prioridade</th>
                                </tr>
                              </thead>
                              <tbody>
                                {itensGrade.map((a) => (
                                  <tr key={a.id}>
                                    <td>{a.nomeCidadao}</td>
                                    <td className="numerico">{a.protocolo}</td>
                                    <td>
                                      {a.tipo === "encaixe" ? (
                                        <span className="chip encaixe">encaixe</span>
                                      ) : (
                                        <span className={`chip ${a.prioridade ?? "atrasado"}`}>{a.prioridade === "no_horario" ? "no horário" : "atrasado"}</span>
                                      )}
                                    </td>
                                  </tr>
                                ))}
                                {itensGrade.length === 0 && <tr><td colSpan={3}>Fila vazia.</td></tr>}
                              </tbody>
                            </table>
                          </div>
                        </div>
                      );
                    })}
                  </div>
                ))}
              </div>
            ) : (
              <div className="tabela-cartao">
                <table>
                  <thead>
                    <tr>
                      <th>Nome</th>
                      <th>Protocolo</th>
                      <th>Grade</th>
                      <th>Status</th>
                    </tr>
                  </thead>
                  <tbody>
                    {ausentesHoje.map((a) => (
                      <tr key={a.id}>
                        <td>{a.nomeCidadao}</td>
                        <td className="numerico">{a.protocolo}</td>
                        <td>{nomeDaGrade(a.gradeId)}</td>
                        <td><span className="chip ausente">faltou</span></td>
                      </tr>
                    ))}
                    {ausentesHoje.length === 0 && <tr><td colSpan={4}>Ninguém faltou depois de chamado, ainda bem.</td></tr>}
                  </tbody>
                </table>
              </div>
            )}
          </section>

          <section className="bloco-atendidos">
            <h2>Meus atendimentos hoje ({atendidosHoje.length})</h2>
            <div className="barra-busca">
              <div className="campo-busca">
                <Search size={16} />
                <input
                  placeholder="Pesquisar por nome ou protocolo"
                  value={buscaAtendimentos}
                  onChange={(e) => setBuscaAtendimentos(e.target.value)}
                />
              </div>
            </div>
            <div className="tabela-cartao">
              <table>
                <thead><tr><th>Nome</th><th>Protocolo</th><th>Grade</th></tr></thead>
                <tbody>
                  {atendidosFiltrados.map((a) => (
                    <tr key={a.id}>
                      <td>{a.nomeCidadao}</td>
                      <td className="numerico">{a.protocolo}</td>
                      <td>{nomeDaGrade(a.gradeId)}</td>
                    </tr>
                  ))}
                  {atendidosFiltrados.length === 0 && (
                    <tr>
                      <td colSpan={3}>{buscaAtendimentos ? "Nenhum resultado para essa busca." : "Nenhum atendimento concluído ainda."}</td>
                    </tr>
                  )}
                </tbody>
              </table>
            </div>
          </section>
        </div>
      </div>
    </>
  );
}
