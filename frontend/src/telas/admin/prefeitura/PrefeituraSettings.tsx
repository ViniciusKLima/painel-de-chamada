import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { Plus, Save } from "lucide-react";
import { api, ApiError, type DetalhePrefeitura } from "../../../api";
import { useNotificar } from "../../../componentes/Notificacoes";

// Aba "Settings" da configuração da prefeitura (22/09, reorganizada em abas 23/09) — nome,
// as duas cores de identidade (destaque + clara, ver skill identidade-visual) e a lista de
// secretarias dela, com criação rápida. Cabeçalho/voltar/abas vêm de PrefeituraLayout, que
// envolve esta tela — aqui só o conteúdo da aba em si.
export default function PrefeituraSettings() {
  const { prefeituraId } = useParams<{ prefeituraId: string }>();
  const notificar = useNotificar();

  const [detalhe, setDetalhe] = useState<DetalhePrefeitura | null>(null);
  const [nome, setNome] = useState("");
  const [corDestaque, setCorDestaque] = useState("#146c8c");
  const [corClara, setCorClara] = useState("#e4f1f5");
  const [salvando, setSalvando] = useState(false);

  const [nomeSecretaria, setNomeSecretaria] = useState("");
  const [siglaSecretaria, setSiglaSecretaria] = useState("");
  const [criandoSecretaria, setCriandoSecretaria] = useState(false);

  const carregar = useCallback(async (id: string) => {
    const d = await api.prefeitura(id);
    setDetalhe(d);
    setNome(d.prefeitura.nome);
    setCorDestaque(d.prefeitura.corDestaque);
    setCorClara(d.prefeitura.corClara);
  }, []);

  useEffect(() => {
    if (prefeituraId) carregar(prefeituraId);
  }, [prefeituraId, carregar]);

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    if (!prefeituraId) return;
    setSalvando(true);
    try {
      await api.atualizarPrefeitura(prefeituraId, { nome, corDestaque, corClara });
      notificar("Configuração salva.");
      await carregar(prefeituraId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao salvar.", "erro");
    } finally {
      setSalvando(false);
    }
  }

  async function criarSecretaria(evento: React.FormEvent) {
    evento.preventDefault();
    if (!prefeituraId || !nomeSecretaria.trim() || !siglaSecretaria.trim()) return;
    setCriandoSecretaria(true);
    try {
      await api.criarSecretaria(prefeituraId, nomeSecretaria.trim(), siglaSecretaria.trim());
      setNomeSecretaria("");
      setSiglaSecretaria("");
      notificar("Secretaria criada.");
      await carregar(prefeituraId);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao criar secretaria.", "erro");
    } finally {
      setCriandoSecretaria(false);
    }
  }

  if (!detalhe) return <p>Carregando…</p>;

  return (
    <>
      <section className="cartao">
        <h3>Identidade</h3>
        <p className="dica">Nome e as duas cores usadas em toda a interface desta prefeitura — a de destaque (ações, links) e a mais clara (fundo, cabeçalho de tabela).</p>
        <form onSubmit={salvar} className="linha" style={{ flexWrap: "wrap" }}>
          <label>
            Nome
            <input value={nome} onChange={(e) => setNome(e.target.value)} required />
          </label>
          <label>
            Cor de destaque
            <span className="linha" style={{ alignItems: "center", gap: 8 }}>
              <span className="bolinha-cor" style={{ background: corDestaque, width: 22, height: 22 }} />
              <input type="color" value={corDestaque} onChange={(e) => setCorDestaque(e.target.value)} />
            </span>
          </label>
          <label>
            Cor clara (fundo/tabela)
            <span className="linha" style={{ alignItems: "center", gap: 8 }}>
              <span className="bolinha-cor" style={{ background: corClara, width: 22, height: 22 }} />
              <input type="color" value={corClara} onChange={(e) => setCorClara(e.target.value)} />
            </span>
          </label>
          <button type="submit" disabled={salvando}><Save size={16} /> {salvando ? "Salvando…" : "Salvar"}</button>
        </form>
      </section>

      <section className="cartao">
        <h3>Secretarias</h3>
        <form onSubmit={criarSecretaria} className="linha" style={{ flexWrap: "wrap" }}>
          <input placeholder="Nome da secretaria" value={nomeSecretaria} onChange={(e) => setNomeSecretaria(e.target.value)} required />
          <input placeholder="Sigla" value={siglaSecretaria} onChange={(e) => setSiglaSecretaria(e.target.value)} style={{ width: 120 }} required />
          <button type="submit" disabled={criandoSecretaria}><Plus size={16} /> Adicionar</button>
        </form>

        <div className="tabela-cartao" style={{ marginTop: 16 }}>
          <table>
            <thead>
              <tr>
                <th>Nome</th>
                <th>Sigla</th>
              </tr>
            </thead>
            <tbody>
              {detalhe.secretarias.map((s) => (
                <tr key={s.id}>
                  <td>{s.nome}</td>
                  <td>{s.sigla}</td>
                </tr>
              ))}
              {detalhe.secretarias.length === 0 && <tr><td colSpan={2}>Nenhuma secretaria ainda.</td></tr>}
            </tbody>
          </table>
        </div>
      </section>
    </>
  );
}
