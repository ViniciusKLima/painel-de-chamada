import { useEffect, useMemo, useRef, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { ArrowLeft, Expand, Link2, Search, Tv } from "lucide-react";
import { api, type Grade, type Unidade } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import { useNotificar } from "../../componentes/Notificacoes";

// Central de painéis (23/09, pedido explícito do dono do produto): como o painel de TV é
// por GRADE (cada local pode ter várias filas, cada uma com seu próprio painel público —
// ver skill painel-tv), uma secretaria com várias unidades pode ter dezenas de painéis
// diferentes. Em vez de precisar saber o link de cada um de cor, esta tela lista todos,
// com busca, e deixa abrir qualquer um "em foco" (dentro da própria tela, via iframe —
// zero duplicação da lógica do painel-tv) com um botão de tela cheia de verdade
// (Fullscreen API no próprio iframe).
export default function Paineis() {
  const { usuario } = useOutletContext<ContextoGestor>();
  const [grades, setGrades] = useState<Grade[]>([]);
  const [unidades, setUnidades] = useState<Unidade[]>([]);
  const [busca, setBusca] = useState("");
  const [gradeFocada, setGradeFocada] = useState<Grade | null>(null);
  const iframeRef = useRef<HTMLIFrameElement>(null);
  const notificar = useNotificar();

  useEffect(() => {
    Promise.all([api.grades(usuario.secretariaId), api.unidadesDaSecretaria(usuario.secretariaId)]).then(([g, u]) => {
      setGrades(g);
      setUnidades(u);
    });
  }, [usuario.secretariaId]);

  const nomeUnidade = useMemo(() => {
    const mapa = new Map(unidades.map((u) => [u.id, u.nome]));
    return (unidadeId: string) => mapa.get(unidadeId) ?? "—";
  }, [unidades]);

  // Só grades ativas fazem sentido aqui — uma grade desativada não tem fila de verdade
  // rodando, não faz sentido oferecer o painel dela pra exibição.
  const gradesFiltradas = useMemo(() => {
    const alvo = busca.trim().toLowerCase();
    const ativas = grades.filter((g) => g.ativo);
    if (!alvo) return ativas;
    return ativas.filter((g) => `${nomeUnidade(g.unidadeId)} ${g.nome} ${g.servico}`.toLowerCase().includes(alvo));
  }, [grades, busca, nomeUnidade]);

  // Agrupado por unidade (23/09, auditoria de multi-grade — "grades do mesmo local devem
  // aparecer agrupadas visualmente, não como painéis independentes"): antes cada cartão só
  // MOSTRAVA o nome da unidade dentro dele, sem agrupar de verdade os cartões da mesma
  // unidade lado a lado sob um cabeçalho comum.
  const gruposPorUnidade = useMemo(() => {
    const mapa = new Map<string, { unidadeId: string; unidadeNome: string; grades: Grade[] }>();
    for (const g of gradesFiltradas) {
      let grupo = mapa.get(g.unidadeId);
      if (!grupo) {
        grupo = { unidadeId: g.unidadeId, unidadeNome: nomeUnidade(g.unidadeId), grades: [] };
        mapa.set(g.unidadeId, grupo);
      }
      grupo.grades.push(g);
    }
    return Array.from(mapa.values()).sort((a, b) => a.unidadeNome.localeCompare(b.unidadeNome));
  }, [gradesFiltradas, nomeUnidade]);

  function copiarLink(gradeId: string) {
    const link = `${window.location.origin}/painel/${gradeId}`;
    navigator.clipboard
      .writeText(link)
      .then(() => notificar("Link do painel copiado."))
      .catch(() => notificar("Não foi possível copiar o link.", "erro"));
  }

  // Acesso pela TV do local, sem login (23/09, pedido explícito: "quero um acesso a essa
  // aba de painéis sem ser pelo gestor... tem que ser algo à parte mesmo") — cada unidade
  // tem sua própria central de painéis pública, escopada só às grades DAQUELE local físico
  // (ver telas/painel-tv/PaineisPublico.tsx e o endpoint público correspondente).
  function copiarLinkTV(unidadeId: string) {
    const link = `${window.location.origin}/tv/${unidadeId}`;
    navigator.clipboard
      .writeText(link)
      .then(() => notificar("Link de acesso pela TV copiado (sem login)."))
      .catch(() => notificar("Não foi possível copiar o link.", "erro"));
  }

  async function abrirTelaCheia() {
    try {
      await iframeRef.current?.requestFullscreen();
    } catch {
      notificar("Não foi possível entrar em tela cheia neste navegador.", "erro");
    }
  }

  if (gradeFocada) {
    return (
      <div className="pagina-gestor pagina-gestor-painel-foco">
        <div className="painel-foco-barra">
          <button onClick={() => setGradeFocada(null)}>
            <ArrowLeft size={16} /> Voltar aos painéis
          </button>
          <strong>{nomeUnidade(gradeFocada.unidadeId)} — {gradeFocada.nome}</strong>
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
    <div className="pagina-gestor">
      <h1>Painéis</h1>
      <p className="dica">Cada grade (fila) tem seu próprio painel de TV público. Escolha um pra visualizar ou colocar em tela cheia — útil pra conferir vários locais sem guardar cada link de cor.</p>

      {unidades.length > 0 && (
        <section className="cartao">
          <h3>Acesso pela TV do local</h3>
          <p className="dica">Um link por unidade, sem precisar de login de gestor — deixe fixo na TV do local pra quem for operar o painel lá.</p>
          <div className="acoes-icones" style={{ flexWrap: "wrap" }}>
            {unidades.map((u) => (
              <button key={u.id} className="botao-icone-secundario" onClick={() => copiarLinkTV(u.id)} style={{ width: "auto", padding: "8px 14px" }}>
                <Link2 size={16} /> {u.nome}
              </button>
            ))}
          </div>
        </section>
      )}

      <div className="barra-busca">
        <div className="campo-busca">
          <Search size={18} />
          <input placeholder="Buscar por unidade, grade ou serviço…" value={busca} onChange={(e) => setBusca(e.target.value)} />
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
                    <span className="dica">{g.servico}</span>
                  </div>
                  <div className="cartao-painel-hub-acoes">
                    <button onClick={() => setGradeFocada(g)}>Ver painel</button>
                    <button className="botao-icone botao-icone-secundario" onClick={() => copiarLink(g.id)} title="Copiar link público">
                      <Link2 size={18} />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          </div>
        ))}
        {gruposPorUnidade.length === 0 && <p>{grades.length === 0 ? "Nenhuma grade ativa ainda." : "Nenhum painel encontrado com essa busca."}</p>}
      </div>
    </div>
  );
}
