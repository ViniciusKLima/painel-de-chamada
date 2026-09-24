import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { Building2, LogOut } from "lucide-react";
import { api, ApiError, type Usuario } from "../../api";
import { resetarTema } from "../../tema";

const NOME_PRODUTO = "Painel de Chamada";

export type ContextoAdmin = { usuario: Usuario };

function itemClasse({ isActive }: { isActive: boolean }) {
  return `gestor-nav-item ${isActive ? "gestor-nav-ativo" : ""}`;
}

// Casca da área do admin (22/09) — mesmo padrão de sidebar do GestorLayout (reaproveita as
// mesmas classes CSS gestor-shell/gestor-sidebar/gestor-nav, que já eram genéricas apesar
// do nome). O admin não pertence a nenhuma prefeitura/secretaria — por isso a marca some
// direto pro nome do produto, sem subtítulo de prefeitura (ele gerencia todas).
//
// Reestruturação grande (23/09, pedido explícito): "Dashboard" e "Usuários internos" eram
// itens GLOBAIS aqui, misturando dado de todas as prefeituras numa lista só — os dois
// mudaram pra dentro da configuração de CADA prefeitura (ver telas/admin/prefeitura/
// PrefeituraLayout.tsx, 4 abas: Dashboard/Usuários/Settings/Painéis). Sobra só "Prefeituras"
// na sidebar — o ponto de entrada pra escolher em qual delas mexer.
export default function AdminLayout() {
  const [usuario, setUsuario] = useState<Usuario | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    // Bug real corrigido (23/09): o admin não pertence a nenhuma prefeitura, mas sem isso
    // ele herdava a cor de quem quer que tivesse logado por último naquela aba (a cor fica
    // como estilo inline em `:root`, e nada revertia ao trocar de contexto). Ver tema.ts.
    resetarTema();
    (async () => {
      try {
        const u = await api.sessao();
        if (u.papel !== "admin") {
          navigate("/login");
          return;
        }
        setUsuario(u);
      } catch (err) {
        if (err instanceof ApiError && err.status === 401) navigate("/login");
      }
    })();
  }, [navigate]);

  if (!usuario) return <div className="tela-central">Carregando…</div>;

  return (
    <div className="gestor-shell">
      <aside className="gestor-sidebar">
        <div className="gestor-marca">
          <span className="gestor-marca-produto">{NOME_PRODUTO}</span>
          <span className="gestor-marca-prefeitura">Administração</span>
        </div>

        <nav className="gestor-nav">
          <NavLink to="/admin/prefeituras" className={itemClasse}>
            <Building2 size={18} /> Prefeituras
          </NavLink>
        </nav>

        <div className="gestor-sidebar-rodape">
          <div className="gestor-sidebar-usuario">
            <strong>{usuario.nome}</strong>
            <span>{usuario.papel}</span>
          </div>
          <button onClick={() => api.sair().then(() => navigate("/login"))}>
            <LogOut size={16} /> Sair
          </button>
        </div>
      </aside>

      <main className="gestor-conteudo">
        <Outlet context={{ usuario } satisfies ContextoAdmin} />
      </main>
    </div>
  );
}
