// Lista de barras por grade mostrando a PROPORÇÃO atendido vs. ausente de cada uma (22/09,
// ajustado — antes mostrava só o volume total de atendidos, sem contar ausência). Cada barra
// preenche 100% da linha, dividida em duas cores — a métrica é a taxa de conclusão daquela
// grade, não o volume comparado às outras (ver skill dataviz: cor por papel/semântica, não
// por rank).
export default function ListaProporcao({ itens }: { itens: { rotulo: string; atendidos: number; ausentes: number }[] }) {
  return (
    <div className="lista-progresso">
      {itens.map((item) => {
        const total = item.atendidos + item.ausentes;
        const percAtendido = total > 0 ? Math.round((item.atendidos / total) * 100) : 0;
        const percAusente = 100 - percAtendido;
        return (
          <div key={item.rotulo} className="lista-progresso-item">
            <div className="lista-progresso-cabecalho">
              <span>{item.rotulo}</span>
              <span className="numerico">
                {item.atendidos} atendido{item.atendidos === 1 ? "" : "s"} ({percAtendido}%) · {item.ausentes} ausente{item.ausentes === 1 ? "" : "s"} ({percAusente}%)
              </span>
            </div>
            <div className="barra-proporcao">
              <div className="barra-proporcao-atendido" style={{ width: `${percAtendido}%` }} />
              <div className="barra-proporcao-ausente" style={{ width: `${percAusente}%` }} />
            </div>
          </div>
        );
      })}
      {itens.length === 0 && <p className="aviso">Nenhum atendimento concluído hoje ainda.</p>}
    </div>
  );
}
