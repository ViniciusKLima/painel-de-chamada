import { useCallback, useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { Plus, Save, Sparkles } from "lucide-react";
import { api, ApiError, type DetalhePrefeitura } from "../../../api";
import { useNotificar } from "../../../componentes/Notificacoes";

// Prefeitura de teste da integração SSO (ver skill auth-conecta-cidades) — a única que ganha
// o botão "Gerar dados de demonstração" (16/10, pedido explícito: "só a Smartândia"). O
// backend tem a mesma trava (PostGerarDadosDemo recusa qualquer outra prefeitura) — isso
// aqui é só pra nem oferecer o botão numa prefeitura real, não é a única defesa.
const SLUG_PREFEITURA_DEMO = "smartandia";

// Aba "Settings" da configuração da prefeitura (22/09, reorganizada em abas 23/09) — nome,
// as duas cores de identidade (destaque + clara, ver skill identidade-visual) e a lista de
// secretarias dela, com criação rápida. Cabeçalho/voltar/abas vêm de PrefeituraLayout, que
// envolve esta tela — aqui só o conteúdo da aba em si.
export default function PrefeituraSettings() {
  const { prefeituraId } = useParams<{ prefeituraId: string }>();
  const notificar = useNotificar();

  const [detalhe, setDetalhe] = useState<DetalhePrefeitura | null>(null);
  const [nome, setNome] = useState("");
  // Default alinhado ao schema (migration 000009) e ao token --cor-primaria do design
  // system do Conecta Cidades — só aparece até a resposta real de `api.prefeitura` chegar.
  const [corDestaque, setCorDestaque] = useState("#d52e5f");
  const [corClara, setCorClara] = useState("#fae6ec");
  const [salvando, setSalvando] = useState(false);

  const [nomeSecretaria, setNomeSecretaria] = useState("");
  const [siglaSecretaria, setSiglaSecretaria] = useState("");
  const [criandoSecretaria, setCriandoSecretaria] = useState(false);
  const [gerandoDemo, setGerandoDemo] = useState(false);

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

  async function gerarDadosDemo() {
    if (!prefeituraId) return;
    setGerandoDemo(true);
    try {
      const resultado = await api.gerarDadosDemo(prefeituraId);
      notificar(`${resultado.gerados} agendamentos de demonstração gerados.`);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao gerar dados de demonstração.", "erro");
    } finally {
      setGerandoDemo(false);
    }
  }

  if (!detalhe) return <p>Carregando…</p>;

  return (
    <>
      {detalhe.prefeitura.slug === SLUG_PREFEITURA_DEMO && (
        <section className="cartao">
          <h3>Dados de demonstração</h3>
          <p className="dica">
            Gera uma fila de agendamentos sintéticos pro dia de hoje (nomes e CPF fictícios,
            nunca de cidadão real) — a fila funciona de verdade, pra demonstrar o sistema sem
            depender de planilha importada. Rodar de novo substitui o lote anterior.
          </p>
          <button onClick={gerarDadosDemo} disabled={gerandoDemo}>
            <Sparkles size={16} /> {gerandoDemo ? "Gerando…" : "Gerar dados de demonstração"}
          </button>
        </section>
      )}

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
