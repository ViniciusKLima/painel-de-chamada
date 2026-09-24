import { useEffect, useRef, useState } from "react";
import { useParams } from "react-router-dom";
import { CheckCircle2, Clock, Volume2, XCircle } from "lucide-react";
import { api, type ChamadaPainel, type RespostaPainel } from "../../api";
import { aplicarTemaPainel } from "../../tema";

// Reduzido de 4s pra 2s (20/09) — o dono do produto sentiu delay entre chamar e o painel
// reagir; 2s é rápido o bastante sem exagerar no tráfego pro backend.
const INTERVALO_POLLING_MS = 2000;
// Padrões usados só até a primeira resposta do backend chegar — dali em diante o valor
// real vem de unidade.repeticoesChamada/intervaloRepeticaoSegundos, configurável pelo
// gestor na tela "Gerenciar CRAS" (decisão 20/09).
const REPETICOES_PADRAO = 3;
const INTERVALO_REPETICAO_PADRAO_MS = 5000;

// Ícone de status na lista de últimos chamados (22/09: trocado de caractere Unicode solto
// pra ícone de verdade, mesma biblioteca — lucide-react — usada no resto do sistema).
// Quarto ícone (23/09, cinza) distinguindo, dentro do status "chamado", quem ainda está
// normalmente sendo esperado pelo próprio atendente (ehOrfao=false, relógio cinza — calmo,
// "dentro do prazo") de quem já foi abandonado e precisa que outro atendente assuma
// (ehOrfao=true, mantém o ícone âmbar de sempre — merece mais atenção).
function StatusIcone({ status, ehOrfao }: { status: string; ehOrfao: boolean }) {
  if (status === "atendido") return <span className="status-icone atendido"><CheckCircle2 /></span>;
  if (status === "ausente") return <span className="status-icone ausente"><XCircle /></span>;
  if (ehOrfao) return <span className="status-icone chamado"><Volume2 /></span>;
  return <span className="status-icone aguardando"><Clock /></span>;
}

// Barra de contagem regressiva (22/09, pedido explícito do dono do produto) — mostra
// visualmente quanto tempo falta pra essa chamada sair de destaque. `duracaoRestante` é
// calculada UMA VEZ na primeira renderização (useState com inicializador lazy) a partir do
// `expiraEm` que o backend manda — nunca recalculada nos polls seguintes da mesma chamada,
// senão a animação CSS reiniciaria a cada 2s em vez de correr suave até o fim. O componente
// só remonta (e recalcula) quando a `key` (protocolo+guichê) muda pra uma chamada nova —
// ver isso no componente pai. A decisão de quando trocar a exibição continua 100% do
// backend; esta barra é só decorativa/informativa.
function BarraTempo({ expiraEm, duracaoTotalSegundos }: { expiraEm: string; duracaoTotalSegundos: number }) {
  const [duracaoRestante] = useState(() => {
    const restanteMs = new Date(expiraEm).getTime() - Date.now();
    return Math.max(0.2, restanteMs / 1000 || duracaoTotalSegundos);
  });
  return (
    <div className="chamada-atual-barra-fundo" aria-hidden>
      <div className="chamada-atual-barra-progresso" style={{ animationDuration: `${duracaoRestante}s` }} />
    </div>
  );
}

// Números por extenso (1-20, faixa realista de guichês) — falar "um" em vez de dígito
// solto evita a leitura colada tipo "guicheum" que o Web Speech dava com "Guichê 1".
const numerosPorExtenso: Record<number, string> = {
  1: "um", 2: "dois", 3: "três", 4: "quatro", 5: "cinco", 6: "seis", 7: "sete", 8: "oito",
  9: "nove", 10: "dez", 11: "onze", 12: "doze", 13: "treze", 14: "catorze", 15: "quinze",
  16: "dezesseis", 17: "dezessete", 18: "dezoito", 19: "dezenove", 20: "vinte",
};

// Formata o texto do guichê ("Guichê 1" -> "Guichê: um") com o número por extenso — o
// dígito solto saía colado ("guicheum"). Decisão 20/09 (dono do produto ouviu ao vivo e
// pediu de volta): simples, só "Prefixo: número por extenso" — sem "número"/"por favor"
// no meio, que soava artificial. O dois-pontos já é suficiente pra separar sem colar.
function textoGuicheFalado(guiche: string): string {
  const m = guiche.trim().match(/^(.*?)\s*(\d+)\s*$/);
  if (!m) return guiche.trim();
  const prefixo = m[1].trim();
  const numero = parseInt(m[2], 10);
  const numeroFalado = numerosPorExtenso[numero] ?? String(numero);
  return `${prefixo}: ${numeroFalado}`;
}

// Web Speech API (nativa do navegador) — decisão de 19/09, ver skill painel-tv: zero
// custo, roda no frontend, sem serviço de TTS de nuvem. Fala nome completo + guichê, sem
// protocolo (estranho de ouvir em voz alta).
function falarChamada(nome: string, guiche: string) {
  try {
    if (!("speechSynthesis" in window)) return;
    const utter = new SpeechSynthesisUtterance(`${nome}. ${textoGuicheFalado(guiche)}`);
    utter.lang = "pt-BR";
    const vozPt = window.speechSynthesis.getVoices().find((v) => v.lang.startsWith("pt"));
    if (vozPt) utter.voice = vozPt;
    window.speechSynthesis.speak(utter);
  } catch {
    // dispositivo sem suporte — a exibição visual continua funcionando normalmente
  }
}

function tocarSinal() {
  try {
    const Ctx = window.AudioContext || (window as unknown as { webkitAudioContext: typeof AudioContext }).webkitAudioContext;
    const ctx = new Ctx();
    const osc = ctx.createOscillator();
    const gain = ctx.createGain();
    osc.frequency.value = 880;
    gain.gain.setValueAtTime(0.15, ctx.currentTime);
    gain.gain.exponentialRampToValueAtTime(0.001, ctx.currentTime + 0.4);
    osc.connect(gain).connect(ctx.destination);
    osc.start();
    osc.stop(ctx.currentTime + 0.4);
  } catch {
    // sem suporte a áudio — segue só com a fala/exibição
  }
}

export default function PainelTV() {
  const { gradeId } = useParams<{ gradeId: string }>();
  const [dados, setDados] = useState<RespostaPainel | null>(null);
  const [erro, setErro] = useState<string | null>(null);
  const [audioLiberado, setAudioLiberado] = useState(false);
  const ultimaChamadaAnunciada = useRef<string | null>(null);
  const intervaloRepeticaoRef = useRef<number | null>(null);
  // Identifica qual chamada de iniciarAnuncio é a "vigente" — necessário porque o
  // StrictMode do React (dev) pode invocar o efeito de polling duas vezes de forma quase
  // simultânea em certas transições; sem essa trava, dois intervalos de repetição
  // ficavam vivos ao mesmo tempo e a voz saía dobrada.
  const idAnuncioVigenteRef = useRef(0);

  function pararRepeticao() {
    if (intervaloRepeticaoRef.current !== null) {
      window.clearInterval(intervaloRepeticaoRef.current);
      intervaloRepeticaoRef.current = null;
    }
  }

  // Anuncia a chamada e repete no intervalo configurado até o máximo de vezes configurado
  // pelo gestor (unidade.repeticoesChamada/intervaloRepeticaoSegundos) — chamada nova
  // cancela qualquer repetição pendente da anterior.
  //
  // Bug real corrigido (23/09): parar o setInterval de repetição (pararRepeticao) não para
  // uma fala que o navegador já está falando/enfileirou — o Web Speech por padrão ENFILEIRA
  // utterances em vez de cancelar a anterior. Sob chamadas em sequência rápida (vários
  // atendentes chamando quase ao mesmo tempo), isso causava o painel já mostrar a pessoa B
  // na tela enquanto o áudio ainda terminava de falar o nome de A ("chamar o nome de uma
  // pessoa com outra na tela"). `speechSynthesis.cancel()` descarta imediatamente qualquer
  // fala em andamento ou enfileirada antes de começar o novo anúncio — cada chamada vira
  // uma unidade de fato única (sinal + fala + repetições), nunca sobreposta pela anterior.
  function iniciarAnuncio(nome: string, guiche: string, maxRepeticoes: number, intervaloMs: number) {
    pararRepeticao();
    try {
      window.speechSynthesis?.cancel();
    } catch {
      // sem suporte — segue sem cancelar, a próxima fala ainda vai ser enfileirada normalmente
    }
    const meuId = ++idAnuncioVigenteRef.current;
    let repeticoes = 0;
    const anunciarUmaVez = () => {
      if (idAnuncioVigenteRef.current !== meuId) return;
      tocarSinal();
      falarChamada(nome, guiche);
      repeticoes++;
      if (repeticoes >= maxRepeticoes) pararRepeticao();
    };
    // Adiada por um tick: se outra chamada a iniciarAnuncio acontecer no mesmo instante
    // (ex.: efeito duplicado pelo StrictMode em dev), o meuId dela já vai ter "vencido"
    // antes deste callback rodar, e este anuncio antigo é descartado silenciosamente.
    window.setTimeout(anunciarUmaVez, 0);
    if (maxRepeticoes > 1) {
      intervaloRepeticaoRef.current = window.setInterval(anunciarUmaVez, intervaloMs);
    }
  }

  useEffect(() => () => pararRepeticao(), []);

  useEffect(() => {
    if (!gradeId) return;
    let cancelado = false;

    async function carregar() {
      try {
        const resposta = await api.painel(gradeId!);
        if (cancelado) return;
        setDados(resposta);
        setErro(null);
        aplicarTemaPainel(resposta.corDestaque, resposta.corClara);

        const atual = resposta.chamadaAtual;
        const chaveAtual = atual ? `${atual.protocolo}-${atual.guiche}` : null;
        if (audioLiberado && atual && chaveAtual !== ultimaChamadaAnunciada.current) {
          const maxRepeticoes = resposta.unidade.repeticoesChamada || REPETICOES_PADRAO;
          const intervaloMs = (resposta.unidade.intervaloRepeticaoSegundos || INTERVALO_REPETICAO_PADRAO_MS / 1000) * 1000;
          iniciarAnuncio(atual.nome, atual.guiche, maxRepeticoes, intervaloMs);
        }
        if (!atual) pararRepeticao();
        ultimaChamadaAnunciada.current = chaveAtual;
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
  }, [gradeId, audioLiberado]);

  if (!audioLiberado) {
    return (
      <div className="painel-tv painel-tv-gate" onClick={() => setAudioLiberado(true)}>
        <p className="chamada-atual-vazio">Toque na tela pra iniciar o painel</p>
      </div>
    );
  }

  // Três modos (23/09, era só ativo/descanso): "ativo" tem uma chamada nova de verdade em
  // destaque (com narração e barra de tempo); "aguardando" é quando não há nada novo pra
  // anunciar mas ainda existe gente chamada, não órfã, dentro do prazo — fica visível como
  // blocos em vez de sumir pra tabela (pedido explícito do dono do produto); "descanso" é só
  // quando não sobra nada disso, a tabela cheia ocupa a tela toda. O backend já exclui da
  // tabela (`ultimasChamadas`) quem estiver ocupando o destaque (atual OU aguardando) — o
  // frontend não precisa mais fazer esse slice manualmente.
  const emAtivo = !!dados?.chamadaAtual;
  // Backend devolve aguardando ordenado do mais antigo pro mais novo (é assim que ele decide
  // quem entra quando tem mais candidatos que o limite — quem espera há mais tempo tem
  // prioridade de aparecer). Pra EXIBIÇÃO, a ordem é invertida (23/09, pedido explícito: "2
  // ficam em coluna, mais recente em cima") — o mais recente sempre ocupa a primeira posição
  // do grid (topo/esquerda), não importa quantos blocos existam.
  const aguardando = [...(dados?.aguardando ?? [])].reverse();
  const emAguardando = !emAtivo && aguardando.length > 0;
  const emDescanso = !emAtivo && !emAguardando;
  const listaTabela = dados?.ultimasChamadas ?? [];
  const chaveAtual = dados?.chamadaAtual ? `${dados.chamadaAtual.protocolo}-${dados.chamadaAtual.guiche}` : null;
  const modo = emAtivo ? "modo-ativo" : emAguardando ? "modo-aguardando" : "modo-descanso";

  return (
    <div className="painel-tv">
      <header className="painel-tv-topo">{dados ? dados.unidade.nome : "Carregando…"}</header>
      {erro && <p className="erro">{erro}</p>}

      {/* Estrutura única (não alterna árvores diferentes) pra permitir a transição
          animada entre modos — só as proporções e opacidade mudam. */}
      <div className={`painel-tv-corpo ${modo}`}>
        <section className="chamada-atual" aria-hidden={emDescanso}>
          {emAtivo && dados?.chamadaAtual && (
            <>
              <span className="chamada-atual-nome">{dados.chamadaAtual.nome}</span>
              <div className="chamada-atual-linha">
                <span className="chamada-atual-protocolo">{dados.chamadaAtual.protocolo}</span>
                <span className="chamada-atual-guiche">{dados.chamadaAtual.guiche}</span>
              </div>
              <BarraTempo key={chaveAtual} expiraEm={dados.chamadaAtual.expiraEm} duracaoTotalSegundos={dados.chamadaAtual.duracaoTotalSegundos} />
            </>
          )}
          {emAguardando && (
            <div className={`aguardando-grade aguardando-grade-${aguardando.length}`}>
              {aguardando.map((ag) => (
                <div className="aguardando-bloco" key={`${ag.protocolo}-${ag.guiche}`}>
                  <span className="aguardando-nome">{ag.nome}</span>
                  {/* Nome, protocolo e guichê empilhados em coluna, reaproveitando as MESMAS
                      classes da tabela padrão (historico-protocolo/historico-guiche-pill) —
                      pedido 23/09: "mesma estrutura do padrão", em vez de uma tipografia
                      bespoke só pra este bloco. */}
                  <span className="historico-protocolo">{ag.protocolo}</span>
                  <span className="historico-guiche-pill">{ag.guiche}</span>
                </div>
              ))}
            </div>
          )}
        </section>

        <section className="historico">
          <h2>Últimas chamadas</h2>
          {listaTabela.length > 0 ? (
            <div className="historico-tabela">
              <div className="historico-cabecalho" aria-hidden>
                <span></span>
                <span>Nome</span>
                <span>Protocolo</span>
                <span>Guichê</span>
              </div>
              <ul>
                {listaTabela.map((c: ChamadaPainel) => (
                  <li key={c.id}>
                    <StatusIcone status={c.status} ehOrfao={c.ehOrfao} />
                    <span className="historico-nome">{c.nome}</span>
                    <span className="historico-protocolo">{c.protocolo}</span>
                    <span className="historico-guiche-pill">{c.guiche}</span>
                  </li>
                ))}
              </ul>
            </div>
          ) : (
            <p className="chamada-atual-vazio">Nenhuma chamada ainda</p>
          )}
        </section>
      </div>
    </div>
  );
}
