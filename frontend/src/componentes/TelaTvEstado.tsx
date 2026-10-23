import type { ReactNode } from "react";

// TV ilustrada em 3D (ver index.css, classes `.tv-*`) extraída do fluxo de login pra um
// componente reaproveitável (23/10, pedido direto do dono do produto: "ja era bom que tivesse
// uma variação dessa tela pra quando tivesse umas outras situações de token invalido ou entao
// uma dessa pra erro 404 e essas coisas no sistema, ja podemos reaproveitar") — qualquer tela
// de "estado vazio"/erro do sistema (sessão sem token, 404, etc.) usa a mesma ilustração, só
// trocando o texto e a legenda que aparece na tela da TV.
type TelaTvEstadoProps = {
  titulo: string;
  mensagem: string;
  /** Texto curto que aparece "na tela" da TV, estilo legenda de transmissão fora do ar.
   * Default "SEM SINAL" (caso de sessão/token) — uma 404 usa "ERRO 404", por exemplo. */
  legenda?: string;
  /** Conteúdo extra abaixo da mensagem principal (ex: aviso de sessão expirada, link de volta). */
  children?: ReactNode;
};

export function IlustracaoTV({ legenda = "SEM SINAL" }: { legenda?: string }) {
  return (
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
            <span className="tv-tela-legenda">{legenda}</span>
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
  );
}

export default function TelaTvEstado({ titulo, mensagem, legenda, children }: TelaTvEstadoProps) {
  return (
    <div className="tela-central">
      <div className="acesso-portal">
        <IlustracaoTV legenda={legenda} />
        <div className="acesso-portal-texto">
          <h1>{titulo}</h1>
          <p>{mensagem}</p>
          {children}
        </div>
      </div>
    </div>
  );
}
