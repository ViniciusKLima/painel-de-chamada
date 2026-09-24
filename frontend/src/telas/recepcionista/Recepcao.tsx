import { useCallback, useEffect, useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { LogOut, Search, UserPlus, UserCheck, Undo2, X } from "lucide-react";
import { api, ApiError, type Agendamento, type Grade, type Usuario } from "../../api";
import Cabecalho from "../../componentes/Cabecalho";
import { useNotificar } from "../../componentes/Notificacoes";
import { aplicarTemaPrefeitura } from "../../tema";

// Normaliza pra comparação de busca: minúsculas e sem pontuação de CPF/telefone, pra
// "12345678900" encontrar "123.456.789-00" e vice-versa.
function normalizarBusca(texto: string): string {
  return texto.toLowerCase().replace(/[.\-/\s]/g, "");
}

type Aba = "recepcao" | "sala_espera" | "ausente";

// Recepcionista sempre tem unidadeId/unidadeNome/secretariaId preenchidos (ver skill
// modelo-dados) — tipo estreitado evita `!`/cast espalhado pela tela inteira.
type UsuarioRecepcao = Usuario & { unidadeId: string; unidadeNome: string; secretariaId: string };

export default function Recepcao() {
  const [usuario, setUsuario] = useState<UsuarioRecepcao | null>(null);
  const [lista, setLista] = useState<Agendamento[]>([]);
  const notificar = useNotificar();
  const [aba, setAba] = useState<Aba>("recepcao");
  const [busca, setBusca] = useState("");
  const [modalEncaixeAberto, setModalEncaixeAberto] = useState(false);
  const [nomeEncaixe, setNomeEncaixe] = useState("");
  const [grades, setGrades] = useState<Grade[]>([]);
  const [gradeEncaixe, setGradeEncaixe] = useState("");
  const [agora, setAgora] = useState(() => Date.now());
  const navigate = useNavigate();

  const carregar = useCallback(async (unidadeId: string) => {
    setLista(await api.recepcao(unidadeId));
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const u = await api.sessao();
        // Correção 23/09 (auditoria de autenticação/autorização): esta tela nunca conferia
        // o papel antes — só buscava a sessão e seguia em frente, confiando que quem chegou
        // aqui só podia ser recepcionista. Gestor/atendente/admin logados que abrissem
        // `/recepcao` diretamente ficavam com a tela "funcionando" mas com a lista sempre
        // vazia e nenhum aviso (o backend barra a chamada por trás, silenciosamente — sem
        // furo de segurança, mas uma experiência confusa). Note: gestor/admin não têm um
        // `unidadeId` único (gerenciam a secretaria/plataforma inteira), então mesmo que
        // fossem liberados aqui, `api.recepcao(u.unidadeId)` quebraria com `unidadeId` nulo
        // — essa tela só faz sentido pra quem tem exatamente uma unidade, hoje só
        // recepcionista. Ver CLAUDE.md/pendências pra um seletor de unidade, se um dia
        // gestor precisar operar a recepção também.
        if (u.papel !== "recepcionista" || !u.unidadeId) {
          navigate("/login");
          return;
        }
        const usuarioRecepcao = u as UsuarioRecepcao;
        setUsuario(usuarioRecepcao);
        aplicarTemaPrefeitura(u.corDestaque, u.corClara);
        await carregar(usuarioRecepcao.unidadeId);
        // Grades da UNIDADE (22/09) — o encaixe (walk-in) precisa dizer pra qual
        // serviço/fila é, já que uma unidade pode ter mais de um.
        const listaGrades = await api.gradesDaUnidade(usuarioRecepcao.unidadeId);
        setGrades(listaGrades);
        setGradeEncaixe(listaGrades[0]?.id ?? "");
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) navigate("/login");
      }
    })();
  }, [carregar, navigate]);

  // Polling simples (fase gratuita, sem WebSocket)
  useEffect(() => {
    if (!usuario) return;
    const id = setInterval(() => {
      carregar(usuario.unidadeId);
      setAgora(Date.now());
    }, 2000);
    return () => clearInterval(id);
  }, [usuario, carregar]);

  async function confirmarChegada(id: string) {
    try {
      await api.confirmarChegada(id);
      notificar("Chegada confirmada.");
      if (usuario) await carregar(usuario.unidadeId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro.", "erro");
    }
  }

  // Puxar de volta pra recepção alguém que confirmou chegada por engano — só faz sentido
  // pra agendado (encaixe nunca passou por "aguardando chegada", ver skill regras-negocio-fila).
  async function desfazerChegada(id: string) {
    try {
      await api.desfazerChegada(id);
      notificar("Cidadão puxado de volta pra recepção.");
      if (usuario) await carregar(usuario.unidadeId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro.", "erro");
    }
  }

  async function criarEncaixe(evento: React.FormEvent) {
    evento.preventDefault();
    if (!usuario || !nomeEncaixe.trim() || !gradeEncaixe) return;
    try {
      await api.criarEncaixe(usuario.unidadeId, nomeEncaixe.trim(), gradeEncaixe);
      setNomeEncaixe("");
      setModalEncaixeAberto(false);
      notificar("Encaixe cadastrado — já está na sala de espera.");
      await carregar(usuario.unidadeId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro.", "erro");
    }
  }

  const listaDaAba = useMemo(() => {
    const status = aba === "recepcao" ? "aguardando_chegada" : aba === "sala_espera" ? "sala_espera" : "ausente";
    const filtrada = lista.filter((a) => a.status === status);
    // Aba "Faltaram" (21/09, pra recepção já saber que a pessoa está atrasada se chegar
    // depois): mais recente primeiro, é o caso mais provável de alguém aparecer na porta.
    if (aba === "ausente") {
      return [...filtrada].sort((a, b) => new Date(b.ausenteEm ?? 0).getTime() - new Date(a.ausenteEm ?? 0).getTime());
    }
    return filtrada;
  }, [lista, aba]);

  const listaFiltrada = useMemo(() => {
    const alvo = normalizarBusca(busca.trim());
    if (!alvo) return listaDaAba;
    return listaDaAba.filter((a) =>
      normalizarBusca(a.nomeCidadao).includes(alvo) ||
      normalizarBusca(a.protocolo).includes(alvo) ||
      (a.cpf && normalizarBusca(a.cpf).includes(alvo)),
    );
  }, [listaDaAba, busca]);

  if (!usuario) return <div className="tela-central">Carregando…</div>;

  const totalRecepcao = lista.filter((a) => a.status === "aguardando_chegada").length;
  const totalSalaEspera = lista.filter((a) => a.status === "sala_espera").length;
  const totalAusente = lista.filter((a) => a.status === "ausente").length;

  return (
    <>
      <Cabecalho
        nome={usuario.nome}
        papel="Recepção"
        unidadeNome={usuario.unidadeNome}
        prefeituraNome={usuario.prefeituraNome}
        acoes={
          <button onClick={() => api.sair().then(() => navigate("/login"))}>
            <LogOut size={16} /> Sair
          </button>
        }
      />

      <div className="pagina">
        <section className="cartao">
          <div className="barra-busca">
            <div className="campo-busca">
              <Search size={18} />
              <input
                placeholder="Pesquisar por nome, protocolo ou CPF"
                value={busca}
                onChange={(e) => setBusca(e.target.value)}
              />
            </div>
            <button onClick={() => setModalEncaixeAberto(true)}>
              <UserPlus size={16} /> Adicionar
            </button>
          </div>

          <div className="abas">
            <button className={`aba ${aba === "recepcao" ? "aba-ativa" : ""}`} onClick={() => setAba("recepcao")}>
              Recepção ({totalRecepcao})
            </button>
            <button className={`aba ${aba === "sala_espera" ? "aba-ativa" : ""}`} onClick={() => setAba("sala_espera")}>
              Sala de espera ({totalSalaEspera})
            </button>
            <button className={`aba ${aba === "ausente" ? "aba-ativa" : ""}`} onClick={() => setAba("ausente")}>
              Faltaram ({totalAusente})
            </button>
          </div>

          {/* Bug real corrigido (23/09): esta tabela era a única do sistema sem o wrapper
              `.tabela-cartao` — sem container próprio, em mobile ela estourava a largura da
              PÁGINA inteira (não só do cartão), causando o scroll lateral extra. */}
          <div className="tabela-cartao" style={{ marginTop: 12 }}>
          <table>
            <thead>
              <tr>
                <th>Horário</th>
                <th>Cidadão</th>
                <th>Protocolo</th>
                <th>Status</th>
                <th className="coluna-acao"></th>
              </tr>
            </thead>
            <tbody>
              {listaFiltrada.map((a) => {
                const jaChegouAHora = a.horarioPrevisto ? new Date(a.horarioPrevisto).getTime() <= agora : false;
                const atrasado = a.status === "aguardando_chegada" && jaChegouAHora;
                return (
                  <tr key={a.id} className={atrasado ? "linha-atrasada" : undefined}>
                    <td className="celula-horario numerico">
                      {a.horarioPrevisto ? new Date(a.horarioPrevisto).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" }) : "—"}
                    </td>
                    <td>
                      <strong>{a.nomeCidadao}</strong>
                      {a.servico && <div className="chamado-meta">{a.servico}</div>}
                    </td>
                    <td className="numerico">{a.protocolo}</td>
                    <td>
                      {a.status === "aguardando_chegada" ? (
                        <span className={`chip ${atrasado ? "atrasado" : "no_horario"}`}>{atrasado ? "atrasado" : "aguardando horário"}</span>
                      ) : a.status === "ausente" ? (
                        <span className="chip ausente">faltou</span>
                      ) : a.tipo === "encaixe" ? (
                        <span className="chip encaixe">encaixe</span>
                      ) : (
                        <span className="chip no_horario">na sala de espera</span>
                      )}
                    </td>
                    <td className="coluna-acao">
                      {a.status === "aguardando_chegada" ? (
                        <button className="botao-icone" onClick={() => confirmarChegada(a.id)} title="Confirmar chegada">
                          <UserCheck size={18} />
                        </button>
                      ) : a.status === "sala_espera" && a.tipo === "agendado" ? (
                        <button className="botao-icone" onClick={() => desfazerChegada(a.id)} title="Puxar de volta pra recepção">
                          <Undo2 size={18} />
                        </button>
                      ) : null}
                    </td>
                  </tr>
                );
              })}
              {listaFiltrada.length === 0 && (
                <tr>
                  <td colSpan={5}>
                    {busca
                      ? "Nenhum resultado para essa busca."
                      : aba === "recepcao"
                        ? "Ninguém aguardando chegada."
                        : aba === "sala_espera"
                          ? "Ninguém na sala de espera."
                          : "Ninguém faltou hoje."}
                  </td>
                </tr>
              )}
            </tbody>
          </table>
          </div>
        </section>

        {modalEncaixeAberto && (
          <div className="modal-fundo" onClick={() => setModalEncaixeAberto(false)}>
            <div className="modal-cartao" onClick={(e) => e.stopPropagation()}>
              <div className="modal-cartao-topo">
                <h2>Cadastrar encaixe</h2>
                <button className="botao-icone" onClick={() => setModalEncaixeAberto(false)} title="Fechar">
                  <X size={18} />
                </button>
              </div>
              <form onSubmit={criarEncaixe} className="linha">
                <input
                  placeholder="Nome do cidadão"
                  value={nomeEncaixe}
                  onChange={(e) => setNomeEncaixe(e.target.value)}
                  autoFocus
                  required
                />
                {grades.length > 1 && (
                  <select value={gradeEncaixe} onChange={(e) => setGradeEncaixe(e.target.value)} required>
                    {grades.map((g) => (
                      <option key={g.id} value={g.id}>{g.nome}</option>
                    ))}
                  </select>
                )}
                <button type="submit"><UserPlus size={16} /> Cadastrar</button>
              </form>
            </div>
          </div>
        )}
      </div>
    </>
  );
}
