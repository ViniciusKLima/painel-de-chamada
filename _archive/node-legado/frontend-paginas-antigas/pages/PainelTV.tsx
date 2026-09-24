import { useEffect, useState } from "react";
import { useParams } from "react-router-dom";
import { api, type ChamadaPainel } from "../api";

const INTERVALO_POLLING_MS = 4000;

export default function PainelTV() {
  const { secretariaId } = useParams<{ secretariaId: string }>();
  const [secretaria, setSecretaria] = useState<{ nome: string; sigla: string } | null>(null);
  const [chamadaAtual, setChamadaAtual] = useState<ChamadaPainel | null>(null);
  const [ultimas, setUltimas] = useState<ChamadaPainel[]>([]);
  const [erro, setErro] = useState<string | null>(null);

  useEffect(() => {
    if (!secretariaId) return;

    let cancelado = false;

    async function carregar() {
      try {
        const dados = await api.painel(secretariaId!);
        if (cancelado) return;
        setSecretaria(dados.secretaria);
        setChamadaAtual(dados.chamadaAtual);
        setUltimas(dados.ultimasChamadas);
        setErro(null);
      } catch {
        if (!cancelado) setErro("Não foi possível carregar o painel.");
      }
    }

    carregar();
    const id = setInterval(carregar, INTERVALO_POLLING_MS);
    return () => {
      cancelado = true;
      clearInterval(id);
    };
  }, [secretariaId]);

  return (
    <div className="painel-tv">
      <header className="painel-tv-topo">{secretaria ? `${secretaria.nome} (${secretaria.sigla})` : "Carregando…"}</header>

      {erro && <p className="erro">{erro}</p>}

      <section className="chamada-atual">
        {chamadaAtual ? (
          <>
            <span className="chamada-atual-protocolo">{chamadaAtual.protocolo}</span>
            <span className="chamada-atual-nome">{chamadaAtual.nome}</span>
            <span className="chamada-atual-guiche">{chamadaAtual.guiche}</span>
          </>
        ) : (
          <span className="chamada-atual-vazio">Nenhuma chamada ainda</span>
        )}
      </section>

      <section className="historico">
        <h2>Últimas chamadas</h2>
        <ul>
          {ultimas.slice(1).map((c) => (
            <li key={c.id}>
              <span>{c.protocolo}</span>
              <span>{c.nome}</span>
              <span>{c.guiche}</span>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
