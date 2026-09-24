import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate } from "react-router-dom";
import { CalendarClock, LayoutDashboard, LogOut, Tv, Upload, Users } from "lucide-react";
import { api, ApiError, type Usuario } from "../../api";
import { aplicarTemaPrefeitura } from "../../tema";

const NOME_PRODUTO = "Painel de Chamada";
const NOME_PREFEITURA_PADRAO = "Prefeitura de Jaboatão dos Guararapes";

// Dentro da área do gestor, secretariaId é sempre garantido (todo gestor tem um — ver
// backfill da migration 000005) — o tipo estreitado evita `!` espalhado em cada tela que
// consome o contexto.
export type UsuarioGestor = Usuario & { secretariaId: string };
export type ContextoGestor = { usuario: UsuarioGestor };

function itemClasse({ isActive }: { isActive: boolean }) {
  return `gestor-nav-item ${isActive ? "gestor-nav-ativo" : ""}`;
}

// Casca da área do gestor (22/09) — sidebar fixa com navegação clássica, replicando o
// desenho anexado (GESTOR-DAHSBOARD.png e demais): marca em cima, 3 seções principais,
// "Importar" separado por um respiro maior, e o usuário logado + sair no rodapé da barra.
export default function GestorLayout() {
  const [usuario, setUsuario] = useState<Usuario | null>(null);
  const navigate = useNavigate();

  useEffect(() => {
    (async () => {
      try {
        const u = await api.sessao();
        if (u.papel !== "gestor") {
          navigate("/login");
          return;
        }
        setUsuario(u);
        aplicarTemaPrefeitura(u.corDestaque, u.corClara);
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
          <span className="gestor-marca-prefeitura">{usuario.prefeituraNome ?? NOME_PREFEITURA_PADRAO}</span>
        </div>

        <nav className="gestor-nav">
          <NavLink to="/gestor" end className={itemClasse}>
            <LayoutDashboard size={18} /> Dashboard
          </NavLink>
          <NavLink to="/gestor/usuarios" className={itemClasse}>
            <Users size={18} /> Usuários internos
          </NavLink>
          <NavLink to="/gestor/grades" className={itemClasse}>
            <CalendarClock size={18} /> Grade de horário
          </NavLink>
          <NavLink to="/gestor/paineis" className={itemClasse}>
            <Tv size={18} /> Painéis
          </NavLink>
          <NavLink to="/gestor/importar" className={({ isActive }) => `${itemClasse({ isActive })} gestor-nav-separado`}>
            <Upload size={18} /> Importar
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
        <Outlet context={{ usuario: usuario as UsuarioGestor } satisfies ContextoGestor} />
      </main>
    </div>
  );
}
