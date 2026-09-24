import { useState } from "react";

// Gráfico de linha simples, em SVG puro (sem biblioteca — dashboard pequeno, não vale a
// pena a dependência) — duas séries no mesmo eixo (nunca dois eixos Y, ver skill dataviz/
// identidade-visual), cores semânticas já usadas no resto do sistema (sucesso/erro).
// Hover nas bolinhas (22/09, pedido do dono do produto) mostra um tooltip de verdade — não
// só o `<title>` nativo do navegador, que é pequeno e demora pra aparecer.
interface Ponto {
  rotulo: string;
  a: number;
  b: number;
}

export default function GraficoLinha({
  pontos,
  nomeA,
  nomeB,
}: {
  pontos: Ponto[];
  nomeA: string;
  nomeB: string;
}) {
  const [hover, setHover] = useState<{ index: number; serie: "a" | "b"; x: number; y: number } | null>(null);

  const largura = 600;
  const altura = 220;
  const margem = { topo: 16, base: 28, esquerda: 28, direita: 12 };
  const areaLargura = largura - margem.esquerda - margem.direita;
  const areaAltura = altura - margem.topo - margem.base;

  const maximo = Math.max(1, ...pontos.map((p) => Math.max(p.a, p.b)));
  const passoX = pontos.length > 1 ? areaLargura / (pontos.length - 1) : 0;

  function coordenadas(valores: number[]) {
    return valores.map((v, i) => {
      const x = margem.esquerda + i * passoX;
      const y = margem.topo + areaAltura - (v / maximo) * areaAltura;
      return [x, y] as const;
    });
  }

  const pontosA = coordenadas(pontos.map((p) => p.a));
  const pontosB = coordenadas(pontos.map((p) => p.b));

  const pontoHover = hover ? pontos[hover.index] : null;
  const valorHover = hover && pontoHover ? (hover.serie === "a" ? pontoHover.a : pontoHover.b) : null;
  const nomeHover = hover?.serie === "a" ? nomeA : nomeB;

  return (
    <div className="grafico-linha">
      <svg viewBox={`0 0 ${largura} ${altura}`} role="img" aria-label={`Gráfico de linha: ${nomeA} e ${nomeB} por dia`}>
        {/* Linhas de grade horizontais, discretas */}
        {[0, 0.5, 1].map((f) => {
          const y = margem.topo + areaAltura * (1 - f);
          return <line key={f} x1={margem.esquerda} y1={y} x2={largura - margem.direita} y2={y} className="grafico-grade" />;
        })}

        <polyline points={pontosB.map(([x, y]) => `${x},${y}`).join(" ")} className="grafico-linha-serie grafico-linha-erro" />
        <polyline points={pontosA.map(([x, y]) => `${x},${y}`).join(" ")} className="grafico-linha-serie grafico-linha-sucesso" />

        {pontos.map((p, i) => {
          const [xa, ya] = pontosA[i];
          const [, yb] = pontosB[i];
          return (
            <g key={p.rotulo}>
              {/* Alvo de hover maior e invisível, só pra facilitar acertar a bolinha */}
              <circle
                cx={xa} cy={ya} r={9} fill="transparent"
                onMouseEnter={() => setHover({ index: i, serie: "a", x: xa, y: ya })}
                onMouseLeave={() => setHover(null)}
                style={{ cursor: "pointer" }}
              />
              <circle
                cx={xa} cy={ya} r={3.5}
                className={`grafico-ponto grafico-linha-sucesso ${hover?.index === i && hover.serie === "a" ? "grafico-ponto-ativo" : ""}`}
              />
              <circle
                cx={xa} cy={yb} r={9} fill="transparent"
                onMouseEnter={() => setHover({ index: i, serie: "b", x: xa, y: yb })}
                onMouseLeave={() => setHover(null)}
                style={{ cursor: "pointer" }}
              />
              <circle
                cx={xa} cy={yb} r={3.5}
                className={`grafico-ponto grafico-linha-erro ${hover?.index === i && hover.serie === "b" ? "grafico-ponto-ativo" : ""}`}
              />
              <text x={xa} y={altura - 6} textAnchor="middle" className="grafico-eixo-texto">
                {p.rotulo}
              </text>
            </g>
          );
        })}
      </svg>

      {hover && pontoHover && (
        <div
          className="grafico-tooltip"
          style={{ left: `${(hover.x / largura) * 100}%`, top: `${(hover.y / altura) * 100}%` }}
        >
          <strong>{pontoHover.rotulo}</strong>
          <span>{nomeHover}: {valorHover}</span>
        </div>
      )}

      <div className="grafico-legenda">
        <span className="grafico-legenda-item"><i className="grafico-legenda-cor grafico-linha-sucesso-fundo" /> {nomeA}</span>
        <span className="grafico-legenda-item"><i className="grafico-legenda-cor grafico-linha-erro-fundo" /> {nomeB}</span>
      </div>
    </div>
  );
}
