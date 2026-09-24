import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { api, type ResumoDashboard } from "../../../api";
import type { ContextoPrefeitura } from "./PrefeituraLayout";
import GraficoLinha from "../../../componentes/GraficoLinha";
import ListaProgresso from "../../../componentes/ListaProgresso";
import ListaProporcao from "../../../componentes/ListaProporcao";

function formatarDiaCurto(data: string): string {
  const [, mes, dia] = data.split("-");
  return `${dia}/${mes}`;
}

// Aba "Dashboard" da prefeitura, no admin (23/09) — mesmo layout do dashboard do gestor
// (telas/gestor/Dashboard.tsx), só que agregado por TODAS as secretarias desta prefeitura de
// uma vez, nunca misturando com outra prefeitura (dado já vem escopado do backend).
export default function PrefeituraDashboard() {
  const { prefeitura } = useOutletContext<ContextoPrefeitura>();
  const [resumo, setResumo] = useState<ResumoDashboard | null>(null);

  useEffect(() => {
    api.dashboardPrefeitura(prefeitura.id).then(setResumo);
  }, [prefeitura.id]);

  return (
    <div>
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
