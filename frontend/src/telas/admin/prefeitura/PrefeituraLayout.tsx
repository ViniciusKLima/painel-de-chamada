import { useEffect, useState } from "react";
import { NavLink, Outlet, useNavigate, useParams } from "react-router-dom";
import { ArrowLeft } from "lucide-react";
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
        <NavLink to="dashboard" className={abaClasse}>Dashboard</NavLink>
        <NavLink to="usuarios" className={abaClasse}>Usuários</NavLink>
        <NavLink to="settings" className={abaClasse}>Settings</NavLink>
        <NavLink to="paineis" className={abaClasse}>Painéis</NavLink>
      </div>

      <Outlet context={{ prefeitura } satisfies ContextoPrefeitura} />
    </div>
  );
}
