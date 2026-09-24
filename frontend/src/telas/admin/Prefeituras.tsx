import { useEffect, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Plus, Settings, X } from "lucide-react";
import { api, ApiError, type Prefeitura } from "../../api";
import { useNotificar } from "../../componentes/Notificacoes";

// Lista de prefeituras do admin (22/09) — cada linha abre pra "Configurações da Prefeitura"
// (nome + as duas cores de identidade + secretarias). Criar uma prefeitura nova é só o
// nome — secretaria/gestor vêm depois, dentro da tela de configuração dela.
export default function Prefeituras() {
  const [lista, setLista] = useState<Prefeitura[]>([]);
  const [modalAberto, setModalAberto] = useState(false);
  const [nomeNovo, setNomeNovo] = useState("");
  const [criando, setCriando] = useState(false);
  const notificar = useNotificar();
  const navigate = useNavigate();

  async function carregar() {
    setLista(await api.prefeituras());
  }

  useEffect(() => {
    carregar();
  }, []);

  async function criar(evento: React.FormEvent) {
    evento.preventDefault();
    if (!nomeNovo.trim()) return;
    setCriando(true);
    try {
      await api.criarPrefeitura(nomeNovo.trim());
      setNomeNovo("");
      setModalAberto(false);
      notificar("Prefeitura criada.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao criar prefeitura.", "erro");
    } finally {
      setCriando(false);
    }
  }

  return (
    <div className="pagina-gestor">
      <div className="secao-cabecalho">
        <h1>Prefeituras</h1>
        <button onClick={() => setModalAberto(true)}>
          <Plus size={16} /> Criar prefeitura
        </button>
      </div>

      <div className="tabela-cartao" style={{ marginTop: 16 }}>
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Slug</th>
              <th className="coluna-acao"></th>
            </tr>
          </thead>
          <tbody>
            {lista.map((p) => (
              <tr key={p.id}>
                <td>
                  <span className="bolinhas-cor-par" title={`Destaque ${p.corDestaque} · Clara ${p.corClara}`}>
                    <span className="bolinha-cor" style={{ background: p.corDestaque }} />
                    <span className="bolinha-cor" style={{ background: p.corClara }} />
                  </span>
                  <strong>{p.nome}</strong>
                </td>
                <td>{p.slug}</td>
                <td className="coluna-acao">
                  <button className="botao-icone" onClick={() => navigate(`/admin/prefeituras/${p.id}`)} title="Configurações da prefeitura">
                    <Settings size={18} />
                  </button>
                </td>
              </tr>
            ))}
            {lista.length === 0 && <tr><td colSpan={3}>Nenhuma prefeitura ainda.</td></tr>}
          </tbody>
        </table>
      </div>

      {modalAberto && (
        <div className="modal-fundo" onClick={() => setModalAberto(false)}>
          <div className="modal-cartao" onClick={(e) => e.stopPropagation()}>
            <div className="modal-cartao-topo">
              <h2>Criar prefeitura</h2>
              <button className="botao-icone" onClick={() => setModalAberto(false)}><X size={18} /></button>
            </div>
            <form onSubmit={criar} className="form-vertical">
              <label>
                Nome
                <input value={nomeNovo} onChange={(e) => setNomeNovo(e.target.value)} placeholder="Ex.: Prefeitura de Jaboatão dos Guararapes" required autoFocus />
              </label>
              <button type="submit" disabled={criando}>{criando ? "Criando…" : "Criar"}</button>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
