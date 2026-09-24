import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { api, type ResumoDashboard } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import GraficoLinha from "../../componentes/GraficoLinha";
import ListaProgresso from "../../componentes/ListaProgresso";
import ListaProporcao from "../../componentes/ListaProporcao";

function formatarDiaCurto(data: string): string {
  const [, mes, dia] = data.split("-");
  return `${dia}/${mes}`;
}

// Dashboard de verdade (22/09, replicando GESTOR-DAHSBOARD.png) — 3 blocos de total,
// atendidos por atendente em lista de progressão, gráfico de linha atendido vs ausente
// (últimos 7 dias) e um gráfico extra (atendimentos por grade) já que a unidade pode ter
// mais de um serviço agora.
export default function Dashboard() {
  const { usuario } = useOutletContext<ContextoGestor>();
  const [resumo, setResumo] = useState<ResumoDashboard | null>(null);

  useEffect(() => {
    api.dashboard(usuario.secretariaId).then(setResumo);
  }, [usuario.secretariaId]);

  return (
    <div className="pagina-gestor">
      <h1>Dashboard</h1>

      <div className="dash-stats">
        <div className="cartao dash-stat-card">
          <span className="estatistica-rotulo">Total hoje</span>
          <span className="estatistica-valor numerico">{resumo?.totalDia ?? "—"}</span>
        </div>
        <div className="cartao dash-stat-card">
          <span className="estatistica-rotulo">Total atendido</span>
          <span className="estatistica-valor numerico sucesso">{resumo?.atendidos ?? "—"}</span>
        </div>
        <div className="cartao dash-stat-card">
          <span className="estatistica-rotulo">Total ausente</span>
          <span className="estatistica-valor numerico erro">{resumo?.ausentes ?? "—"}</span>
        </div>
      </div>

      <div className="dash-linha-2">
        <section className="cartao dash-progresso">
          <h2>Atendido por atendente</h2>
          <ListaProgresso itens={(resumo?.porAtendente ?? []).map((p) => ({ rotulo: p.nome, total: p.total }))} />
        </section>

        <section className="cartao dash-grafico">
          <h2>Atendido vs. ausente (últimos 7 dias)</h2>
          {resumo && (
            <GraficoLinha
              pontos={resumo.serieDiaria.map((p) => ({ rotulo: formatarDiaCurto(p.data), a: p.atendidos, b: p.ausentes }))}
              nomeA="Atendido"
              nomeB="Ausente"
            />
          )}
        </section>
      </div>

      <section className="cartao dash-full">
        <h2>Atendimentos por grade hoje</h2>
        <ListaProporcao itens={(resumo?.porGrade ?? []).map((g) => ({ rotulo: g.nome, atendidos: g.atendidos, ausentes: g.ausentes }))} />
      </section>
    </div>
  );
}
