import { useEffect, useState } from "react";
import { useNavigate, useSearchParams } from "react-router-dom";
import { api, ApiError } from "../../api";
import { resetarTema } from "../../tema";

// Login por e-mail+senha REMOVIDO POR COMPLETO (22/10, pedido direto do dono do produto:
// "derrube toda aquela autenticação antiga que tinhamos feito e suba apenas com o metodo
// jwt"). A ÚNICA forma de entrar agora é o login integrado via portal (SSO real da Conecta
// Cidades, ver skill auth-conecta-cidades) — a prefeitura abre esta tela com `?token=` na
// URL. Sem esse token (ou com um token inválido/expirado), a tela mostra um aviso fixo
// pra voltar pelo portal — não existe mais formulário nenhum aqui.
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
  expirada: "Sua sessão expirou.",
  alterada: "Sua sessão foi encerrada ou substituída por outro login neste navegador.",
};

export default function Login() {
  const [erro, setErro] = useState<string | null>(null);
  // Entrando via portal (SSO)? Mostra só "entrando…" enquanto troca o token.
  const [entrandoViaPortal, setEntrandoViaPortal] = useState(() => new URLSearchParams(window.location.search).has("token"));
  const navigate = useNavigate();
  const [parametros] = useSearchParams();
  const avisoSessao = MENSAGEM_POR_MOTIVO[parametros.get("sessao") ?? ""];

  // Bug real corrigido (23/09): a tela de login não pertence a nenhuma prefeitura, mas sem
  // isso ela herdava a cor de quem quer que tivesse logado por último naquela aba (ver
  // tema.ts, resetarTema) — mostrando a cor errada bem no primeiro contato com o sistema.
  useEffect(() => {
    resetarTema();
  }, []);

  // Login integrado via portal (ver skill auth-conecta-cidades): a prefeitura abre esta tela
  // com `?token=<JWT do portal>` na URL. Lido na guarda de entrada da tela (não num
  // componente aninhado), e a URL é limpa IMEDIATAMENTE — antes mesmo da resposta do backend
  // chegar — porque a query string fica no histórico do navegador, no `Referer` e em log de
  // proxy; não pode sobreviver além do instante da troca, sucesso ou falha.
  useEffect(() => {
    const token = new URLSearchParams(window.location.search).get("token");
    if (!token) return;
    window.history.replaceState(null, "", window.location.pathname);
    (async () => {
      try {
        const usuario = await api.loginPortal(token);
        navigate(DESTINO_POR_PAPEL[usuario.papel] ?? "/login", { replace: true });
      } catch (err) {
        setErro(err instanceof ApiError ? err.message : "Não foi possível entrar pelo portal.");
        setEntrandoViaPortal(false);
      }
    })();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  if (entrandoViaPortal) {
    return (
      <div className="tela-central">
        <div className="cartao">
          <h1>Painel de Chamada</h1>
          <p className="dica">Entrando pelo portal…</p>
        </div>
      </div>
    );
  }

  // Tela única pra qualquer caso de entrada que não seja um `?token=` válido: acesso direto
  // sem token, ou um token que falhou (expirado, inválido, papel sem acesso etc. — `erro`
  // traz o motivo específico quando veio de uma tentativa real). Visual de "painel de TV sem
  // sinal" — pedido explícito do dono do produto — reforça o tema do produto em si (um
  // sistema de painel de chamada) até no próprio aviso de acesso. TV ilustrada em 3D de
  // verdade (perspective + preserve-3d, 3 faces reais — frente/lado/topo — não só um ícone
  // com um retângulo atrás, 2ª rodada de ajuste depois do feedback de que ficou "estranho,
  // só ícone e texto"), com o chiado estático correndo DENTRO da tela da TV, não no corpo
  // inteiro. Detalhe completo das classes em index.css.
  return (
    <div className="tela-central">
      <div className="acesso-portal">
        <div className="tv-cena" aria-hidden="true">
          <div className="tv-sombra-piso" />
          <div className="tv-antena tv-antena-esquerda" />
          <div className="tv-antena tv-antena-direita" />
          <div className="tv-caixa">
            <div className="tv-face tv-face-topo" />
            <div className="tv-face tv-face-lado" />
            <div className="tv-face tv-face-frente">
              <div className="tv-tela">
                <span className="tv-tela-reflexo" />
                <span className="tv-tela-legenda">SEM SINAL</span>
              </div>
              <div className="tv-painel">
                <span className="tv-luz" />
                <span className="tv-botao" />
                <span className="tv-botao" />
              </div>
            </div>
          </div>
          <div className="tv-pescoco" />
          <div className="tv-pe" />
        </div>
        <div className="acesso-portal-texto">
          <h1>Acesse pelo portal da sua prefeitura</h1>
          <p>Para entrar no Painel de Chamada, abra o sistema a partir do portal da sua prefeitura.</p>
          {(avisoSessao || erro) && <p className="dica">{avisoSessao ?? erro}</p>}
        </div>
      </div>
    </div>
  );
}
