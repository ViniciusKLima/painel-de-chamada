import { useEffect, useState } from "react";
import { useNavigate, useOutletContext } from "react-router-dom";
import { Link2, RotateCcw, Settings, Trash2 } from "lucide-react";
import { api, ApiError, type Grade } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import { useNotificar } from "../../componentes/Notificacoes";

type AbaGrade = "ativas" | "inativas" | "excluidas";

// Lista de grades (22/09, replicando GESTOR-GRADES.png) — cada linha é um serviço/fila
// dentro da unidade, com botão de copiar o link do painel público e engrenagem pra abrir as
// configurações individuais daquela grade (guichês, SLA, painel). Abas Ativas/Inativas
// (23/09, pedido explícito) — antes tudo vinha numa lista só com um chip "inativa" solto.
// Aba "Excluídas" (23/09, auditoria de reativação) — antes de existir, excluir uma grade não
// tinha volta nenhuma pela UI; agora aparece aqui, só com o botão de reativar (as outras
// ações — copiar link, configurar — não fazem sentido pra uma grade que ainda está excluída).
export default function Grades() {
  const { usuario } = useOutletContext<ContextoGestor>();
  const [grades, setGrades] = useState<Grade[]>([]);
  const [aba, setAba] = useState<AbaGrade>("ativas");
  const notificar = useNotificar();
  const navigate = useNavigate();

  async function carregar() {
    setGrades(await api.grades(usuario.secretariaId));
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [usuario.secretariaId]);

  function copiarLink(gradeId: string) {
    const link = `${window.location.origin}/painel/${gradeId}`;
    navigator.clipboard
      .writeText(link)
      .then(() => notificar("Link do painel copiado."))
      .catch(() => notificar("Não foi possível copiar o link.", "erro"));
  }

  async function excluirGrade(g: Grade) {
    if (!window.confirm(`Excluir a grade "${g.nome}"? Ela pode ser reativada depois na aba "Excluídas" — o histórico de agendamentos continua guardado.`)) return;
    try {
      await api.excluirGrade(g.id);
      notificar("Grade excluída.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao excluir.", "erro");
    }
  }

  async function reativarGrade(g: Grade) {
    try {
      await api.reativarGrade(g.id);
      notificar("Grade reativada.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao reativar.", "erro");
    }
  }

  const ativas = grades.filter((g) => !g.excluida && g.ativo);
  const inativas = grades.filter((g) => !g.excluida && !g.ativo);
  const excluidas = grades.filter((g) => g.excluida);
  const listaFiltrada = aba === "ativas" ? ativas : aba === "inativas" ? inativas : excluidas;

  return (
    <div className="pagina-gestor">
      <h1>Grade de horário</h1>

      <div className="abas">
        <button className={`aba ${aba === "ativas" ? "aba-ativa" : ""}`} onClick={() => setAba("ativas")}>
          Ativas ({ativas.length})
        </button>
        <button className={`aba ${aba === "inativas" ? "aba-ativa" : ""}`} onClick={() => setAba("inativas")}>
          Inativas ({inativas.length})
        </button>
        <button className={`aba ${aba === "excluidas" ? "aba-ativa" : ""}`} onClick={() => setAba("excluidas")}>
          Excluídas ({excluidas.length})
        </button>
      </div>

      <div className="tabela-cartao" style={{ marginTop: 16 }}>
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Serviço</th>
              <th className="coluna-acao"></th>
            </tr>
          </thead>
          <tbody>
            {listaFiltrada.map((g) => (
              <tr key={g.id}>
                <td><strong>{g.nome}</strong></td>
                <td>{g.servico}</td>
                <td className="coluna-acao">
                  <div className="acoes-icones">
                    {aba === "excluidas" ? (
                      <button className="botao-icone" onClick={() => reativarGrade(g)} title="Reativar grade">
                        <RotateCcw size={18} />
                      </button>
                    ) : (
                      <>
                        <button className="botao-icone botao-icone-secundario" onClick={() => copiarLink(g.id)} title="Copiar link do painel">
                          <Link2 size={18} />
                        </button>
                        <button className="botao-icone" onClick={() => navigate(`/gestor/grades/${g.id}`)} title="Configurações da grade">
                          <Settings size={18} />
                        </button>
                        <button className="botao-icone botao-icone-perigo" onClick={() => excluirGrade(g)} title="Excluir grade">
                          <Trash2 size={18} />
                        </button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {listaFiltrada.length === 0 && (
              <tr><td colSpan={3}>{aba === "ativas" ? "Nenhuma grade ativa — importe uma planilha pra criar automaticamente." : aba === "inativas" ? "Nenhuma grade inativa." : "Nenhuma grade excluída."}</td></tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
