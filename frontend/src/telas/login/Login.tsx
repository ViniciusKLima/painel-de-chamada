import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, ApiError } from "../../api";
import { resetarTema } from "../../tema";

// Login real de verdade (22/09) — voltou a ser email+senha (era um acesso rápido por botão,
// desde 20/09, porque na época o dono do produto disse "esse login não vai ser usado", com o
// SSO real da plataforma previsto pro pós-MVP). Pedido explícito agora, depois da tela de
// Admin: contas são criadas só pela gestão (admin cria gestor, gestor/admin criam
// atendente/recepcionista — ver telas do Admin) — não existe cadastro nenhum aqui, de
// propósito. O mecanismo de sessão em si nunca mudou (cookie HttpOnly via
// POST /api/v1/login), só a UI que chega até ele.
const DESTINO_POR_PAPEL: Record<string, string> = {
  admin: "/admin",
  gestor: "/gestor",
  recepcionista: "/recepcao",
  atendente: "/sala-espera",
};

// Mensagem quando o próprio api.ts redireciona pra cá sozinho (23/09) — ver o interceptor
// de sessão em api.ts pro porquê disso acontecer (cookie de sessão trocado por outra aba/
// login no mesmo navegador, ou expirado no meio do uso).
const MENSAGEM_POR_MOTIVO: Record<string, string> = {
  expirada: "Sua sessão expirou. Entre novamente.",
  alterada: "Sua sessão foi encerrada ou substituída por outro login neste navegador. Entre novamente.",
};

export default function Login() {
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [carregando, setCarregando] = useState(false);
  const [erro, setErro] = useState<string | null>(null);
  const navigate = useNavigate();
  const [parametros] = useSearchParams();
  const avisoSessao = MENSAGEM_POR_MOTIVO[parametros.get("sessao") ?? ""];

  // Bug real corrigido (23/09): a tela de login não pertence a nenhuma prefeitura, mas sem
  // isso ela herdava a cor de quem quer que tivesse logado por último naquela aba (ver
  // tema.ts, resetarTema) — mostrando a cor errada bem no primeiro contato com o sistema.
  useEffect(() => {
    resetarTema();
  }, []);

  async function entrar(evento: React.FormEvent) {
    evento.preventDefault();
    setErro(null);
    setCarregando(true);
    try {
      const usuario = await api.login(email.trim(), senha);
      navigate(DESTINO_POR_PAPEL[usuario.papel] ?? "/login");
    } catch (err) {
      setErro(err instanceof ApiError ? err.message : "Erro ao entrar.");
    } finally {
      setCarregando(false);
    }
  }

  return (
    <div className="tela-central">
      <div className="cartao">
        <h1>Painel de Chamada</h1>
        {avisoSessao && <p className="aviso">{avisoSessao}</p>}
        <form onSubmit={entrar} className="form-vertical">
          <label>
            Email
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} disabled={carregando} required autoFocus />
          </label>
          <label>
            Senha
            <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)} disabled={carregando} required />
          </label>
          <button type="submit" disabled={carregando}>{carregando ? "Entrando…" : "Entrar"}</button>
        </form>
        {erro && <p className="erro">{erro}</p>}
        <p className="dica">Não tem acesso? Fale com o gestor da sua secretaria ou com o administrador da plataforma — não há cadastro por aqui.</p>
      </div>
    </div>
  );
}
