import { useCallback, useEffect, useState } from "react";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { ArrowLeft, Plus, Power, PowerOff, RotateCcw, Save, Settings, Trash2 } from "lucide-react";
import { api, type ConfiguracaoGrade, type Grade, type Guiche, type TipoGuiche } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import { useNotificar } from "../../componentes/Notificacoes";

// Uma linha da lista de guichês/salas: aparece fechada (só nome + tipo + status) e a
// engrenagem ao lado abre o bloco de edição — decisão do dono do produto (20/09), pra não
// poluir a tela com todos os campos de todo mundo ao mesmo tempo.
function LinhaGuiche({
  guiche,
  expandida,
  onAlternarExpandir,
  onSalvar,
  onAlternarAtivo,
  onExcluir,
  onReativar,
}: {
  guiche: Guiche;
  expandida: boolean;
  onAlternarExpandir: () => void;
  onSalvar: (dados: Pick<Guiche, "nome" | "tipo" | "andar" | "capacidade" | "ativo">) => Promise<void>;
  onAlternarAtivo: () => void;
  onExcluir: () => void;
  onReativar: () => void;
}) {
  const [nome, setNome] = useState(guiche.nome);
  const [tipo, setTipo] = useState<TipoGuiche>(guiche.tipo);
  const [andar, setAndar] = useState(guiche.andar ?? "");
  const [capacidade, setCapacidade] = useState(guiche.capacidade);
  const [salvando, setSalvando] = useState(false);

  useEffect(() => {
    if (!expandida) return;
    setNome(guiche.nome);
    setTipo(guiche.tipo);
    setAndar(guiche.andar ?? "");
    setCapacidade(guiche.capacidade);
  }, [expandida, guiche]);

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    setSalvando(true);
    try {
      await onSalvar({ nome, tipo, andar: andar.trim() || null, capacidade, ativo: guiche.ativo });
    } finally {
      setSalvando(false);
    }
  }

  // Guichê excluído (23/09, auditoria de reativação): não faz sentido oferecer edição pra um
  // registro soft-deleted — só um resumo (com chip "excluído") e o botão de reativar. Depois
  // de reativado, ele volta a se comportar como qualquer outro guichê (inclusive podendo
  // estar `ativo=false`, se já estivesse assim antes de ser excluído).
  if (guiche.excluida) {
    return (
      <div className="linha-guiche">
        <div className="linha-guiche-resumo">
          <strong>{guiche.nome}</strong>
          <span className={`chip ${guiche.tipo === "sala" ? "encaixe" : "no_horario"}`}>
            {guiche.tipo === "sala" ? "sala" : "guichê"}
          </span>
          {guiche.andar && <span className="linha-guiche-andar">{guiche.andar}</span>}
          <span className="chip ausente">excluído</span>
          <button type="button" className="botao-icone" onClick={onReativar} title="Reativar guichê/sala" style={{ marginLeft: "auto" }}>
            <RotateCcw size={16} />
          </button>
        </div>
      </div>
    );
  }

  return (
    <div className="linha-guiche">
      <button type="button" className="linha-guiche-resumo" onClick={onAlternarExpandir}>
        <Settings size={16} className="engrenagem" aria-hidden />
        <strong>{guiche.nome}</strong>
        <span className={`chip ${guiche.tipo === "sala" ? "encaixe" : "no_horario"}`}>
          {guiche.tipo === "sala" ? "sala" : "guichê"}
        </span>
        {guiche.andar && <span className="linha-guiche-andar">{guiche.andar}</span>}
        <span className="linha-guiche-status">{guiche.ativo ? "ativo" : "inativo"}</span>
      </button>

      {expandida && (
        <form onSubmit={salvar} className="linha-guiche-edicao">
          <label>
            Nome
            <input value={nome} onChange={(e) => setNome(e.target.value)} required />
          </label>
          <label>
            Tipo
            <select value={tipo} onChange={(e) => setTipo(e.target.value as TipoGuiche)}>
              <option value="guiche">Guichê</option>
              <option value="sala">Sala</option>
            </select>
          </label>
          <label>
            Andar (opcional)
            <input value={andar} onChange={(e) => setAndar(e.target.value)} placeholder="Ex.: Térreo, 2º andar" />
          </label>
          <label>
            Capacidade
            <input type="number" min={1} value={capacidade} onChange={(e) => setCapacidade(Number(e.target.value))} />
          </label>
          <div className="acoes">
            <button type="submit" disabled={salvando}><Save size={16} /> {salvando ? "Salvando…" : "Salvar"}</button>
            <button type="button" onClick={onAlternarAtivo}>
              {guiche.ativo ? <PowerOff size={16} /> : <Power size={16} />}
              {guiche.ativo ? "Desativar" : "Ativar"}
            </button>
            {/* Excluir (23/09, "quero que seja possível apagar um guichê ou sala") — some da
                tela de gestão pra sempre (diferente de "Desativar", que só some da seleção do
                atendente mas pode reativar depois). Bloqueado pelo backend se for o último
                guichê/sala da grade (409 "ultimo_guiche") — toda grade precisa de pelo menos
                um. */}
            <button type="button" className="botao-icone-perigo" onClick={onExcluir} title="Excluir guichê/sala">
              <Trash2 size={16} /> Excluir
            </button>
          </div>
        </form>
      )}
    </div>
  );
}

// Configurações de uma grade específica (22/09, replicando GESTOR-GRADE-SETTINGS.png) —
// era "Gerenciar CRAS" (por unidade); agora vive numa tela própria por grade, acessada pela
// engrenagem na lista de grades.
export default function GradeConfiguracao() {
  const { gradeId } = useParams<{ gradeId: string }>();
  const { usuario } = useOutletContext<ContextoGestor>();
  const navigate = useNavigate();
  const notificar = useNotificar();

  const [config, setConfig] = useState<ConfiguracaoGrade | null>(null);
  // Copiar predefinições de outra grade (23/09) — só as OUTRAS grades da mesma secretaria,
  // carregadas uma vez pra popular o seletor.
  const [outrasGrades, setOutrasGrades] = useState<Grade[]>([]);
  const [gradeOrigemCopia, setGradeOrigemCopia] = useState("");
  const [nomeGrade, setNomeGrade] = useState("");
  const [ativa, setAtiva] = useState(true);
  const [slaChegada, setSlaChegada] = useState(60);
  const [slaAtendimento, setSlaAtendimento] = useState(20);
  const [duracaoNormal, setDuracaoNormal] = useState(30);
  const [duracaoFila, setDuracaoFila] = useState(15);
  const [repeticoesChamada, setRepeticoesChamada] = useState(3);
  const [intervaloRepeticao, setIntervaloRepeticao] = useState(5);
  const [cooldownRechamada, setCooldownRechamada] = useState(3);
  const [permiteEncaixeRecepcao, setPermiteEncaixeRecepcao] = useState(true);

  const [guicheExpandidoId, setGuicheExpandidoId] = useState<string | null>(null);
  const [novoNome, setNovoNome] = useState("");
  const [novoTipo, setNovoTipo] = useState<TipoGuiche>("guiche");
  const [novoAndar, setNovoAndar] = useState("");
  const [novaCapacidade, setNovaCapacidade] = useState(1);

  const carregar = useCallback(async (id: string) => {
    const c = await api.configuracaoGrade(id);
    setConfig(c);
    setNomeGrade(c.grade.nome);
    setAtiva(c.grade.ativo);
    setSlaChegada(c.grade.slaChegadaMinutos);
    setSlaAtendimento(c.grade.slaAtendimentoMinutos);
    setDuracaoNormal(c.grade.duracaoChamadaPainelSegundos);
    setDuracaoFila(c.grade.duracaoChamadaPainelFilaSegundos);
    setRepeticoesChamada(c.grade.repeticoesChamada);
    setIntervaloRepeticao(c.grade.intervaloRepeticaoSegundos);
    setCooldownRechamada(c.grade.cooldownRechamadaMinutos);
    setPermiteEncaixeRecepcao(c.grade.permiteEncaixeRecepcao);
  }, []);

  useEffect(() => {
    if (gradeId) carregar(gradeId);
  }, [gradeId, carregar]);

  useEffect(() => {
    // Grade excluída nunca é uma opção válida de "copiar predefinições de" (23/09, auditoria
    // de reativação) — `api.grades()` inclui excluídas pra tela "Grade de horário" poder
    // reativá-las, mas aqui elas ficam de fora.
    api.grades(usuario.secretariaId).then((lista) => setOutrasGrades(lista.filter((g) => g.id !== gradeId && !g.excluida)));
  }, [usuario.secretariaId, gradeId]);

  // Copia SLA/painel/cooldown/encaixe de outra grade pro formulário LOCAL (23/09, pedido:
  // "opção de copiar predefinições de outra grade") — não salva sozinho, só preenche os
  // campos; o gestor confere e clica em Salvar como de costume. Nome e ativo/inativo da
  // grade nunca são copiados (são identidade da própria grade, não uma predefinição).
  async function copiarPredefinicoes(origemId: string) {
    if (!origemId) return;
    try {
      const origem = await api.configuracaoGrade(origemId);
      setSlaChegada(origem.grade.slaChegadaMinutos);
      setSlaAtendimento(origem.grade.slaAtendimentoMinutos);
      setDuracaoNormal(origem.grade.duracaoChamadaPainelSegundos);
      setDuracaoFila(origem.grade.duracaoChamadaPainelFilaSegundos);
      setRepeticoesChamada(origem.grade.repeticoesChamada);
      setIntervaloRepeticao(origem.grade.intervaloRepeticaoSegundos);
      setCooldownRechamada(origem.grade.cooldownRechamadaMinutos);
      setPermiteEncaixeRecepcao(origem.grade.permiteEncaixeRecepcao);
      notificar(`Predefinições de "${origem.grade.nome}" copiadas — confira e salve.`);
    } catch {
      notificar("Não foi possível copiar as predefinições.", "erro");
    }
  }

  // Botão único e geral (23/09, pedido explícito: "não quero botão de salvar em cada bloco
  // dessas configurações, quero um botão geral, ícone em cima e botão ao final tbm pra ter
  // uma boa UX") — antes cada um dos 3 blocos (Recepção/SLA/Painel) tinha seu próprio
  // "Salvar", todos gravando o MESMO registro de configuração de qualquer forma (nunca foi
  // PATCH parcial de verdade no backend). Virou uma função sem evento de formulário, chamada
  // tanto pelo ícone no topo da página quanto pelo botão ao final.
  async function salvarConfig() {
    if (!gradeId) return;
    try {
      await api.atualizarConfiguracaoGrade(gradeId, {
        nome: nomeGrade,
        slaChegadaMinutos: slaChegada,
        slaAtendimentoMinutos: slaAtendimento,
        duracaoChamadaPainelSegundos: duracaoNormal,
        duracaoChamadaPainelFilaSegundos: duracaoFila,
        repeticoesChamada,
        intervaloRepeticaoSegundos: intervaloRepeticao,
        cooldownRechamadaMinutos: cooldownRechamada,
        permiteEncaixeRecepcao,
        ativo: ativa,
      });
      notificar("Configuração salva.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao salvar.", "erro");
    }
  }

  // Ativar/desativar virou uma ação de topo, direto no cabeçalho da página (22/09) — antes
  // vivia dentro do bloco "Identificação da grade", removido porque o nome não é mais
  // editável aqui (vem da planilha, ver skill modelo-dados). Salva na hora, sem precisar de
  // um botão "Salvar" separado — reenvia a configuração atual só trocando `ativo`.
  async function alternarAtivaGrade() {
    if (!gradeId) return;
    const novoValor = !ativa;
    try {
      await api.atualizarConfiguracaoGrade(gradeId, {
        nome: nomeGrade,
        slaChegadaMinutos: slaChegada,
        slaAtendimentoMinutos: slaAtendimento,
        duracaoChamadaPainelSegundos: duracaoNormal,
        duracaoChamadaPainelFilaSegundos: duracaoFila,
        repeticoesChamada,
        intervaloRepeticaoSegundos: intervaloRepeticao,
        cooldownRechamadaMinutos: cooldownRechamada,
        permiteEncaixeRecepcao,
        ativo: novoValor,
      });
      notificar(novoValor ? "Grade ativada." : "Grade desativada.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao atualizar.", "erro");
    }
  }

  async function criarGuiche(evento: React.FormEvent) {
    evento.preventDefault();
    if (!gradeId || !novoNome.trim()) return;
    try {
      await api.criarGuiche(gradeId, { nome: novoNome.trim(), tipo: novoTipo, andar: novoAndar.trim() || null, capacidade: novaCapacidade });
      setNovoNome("");
      setNovoTipo("guiche");
      setNovoAndar("");
      setNovaCapacidade(1);
      notificar("Guichê/sala cadastrado.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao criar guichê/sala.", "erro");
    }
  }

  async function salvarEdicaoGuiche(guicheId: string, dados: Pick<Guiche, "nome" | "tipo" | "andar" | "capacidade" | "ativo">) {
    if (!gradeId) return;
    try {
      await api.atualizarGuiche(gradeId, guicheId, dados);
      notificar("Guichê/sala salvo.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao salvar guichê/sala.", "erro");
    }
  }

  async function alternarAtivo(guicheId: string, ativo: boolean) {
    if (!gradeId) return;
    try {
      await api.atualizarAtivoGuiche(gradeId, guicheId, ativo);
      notificar(ativo ? "Guichê/sala ativado." : "Guichê/sala desativado.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao atualizar guichê/sala.", "erro");
    }
  }

  async function excluirGuiche(guiche: Guiche) {
    if (!gradeId) return;
    if (!window.confirm(`Excluir "${guiche.nome}"? Pode ser reativado depois, mas some da lista de edição até lá.`)) return;
    try {
      await api.excluirGuiche(gradeId, guiche.id);
      notificar("Guichê/sala excluído.");
      setGuicheExpandidoId(null);
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao excluir guichê/sala.", "erro");
    }
  }

  async function reativarGuicheGrade(guicheId: string) {
    if (!gradeId) return;
    try {
      await api.reativarGuiche(gradeId, guicheId);
      notificar("Guichê/sala reativado.");
      await carregar(gradeId);
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao reativar guichê/sala.", "erro");
    }
  }

  if (!config) return <div className="pagina-gestor">Carregando…</div>;

  return (
    <div className="pagina-gestor">
      <button className="botao-voltar" onClick={() => navigate("/gestor/grades")}>
        <ArrowLeft size={16} /> Grade de horário
      </button>
      <div className="secao-cabecalho">
        <div>
          <h1>{config.grade.nome}</h1>
          <p className="dica">Serviço: {config.grade.servico}</p>
        </div>
        <div className="acoes-icones">
          {outrasGrades.length > 0 && (
            <select
              value={gradeOrigemCopia}
              onChange={(e) => { setGradeOrigemCopia(e.target.value); copiarPredefinicoes(e.target.value); }}
              title="Copiar predefinições de outra grade"
            >
              <option value="">Copiar predefinições de…</option>
              {outrasGrades.map((g) => <option key={g.id} value={g.id}>{g.nome}</option>)}
            </select>
          )}
          {/* Ícone de salvar no topo (23/09) — mesmo destino do botão ao final da página,
              só pra não obrigar rolar tudo de volta pra cima depois de ajustar um campo. */}
          <button className="botao-icone" onClick={salvarConfig} title="Salvar configurações">
            <Save size={18} />
          </button>
          <button onClick={alternarAtivaGrade}>
            {ativa ? <PowerOff size={16} /> : <Power size={16} />} {ativa ? "Desativar grade" : "Ativar grade"}
          </button>
        </div>
      </div>

      <section className="cartao">
        <h3>Guichês e salas</h3>
        <form onSubmit={criarGuiche} className="linha" style={{ flexWrap: "wrap" }}>
          <input placeholder="Nome" value={novoNome} onChange={(e) => setNovoNome(e.target.value)} required />
          <select value={novoTipo} onChange={(e) => setNovoTipo(e.target.value as TipoGuiche)}>
            <option value="guiche">Guichê</option>
            <option value="sala">Sala</option>
          </select>
          <input placeholder="Andar (opcional)" value={novoAndar} onChange={(e) => setNovoAndar(e.target.value)} />
          <input
            type="number"
            min={1}
            title="Capacidade"
            value={novaCapacidade}
            onChange={(e) => setNovaCapacidade(Number(e.target.value))}
            style={{ width: 90 }}
          />
          <button type="submit"><Plus size={16} /> Adicionar</button>
        </form>

        <div className="lista-guiches">
          {config.guiches.map((g) => (
            <LinhaGuiche
              key={g.id}
              guiche={g}
              expandida={guicheExpandidoId === g.id}
              onAlternarExpandir={() => setGuicheExpandidoId(guicheExpandidoId === g.id ? null : g.id)}
              onSalvar={(dados) => salvarEdicaoGuiche(g.id, dados)}
              onAlternarAtivo={() => alternarAtivo(g.id, !g.ativo)}
              onExcluir={() => excluirGuiche(g)}
              onReativar={() => reativarGuicheGrade(g.id)}
            />
          ))}
          {config.guiches.length === 0 && <p>Nenhum guichê ou sala cadastrado ainda.</p>}
        </div>
      </section>

      <section className="cartao">
        <h3>Recepção</h3>
        <p className="dica">Nome de exibição (usado no painel de TV e em outras telas), tolerância de chegada e se a recepção pode cadastrar encaixe nesta grade.</p>
        <div className="linha" style={{ flexWrap: "wrap" }}>
          <label>
            Nome de exibição
            <input value={nomeGrade} onChange={(e) => setNomeGrade(e.target.value)} required />
          </label>
          <label>
            Tolerância de chegada (min)
            <input type="number" min={1} value={slaChegada} onChange={(e) => setSlaChegada(Number(e.target.value))} />
          </label>
          <label className="linha" style={{ alignItems: "center", gap: 8 }}>
            <input type="checkbox" checked={permiteEncaixeRecepcao} onChange={(e) => setPermiteEncaixeRecepcao(e.target.checked)} />
            Recepção pode cadastrar encaixe
          </label>
        </div>
        {/* Gestor/admin sempre podem cadastrar encaixe, mesmo com a caixa acima desmarcada
            (23/09, pedido explícito) — só restringe a recepção. */}
        {!permiteEncaixeRecepcao && <p className="dica">Gestor e admin continuam podendo cadastrar encaixe normalmente — a restrição vale só pra recepção.</p>}
      </section>

      <section className="cartao">
        <h3>SLA do atendimento</h3>
        <p className="dica">
          Tolerância de atendimento — quanto tempo total o atendente tem antes do sistema
          marcar ausência automática (o tempo mínimo entre chamadas abaixo é uma fatia desse
          mesmo prazo, não algo somado a mais). Mudar esses valores aqui vale a partir de
          agora pra qualquer chamada nova — quem já está no meio de uma chamada continua com
          o prazo antigo até a próxima.
        </p>
        <div className="linha" style={{ flexWrap: "wrap" }}>
          <label>
            Tolerância de atendimento (min)
            <input type="number" min={1} value={slaAtendimento} onChange={(e) => setSlaAtendimento(Number(e.target.value))} />
          </label>
          <label>
            Tempo mínimo entre chamadas (min)
            <input type="number" min={1} max={30} value={cooldownRechamada} onChange={(e) => setCooldownRechamada(Number(e.target.value))} />
          </label>
        </div>
      </section>

      <section className="cartao">
        <h3>Configurações do painel</h3>
        <div className="linha" style={{ flexWrap: "wrap" }}>
          <label>
            Exibição no painel (s)
            <input type="number" min={1} value={duracaoNormal} onChange={(e) => setDuracaoNormal(Number(e.target.value))} />
          </label>
          <label>
            Exibição c/ fila (s)
            <input type="number" min={1} value={duracaoFila} onChange={(e) => setDuracaoFila(Number(e.target.value))} />
          </label>
          <label>
            Repetições da chamada
            <input type="number" min={1} max={10} value={repeticoesChamada} onChange={(e) => setRepeticoesChamada(Number(e.target.value))} />
          </label>
          <label>
            Intervalo entre repetições (s)
            <input type="number" min={1} max={120} value={intervaloRepeticao} onChange={(e) => setIntervaloRepeticao(Number(e.target.value))} />
          </label>
        </div>
      </section>

      <button onClick={salvarConfig} style={{ width: "100%", justifyContent: "center", padding: "14px" }}>
        <Save size={18} /> Salvar configurações
      </button>
    </div>
  );
}
