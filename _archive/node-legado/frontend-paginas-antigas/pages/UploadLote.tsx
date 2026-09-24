import { useEffect, useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { api, type RespostaUpload, type Usuario } from "../api";

export default function UploadLote() {
  const [usuario, setUsuario] = useState<Usuario | null>(null);
  const [arquivo, setArquivo] = useState<File | null>(null);
  const [enviando, setEnviando] = useState(false);
  const [resultado, setResultado] = useState<RespostaUpload | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    api.me().then(setUsuario).catch(() => navigate("/login"));
  }, [navigate]);

  async function enviar(evento: React.FormEvent) {
    evento.preventDefault();
    if (!arquivo || !usuario) return;
    setEnviando(true);
    setErro(null);
    setResultado(null);
    try {
      const resposta = await api.uploadLote(usuario.secretariaId, arquivo);
      setResultado(resposta);
    } catch (err) {
      setErro(err instanceof Error ? err.message : "Erro ao subir planilha.");
    } finally {
      setEnviando(false);
    }
  }

  if (!usuario) return <div className="tela-central">Carregando…</div>;

  return (
    <div className="pagina">
      <header className="topo">
        <div>
          <strong>{usuario.nome}</strong> — {usuario.secretaria.sigla}
        </div>
        <nav>
          <Link to="/fila">Voltar pra fila</Link>
        </nav>
      </header>

      <section className="cartao">
        <h2>Subir planilha do dia (.csv ou .xlsx)</h2>
        <p className="aviso">
          Colunas esperadas: <code>protocolo</code>, <code>nome</code>, <code>horario</code> (HH:mm) e{" "}
          <code>guiche</code> (opcional).
        </p>
        <form onSubmit={enviar} className="linha">
          <input
            type="file"
            accept=".csv,.xlsx"
            onChange={(e) => setArquivo(e.target.files?.[0] ?? null)}
            required
          />
          <button type="submit" disabled={!arquivo || enviando}>
            {enviando ? "Enviando…" : "Enviar"}
          </button>
        </form>

        {erro && <p className="erro">{erro}</p>}

        {resultado && (
          <div className="resultado-upload">
            <p>
              <strong>{resultado.totalCriados}</strong> agendamentos criados de{" "}
              <strong>{resultado.totalLinhas}</strong> linhas ({resultado.totalErros} com erro).
            </p>
            {resultado.erros.length > 0 && (
              <table>
                <thead>
                  <tr>
                    <th>Linha</th>
                    <th>Motivo</th>
                  </tr>
                </thead>
                <tbody>
                  {resultado.erros.map((e, i) => (
                    <tr key={i}>
                      <td>{e.linha}</td>
                      <td>{e.motivo}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            )}
          </div>
        )}
      </section>
    </div>
  );
}
