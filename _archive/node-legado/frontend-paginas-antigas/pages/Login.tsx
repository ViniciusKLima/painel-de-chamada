import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { api, definirUsuarioId } from "../api";

// Login provisório enquanto o SSO/JWT da plataforma Jornada não está disponível
// (ver CLAUDE.md > Decisões em aberto). Aqui só se cola o UUID de um Usuario
// já existente no banco (gerado pelo `npm run db:seed` no backend).
export default function Login() {
  const [usuarioId, setUsuarioId] = useState("");
  const [erro, setErro] = useState<string | null>(null);
  const [carregando, setCarregando] = useState(false);
  const navigate = useNavigate();

  async function entrar(evento: React.FormEvent) {
    evento.preventDefault();
    setErro(null);
    setCarregando(true);
    definirUsuarioId(usuarioId.trim());
    try {
      await api.me();
      navigate("/fila");
    } catch {
      setErro("UUID de usuário inválido. Rode `npm run db:seed` no backend e cole um dos UUIDs impressos.");
    } finally {
      setCarregando(false);
    }
  }

  return (
    <div className="tela-central">
      <form className="cartao" onSubmit={entrar}>
        <h1>Painel de Chamada</h1>
        <p className="aviso">
          Login provisório de desenvolvimento — sem SSO ainda. Cole o UUID de um usuário
          (gerado pelo <code>npm run db:seed</code>).
        </p>
        <input
          type="text"
          placeholder="UUID do usuário"
          value={usuarioId}
          onChange={(e) => setUsuarioId(e.target.value)}
          required
        />
        {erro && <p className="erro">{erro}</p>}
        <button type="submit" disabled={carregando}>
          {carregando ? "Entrando…" : "Entrar"}
        </button>
      </form>
    </div>
  );
}
