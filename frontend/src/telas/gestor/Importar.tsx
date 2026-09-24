import { useRef, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { CheckCircle2, FileSpreadsheet, Upload } from "lucide-react";
import { api, ApiError, type RespostaImportacao, type RespostaPreview } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import { useNotificar } from "../../componentes/Notificacoes";

function formatarDia(data: string): string {
  const [ano, mes, dia] = data.split("-");
  return `${dia}/${mes}/${ano}`;
}

// Importação em duas etapas (22/09, replicando GESTOR-IMPORTAR.png + pedido explícito):
// arrastar/escolher o arquivo dispara uma ANÁLISE (sem gravar nada) — o gestor vê quantos
// agendamentos por dia, quantos erros, e se alguma unidade/grade nova seria criada — e só
// depois confirma de verdade.
export default function Importar() {
  const { usuario } = useOutletContext<ContextoGestor>();
  const [arquivo, setArquivo] = useState<File | null>(null);
  const [analisando, setAnalisando] = useState(false);
  const [preview, setPreview] = useState<RespostaPreview | null>(null);
  const [confirmando, setConfirmando] = useState(false);
  const [resultado, setResultado] = useState<RespostaImportacao | null>(null);
  const [arrastando, setArrastando] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);
  const notificar = useNotificar();

  async function analisar(f: File) {
    setArquivo(f);
    setPreview(null);
    setResultado(null);
    setAnalisando(true);
    try {
      const resposta = await api.previewPlanilha(usuario.secretariaId, f);
      setPreview(resposta);
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao analisar planilha.", "erro");
      setArquivo(null);
    } finally {
      setAnalisando(false);
    }
  }

  async function confirmar() {
    if (!arquivo) return;
    setConfirmando(true);
    try {
      const resposta = await api.importarPlanilha(usuario.secretariaId, arquivo);
      setResultado(resposta);
      setPreview(null);
      notificar("Planilha importada.");
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao importar.", "erro");
    } finally {
      setConfirmando(false);
    }
  }

  function recomecar() {
    setArquivo(null);
    setPreview(null);
    setResultado(null);
    if (inputRef.current) inputRef.current.value = "";
  }

  function aoSoltar(evento: React.DragEvent<HTMLDivElement>) {
    evento.preventDefault();
    setArrastando(false);
    const f = evento.dataTransfer.files?.[0];
    if (f) analisar(f);
  }

  return (
    <div className="pagina-gestor">
      <h1>Importar agendamentos</h1>

      {!preview && !resultado && (
        <div
          className={`zona-upload ${arrastando ? "zona-upload-ativa" : ""}`}
          onDragOver={(e) => { e.preventDefault(); setArrastando(true); }}
          onDragLeave={() => setArrastando(false)}
          onDrop={aoSoltar}
          onClick={() => inputRef.current?.click()}
        >
          <input
            ref={inputRef}
            type="file"
            accept=".xlsx"
            hidden
            onChange={(e) => { const f = e.target.files?.[0]; if (f) analisar(f); }}
          />
          <FileSpreadsheet size={40} />
          <p>{analisando ? "Analisando planilha…" : "Arraste ou clique pra importar o Excel dos agendamentos do dia"}</p>
        </div>
      )}

      {preview && arquivo && (
        <section className="cartao">
          <h2>Confirmar importação de "{arquivo.name}"</h2>

          <div className="estatisticas" style={{ marginBottom: 16 }}>
            <div className="estatistica">
              <span className="estatistica-valor numerico sucesso">{preview.totalValidas}</span>
              <span className="estatistica-rotulo">agendamentos válidos</span>
            </div>
            <div className="estatistica">
              <span className="estatistica-valor numerico">{preview.totalCancelados}</span>
              <span className="estatistica-rotulo">cancelados</span>
            </div>
            <div className="estatistica">
              <span className="estatistica-valor numerico erro">{preview.totalErros}</span>
              <span className="estatistica-rotulo">com erro</span>
            </div>
          </div>

          {(preview.porDia ?? []).length > 0 && (
            <>
              <h3>Agendamentos por dia</h3>
              <ul className="lista-atendidos-hoje">
                {preview.porDia.map((p) => (
                  <li key={p.data}><span>{formatarDia(p.data)}</span><span className="numerico">{p.total} agendamentos</span></li>
                ))}
              </ul>
            </>
          )}

          {(preview.unidadesNovas ?? []).length > 0 && (
            <p className="aviso">Unidade(s) nova(s), criada(s) automaticamente: {preview.unidadesNovas.join(", ")}.</p>
          )}
          {(preview.gradesNovas ?? []).length > 0 && (
            <p className="aviso">
              Grade(s) nova(s): {preview.gradesNovas.map((g) => `${g.unidade} — ${g.servico || "Geral"}`).join(", ")}.
            </p>
          )}

          {(preview.erros ?? []).length > 0 && (
            <>
              <h3>Erros ({preview.erros.length})</h3>
              <div className="tabela-cartao">
                <table>
                  <thead><tr><th>Linha</th><th>Motivo</th></tr></thead>
                  <tbody>
                    {preview.erros.slice(0, 20).map((e, i) => <tr key={i}><td className="numerico">{e.linha}</td><td>{e.motivo}</td></tr>)}
                  </tbody>
                </table>
              </div>
              {preview.erros.length > 20 && <p className="aviso">Mostrando as primeiras 20 de {preview.erros.length} linhas com erro.</p>}
            </>
          )}

          <div className="acoes" style={{ marginTop: 16 }}>
            <button onClick={confirmar} disabled={confirmando || preview.totalValidas === 0}>
              <CheckCircle2 size={16} /> {confirmando ? "Importando…" : `Confirmar ${preview.totalValidas} agendamentos`}
            </button>
            <button onClick={recomecar} disabled={confirmando}>Cancelar</button>
          </div>
        </section>
      )}

      {resultado && (
        <section className="cartao">
          <h2>Importação concluída</h2>
          <p>
            <strong className="numerico">{resultado.totalCriados}</strong> agendamentos criados de{" "}
            <strong className="numerico">{resultado.totalLinhas}</strong> linhas ({resultado.totalCancelados} cancelados,{" "}
            {resultado.totalErros} com erro).
          </p>
          <button onClick={recomecar}><Upload size={16} /> Importar outra planilha</button>
        </section>
      )}
    </div>
  );
}
