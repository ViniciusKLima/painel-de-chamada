import { useEffect, useMemo, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { ArrowLeft, Expand, Search, Tv } from "lucide-react";
import { api, type GradeResumo } from "../../api";

// Central de painéis PÚBLICA, por unidade (23/09, pedido explícito do dono do produto):
// "quero um acesso a essa aba de painéis sem ser pelo gestor... quando for acessar esse
// painel pela tv do local". A versão de dentro do gestor (`telas/gestor/Paineis.tsx`) exige
// login e mostra TODAS as unidades da secretaria; esta aqui é o equivalente sem sessão
// nenhuma, escopada a UMA unidade só (o que faz sentido fisicamente — uma TV está num local
// só) — pra deixar como link fixo na própria TV do CRAS, sem depender de ninguém logado.
// Reaproveita as mesmas classes visuais e o mesmo mecanismo de foco/tela-cheia via iframe
// (`/painel/:gradeId`, já público desde sempre) — zero duplicação da lógica do painel-tv.
export default function PaineisPublico() {
  const { unidadeId } = useParams<{ unidadeId: string }>();
  const [unidadeNome, setUnidadeNome] = useState("");
  const [grades, setGrades] = useState<GradeResumo[]>([]);
  const [carregando, setCarregando] = useState(true);
  const [erro, setErro] = useState(false);
  const [busca, setBusca] = useState("");
  const [gradeFocada, setGradeFocada] = useState<GradeResumo | null>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);

  useEffect(() => {
    if (!unidadeId) return;
    api.paineisPublicoDaUnidade(unidadeId)
      .then((r) => {
        setUnidadeNome(r.unidadeNome);
        setGrades(r.grades);
      })
      .catch(() => setErro(true))
      .finally(() => setCarregando(false));
  }, [unidadeId]);

  const gradesFiltradas = useMemo(() => {
    const alvo = busca.trim().toLowerCase();
    if (!alvo) return grades;
    return grades.filter((g) => g.nome.toLowerCase().includes(alvo));
  }, [grades, busca]);

  async function abrirTelaCheia() {
    try {
      await iframeRef.current?.requestFullscreen();
    } catch {
      // navegador sem suporte a fullscreen aqui — o botão só não faz nada, sem quebrar a tela
    }
  }

  if (carregando) return <div className="tela-central">Carregando…</div>;
  if (erro || !unidadeId) {
    return (
      <div className="tela-central">
        <div className="cartao">
          <h1>Unidade não encontrada</h1>
          <p>Confira o link — ele deve apontar pra uma unidade cadastrada no sistema.</p>
        </div>
      </div>
    );
  }

  if (gradeFocada) {
    return (
      <div className="pagina pagina-larga pagina-gestor-painel-foco" style={{ height: "100vh" }}>
        <div className="painel-foco-barra">
          <button onClick={() => setGradeFocada(null)}>
            <ArrowLeft size={16} /> Voltar aos painéis
          </button>
          <strong>{unidadeNome} — {gradeFocada.nome}</strong>
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
    <div className="pagina pagina-larga">
      <h1>{unidadeNome}</h1>
      <p className="dica">Escolha uma grade pra exibir o painel — use "Tela cheia" pra deixar rodando na TV.</p>

      {grades.length > 1 && (
        <div className="barra-busca">
          <div className="campo-busca">
            <Search size={18} />
            <input placeholder="Buscar grade…" value={busca} onChange={(e) => setBusca(e.target.value)} />
          </div>
        </div>
      )}

      <div className="grade-paineis">
        {gradesFiltradas.map((g) => (
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
        {gradesFiltradas.length === 0 && <p>{grades.length === 0 ? "Nenhuma grade ativa nesta unidade ainda." : "Nenhum painel encontrado com essa busca."}</p>}
      </div>
    </div>
  );
}
