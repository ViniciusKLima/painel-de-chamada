import { useEffect, useMemo, useRef, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { ArrowLeft, Expand, Search, Tv } from "lucide-react";
import { api } from "../../../api";
import type { ContextoPrefeitura } from "./PrefeituraLayout";

type GradeComUnidade = { id: string; nome: string; unidadeId: string; unidadeNome: string };

// Aba "Painéis" da prefeitura, no admin (23/09) — mesma ideia da central de painéis do
// gestor (telas/gestor/Paineis.tsx), agregada por TODAS as secretarias/unidades desta
// prefeitura de uma vez (o gestor só vê a própria secretaria).
export default function PrefeituraPaineis() {
  const { prefeitura } = useOutletContext<ContextoPrefeitura>();
  const [grades, setGrades] = useState<GradeComUnidade[]>([]);
  const [busca, setBusca] = useState("");
  const [gradeFocada, setGradeFocada] = useState<GradeComUnidade | null>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    api.paineisPrefeitura(prefeitura.id).then(setGrades);
  }, [prefeitura.id]);

  const gradesFiltradas = useMemo(() => {
    const alvo = busca.trim().toLowerCase();
    if (!alvo) return grades;
    return grades.filter((g) => `${g.unidadeNome} ${g.nome}`.toLowerCase().includes(alvo));
  }, [grades, busca]);

  // Agrupado por unidade (23/09, mesma regra da central de painéis do gestor — ver
  // telas/gestor/Paineis.tsx): grades do mesmo local físico ficam visualmente juntas, mesmo
  // vindo de secretarias diferentes dentro da prefeitura.
  const gruposPorUnidade = useMemo(() => {
    const mapa = new Map<string, { unidadeId: string; unidadeNome: string; grades: GradeComUnidade[] }>();
    for (const g of gradesFiltradas) {
      let grupo = mapa.get(g.unidadeId);
      if (!grupo) {
        grupo = { unidadeId: g.unidadeId, unidadeNome: g.unidadeNome, grades: [] };
        mapa.set(g.unidadeId, grupo);
      }
      grupo.grades.push(g);
    }
    return Array.from(mapa.values()).sort((a, b) => a.unidadeNome.localeCompare(b.unidadeNome));
  }, [gradesFiltradas]);

  async function abrirTelaCheia() {
    try {
      await iframeRef.current?.requestFullscreen();
    } catch {
      // navegador sem suporte a fullscreen aqui — só não faz nada
    }
  }

  if (gradeFocada) {
    return (
      <div className="pagina-gestor-painel-foco" style={{ height: "70vh" }}>
        <div className="painel-foco-barra">
          <button onClick={() => setGradeFocada(null)}>
            <ArrowLeft size={16} /> Voltar aos painéis
          </button>
          <strong>{gradeFocada.unidadeNome} — {gradeFocada.nome}</strong>
          <button onClick={abrirTelaCheia}>
            <Expand size={16} /> Tela cheia
          </button>
        </div>
        <iframe
          ref={iframeRef}
          key={gradeFocada.id}
          src={`/painel/${gradeFocada.id}`}
          className="painel-foco-iframe"
          allowFullScreen
          title={`Painel — ${gradeFocada.nome}`}
        />
      </div>
    );
  }

  return (
    <div>
      <p className="dica">Todas as grades ativas de todas as secretarias desta prefeitura — escolha uma pra visualizar ou colocar em tela cheia.</p>

      <div className="barra-busca">
        <div className="campo-busca">
          <Search size={18} />
          <input placeholder="Buscar por unidade ou grade…" value={busca} onChange={(e) => setBusca(e.target.value)} />
        </div>
      </div>

      <div className="grupos-unidade">
        {gruposPorUnidade.map((grupo) => (
          <div key={grupo.unidadeId} className="grupo-unidade">
            <h3 className="grupo-unidade-titulo">{grupo.unidadeNome}</h3>
            <div className="grade-paineis">
              {grupo.grades.map((g) => (
                <div className="cartao-painel-hub" key={g.id}>
                  <div className="cartao-painel-hub-icone"><Tv size={22} /></div>
                  <div className="cartao-painel-hub-texto">
                    <strong>{g.nome}</strong>
                  </div>
                  <div className="cartao-painel-hub-acoes">
                    <button onClick={() => setGradeFocada(g)}>Ver painel</button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
        {gruposPorUnidade.length === 0 && <p>{grades.length === 0 ? "Nenhuma grade ativa nesta prefeitura ainda." : "Nenhum painel encontrado com essa busca."}</p>}
      </div>
    </div>
  );
}
