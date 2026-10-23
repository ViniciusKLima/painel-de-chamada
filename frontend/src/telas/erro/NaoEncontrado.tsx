import { Link } from "react-router-dom";
import TelaTvEstado from "../../componentes/TelaTvEstado";

// Página 404 (23/10) — reaproveita a TV ilustrada do login (`TelaTvEstado`), só trocando a
// legenda "na tela" pra "ERRO 404" e o texto. Rota catch-all registrada em App.tsx
// (`path="*"`), que antes não existia (uma URL sem rota correspondente caía numa página em
// branco, sem nenhum feedback).
export default function NaoEncontrado() {
  return (
    <TelaTvEstado
      titulo="Página não encontrada"
      mensagem="O endereço que você acessou não existe ou foi movido."
      legenda="ERRO 404"
    >
      <Link to="/login" className="acesso-portal-link">
        Voltar pro início
      </Link>
    </TelaTvEstado>
  );
}
