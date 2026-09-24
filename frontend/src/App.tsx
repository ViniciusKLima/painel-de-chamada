import { Navigate, Route, Routes } from "react-router-dom";
import Login from "./telas/login/Login";
import GestorLayout from "./telas/gestor/GestorLayout";
import Dashboard from "./telas/gestor/Dashboard";
import Grades from "./telas/gestor/Grades";
import Paineis from "./telas/gestor/Paineis";
import GradeConfiguracao from "./telas/gestor/GradeConfiguracao";
import Usuarios from "./telas/gestor/Usuarios";
import Importar from "./telas/gestor/Importar";
import Recepcao from "./telas/recepcionista/Recepcao";
import SalaDeEspera from "./telas/atendente/SalaDeEspera";
import PainelTV from "./telas/painel-tv/PainelTV";
import PaineisPublico from "./telas/painel-tv/PaineisPublico";
import AdminLayout from "./telas/admin/AdminLayout";
import Prefeituras from "./telas/admin/Prefeituras";
import PrefeituraLayout from "./telas/admin/prefeitura/PrefeituraLayout";
import PrefeituraDashboard from "./telas/admin/prefeitura/PrefeituraDashboard";
import PrefeituraUsuarios from "./telas/admin/prefeitura/PrefeituraUsuarios";
import PrefeituraSettings from "./telas/admin/prefeitura/PrefeituraSettings";
import PrefeituraPaineis from "./telas/admin/prefeitura/PrefeituraPaineis";

export default function App() {
  return (
    <Routes>
      <Route path="/" element={<Navigate to="/login" replace />} />
      <Route path="/login" element={<Login />} />
      <Route path="/admin" element={<AdminLayout />}>
        <Route index element={<Navigate to="prefeituras" replace />} />
        <Route path="prefeituras" element={<Prefeituras />} />
        <Route path="prefeituras/:prefeituraId" element={<PrefeituraLayout />}>
          <Route index element={<Navigate to="dashboard" replace />} />
          <Route path="dashboard" element={<PrefeituraDashboard />} />
          <Route path="usuarios" element={<PrefeituraUsuarios />} />
          <Route path="settings" element={<PrefeituraSettings />} />
          <Route path="paineis" element={<PrefeituraPaineis />} />
        </Route>
      </Route>
      <Route path="/gestor" element={<GestorLayout />}>
        <Route index element={<Dashboard />} />
        <Route path="grades" element={<Grades />} />
        <Route path="paineis" element={<Paineis />} />
        <Route path="grades/:gradeId" element={<GradeConfiguracao />} />
        <Route path="usuarios" element={<Usuarios />} />
        <Route path="importar" element={<Importar />} />
      </Route>
      <Route path="/recepcao" element={<Recepcao />} />
      <Route path="/sala-espera" element={<SalaDeEspera />} />
      <Route path="/painel/:gradeId" element={<PainelTV />} />
      {/* Central de painéis pública por unidade (23/09) — pra deixar fixo na TV do local,
          sem precisar de login de gestor. Ver skill painel-tv. */}
      <Route path="/tv/:unidadeId" element={<PaineisPublico />} />
    </Routes>
  );
}
