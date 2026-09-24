import { useCallback, useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, ApiError, limparUsuarioId, type Agendamento, type Guiche, type Usuario } from "../api";

export default function Fila() {
  const [usuario, setUsuario] = useState<Usuario | null>(null);
  const [guiches, setGuiches] = useState<Guiche[]>([]);
  const [guicheSelecionado, setGuicheSelecionado] = useState("");
  const [aguardando, setAguardando] = useState<Agendamento[]>([]);
  const [chamados, setChamados] = useState<Agendamento[]>([]);
  const [mensagem, setMensagem] = useState<string | null>(null);
  const navigate = useNavigate();

  const carregarFila = useCallback(async (secretariaId: string) => {
    const [aguardandoLista, chamadosLista] = await Promise.all([
      api.agendamentos(secretariaId, "aguardando"),
      api.agendamentos(secretariaId, "chamado"),
    ]);
    setAguardando(aguardandoLista);
    setChamados(chamadosLista);
  }, []);

  useEffect(() => {
    (async () => {
      try {
        const me = await api.me();
        setUsuario(me);
        const listaGuiches = await api.guiches(me.secretariaId);
        setGuiches(listaGuiches);
        if (listaGuiches[0]) setGuicheSelecionado(listaGuiches[0].id);
        await carregarFila(me.secretariaId);
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) {
          limparUsuarioId();
          navigate("/login");
        }
      }
    })();
  }, [carregarFila, navigate]);

  // Polling simples (fase gratuita, sem WebSocket — ver CLAUDE.md)
  useEffect(() => {
    if (!usuario) return;
    const id = setInterval(() => carregarFila(usuario.secretariaId), 4000);
    return () => clearInterval(id);
  }, [usuario, carregarFila]);

  async function acao(fn: () => Promise<unknown>, sucesso: string) {
    setMensagem(null);
    try {
      await fn();
      setMensagem(sucesso);
      if (usuario) await carregarFila(usuario.secretariaId);
    } catch (err) {
      setMensagem(err instanceof Error ? err.message : "Erro inesperado.");
    }
  }

  function sair() {
    limparUsuarioId();
    navigate("/login");
  }

  if (!usuario) return <div className="tela-central">Carregando…</div>;

  return (
    <div className="pagina">
      <header className="topo">
        <div>
          <strong>{usuario.nome}</strong> — {usuario.secretaria.sigla} ({usuario.papel})
        </div>
        <nav>
          {(usuario.papel === "gestor" || usuario.papel === "admin") && (
            <Link to="/upload">Subir planilha</Link>
          )}
          <Link to={`/painel/${usuario.secretariaId}`} target="_blank">
            Ver painel de TV
          </Link>
          <button onClick={sair}>Sair</button>
        </nav>
      </header>

      {mensagem && <p className="mensagem">{mensagem}</p>}

      <section className="cartao">
        <h2>Chamar próximo</h2>
        <div className="linha">
          <select value={guicheSelecionado} onChange={(e) => setGuicheSelecionado(e.target.value)}>
            {guiches.map((g) => (
              <option key={g.id} value={g.id}>
                {g.nome}
              </option>
            ))}
          </select>
          <button
            disabled={!guicheSelecionado}
            onClick={() =>
              acao(() => api.chamarProximo(usuario.secretariaId, guicheSelecionado), "Próximo chamado.")
            }
          >
            Chamar próximo
          </button>
        </div>
      </section>

      <section className="cartao">
        <h2>Chamados no momento ({chamados.length})</h2>
        <table>
          <thead>
            <tr>
              <th>Protocolo</th>
              <th>Nome</th>
              <th>Guichê</th>
              <th>Ações</th>
            </tr>
          </thead>
          <tbody>
            {chamados.map((a) => (
              <tr key={a.id}>
                <td>{a.protocolo}</td>
                <td>{a.nomeCidadao}</td>
                <td>{a.guiche?.nome ?? "—"}</td>
                <td className="acoes">
                  <button onClick={() => acao(() => api.rechamar(a.id), "Rechamado.")}>Rechamar</button>
                  <button onClick={() => acao(() => api.marcarAtendido(a.id), "Marcado como atendido.")}>
                    Atendido
                  </button>
                  <button onClick={() => acao(() => api.marcarAusente(a.id), "Marcado como ausente.")}>
                    Ausente
                  </button>
                </td>
              </tr>
            ))}
            {chamados.length === 0 && (
              <tr>
                <td colSpan={4}>Ninguém chamado no momento.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>

      <section className="cartao">
        <h2>Aguardando ({aguardando.length})</h2>
        <table>
          <thead>
            <tr>
              <th>Horário</th>
              <th>Protocolo</th>
              <th>Nome</th>
              <th>Guichê</th>
            </tr>
          </thead>
          <tbody>
            {aguardando.map((a) => (
              <tr key={a.id}>
                <td>{new Date(a.horarioPrevisto).toLocaleTimeString("pt-BR", { hour: "2-digit", minute: "2-digit" })}</td>
                <td>{a.protocolo}</td>
                <td>{a.nomeCidadao}</td>
                <td>{a.guiche?.nome ?? "Qualquer guichê"}</td>
              </tr>
            ))}
            {aguardando.length === 0 && (
              <tr>
                <td colSpan={4}>Fila vazia.</td>
              </tr>
            )}
          </tbody>
        </table>
      </section>
    </div>
  );
}
