import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft, LayoutDashboard, Settings, Tv, Users } from "lucide-react";
import { api, type Prefeitura } from "../../../api";

export type ContextoPrefeitura = { prefeitura: Prefeitura };

function abaClasse({ isActive }: { isActive: boolean }) {
  return `aba ${isActive ? "aba-ativa" : ""}`;
}

// Casca das 4 abas da configuração de uma prefeitura (23/09, mudança grande pedida pelo dono
// do produto: "dashboard e usuarios internos... dentro da configuração de cada prefeitura...
// vai ter um header com 4 abas: dashboard, usuarios, settings e paineis"). Antes, "Dashboard"
// e "Usuários internos" eram itens GLOBAIS da sidebar do admin, misturando dados de todas as
// prefeituras numa lista só — agora cada prefeitura tem a sua própria central, sem misturar
// com as outras (dado real também escopado no backend, não só filtrado aqui).
//
// Reorganizado 20/10 (pedido direto: "organize melhor o nav do admin e o gestor... deixe em
// português também"): a aba "Settings" era o único texto em inglês sobrando no nav inteiro
// do admin/gestor — virou "Configurações" (rota também: `/settings` → `/configuracoes`,
// arquivo/componente `PrefeituraSettings` → `PrefeituraConfiguracoes`). As 4 abas ganharam
// ícone + texto, mesmo padrão já usado na sidebar do gestor (`GestorLayout.tsx`), em vez de
// texto solto — fica visualmente consistente entre as duas áreas administrativas do sistema.
export default function PrefeituraLayout() {
  const { prefeituraId } = useParams<{ prefeituraId: string }>();
  const navigate = useNavigate();
  const [prefeitura, setPrefeitura] = useState<Prefeitura | null>(null);

  useEffect(() => {
    if (!prefeituraId) return;
    api.prefeitura(prefeituraId).then((d) => setPrefeitura(d.prefeitura));
  }, [prefeituraId]);

  if (!prefeitura) return <div className="pagina-gestor">Carregando…</div>;

  return (
    <div className="pagina-gestor">
      <button className="botao-voltar" onClick={() => navigate("/admin/prefeituras")}>
        <ArrowLeft size={16} /> Prefeituras
      </button>
      <h1>{prefeitura.nome}</h1>

      <div className="abas">
        <NavLink to="dashboard" className={abaClasse}><LayoutDashboard size={16} /> Dashboard</NavLink>
        <NavLink to="usuarios" className={abaClasse}><Users size={16} /> Usuários</NavLink>
        <NavLink to="configuracoes" className={abaClasse}><Settings size={16} /> Configurações</NavLink>
        <NavLink to="paineis" className={abaClasse}><Tv size={16} /> Painéis</NavLink>
      </div>

      <Outlet context={{ prefeitura } satisfies ContextoPrefeitura} />
    </div>
  );
}
