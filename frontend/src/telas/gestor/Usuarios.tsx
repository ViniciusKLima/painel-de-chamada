import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Plus, RotateCcw, Settings, Trash2, X } from "lucide-react";
import { api, ApiError, type Grade, type UsuarioInterno } from "../../api";
import type { ContextoGestor } from "./GestorLayout";
import { useNotificar } from "../../componentes/Notificacoes";
import SelecaoGrades from "../../componentes/SelecaoGrades";

function formatarUltimoAcesso(iso: string | null): string {
  if (!iso) return "";
  return new Date(iso).toLocaleString("pt-BR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

// Modal de edição de um usuário interno (22/09, replicando GESTOR-USUARIOS.png) — nome,
// email, cargo, senha (opcional) e a grade em que está alocado. `ehEuMesmo` (22/09, pedido
// explícito) desabilita o campo Cargo quando o gestor está editando a própria conta — ele
// não pode se rebaixar/mudar o próprio papel, só nome/email/senha; editar outra pessoa
// continua permitindo trocar o cargo normalmente.
function ModalEdicaoUsuario({
  usuario,
  grades,
  ehEuMesmo,
  onFechar,
  onSalvar,
}: {
  usuario: UsuarioInterno;
  grades: Grade[];
  ehEuMesmo: boolean;
  onFechar: () => void;
  onSalvar: (dados: { nome: string; email: string; papel: string; senha?: string; gradeIds: string[] }) => Promise<void>;
}) {
  const [nome, setNome] = useState(usuario.nome);
  const [email, setEmail] = useState(usuario.email);
  const [papel, setPapel] = useState(usuario.papel);
  const [senha, setSenha] = useState("");
  const [gradeIds, setGradeIds] = useState(usuario.grades.map((g) => g.id));
  const [salvando, setSalvando] = useState(false);

  async function salvar(evento: React.FormEvent) {
    evento.preventDefault();
    setSalvando(true);
    try {
      await onSalvar({ nome, email, papel, senha: senha || undefined, gradeIds });
    } finally {
      setSalvando(false);
    }
  }

  return (
    <div className="modal-fundo" onClick={onFechar}>
      <div className="modal-cartao" onClick={(e) => e.stopPropagation()}>
        <div className="modal-cartao-topo">
          <h2>Editar usuário</h2>
          <button className="botao-icone" onClick={onFechar} title="Fechar"><X size={18} /></button>
        </div>
        <form onSubmit={salvar} className="form-vertical">
          <label>
            Nome
            <input value={nome} onChange={(e) => setNome(e.target.value)} required />
          </label>
          <label>
            Email
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label>
            Cargo
            <select value={papel} onChange={(e) => setPapel(e.target.value as UsuarioInterno["papel"])} disabled={ehEuMesmo}>
              <option value="gestor">Gestor</option>
              <option value="atendente">Atendente</option>
              <option value="recepcionista">Recepcionista</option>
            </select>
          </label>
          {ehEuMesmo && <p className="dica">Você não pode mudar o próprio cargo — peça a outro gestor, se precisar.</p>}
          <label>
            Nova senha (opcional)
            <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)} placeholder="Deixe em branco pra manter a atual" />
          </label>
          <label>
            Grades alocadas
            <SelecaoGrades grades={grades} selecionadas={gradeIds} onMudar={setGradeIds} />
          </label>
          {papel === "recepcionista" && (
            <p className="dica">Sem nenhuma marcada, a recepção continua vendo/cadastrando encaixe em qualquer grade da unidade.</p>
          )}
          <button type="submit" disabled={salvando}>{salvando ? "Salvando…" : "Salvar"}</button>
        </form>
      </div>
    </div>
  );
}

type PapelFuncionario = "atendente" | "recepcionista";

// Modal de criação de funcionário do gestor (22/09, botão que estava faltando) — sem
// perguntar prefeitura/secretaria (são sempre as do próprio gestor, obrigatório e implícito
// — pedido explícito do dono do produto: "gestor só cria dentro da sua secretaria, não
// precisa perguntar isso de novo"). Só pede cargo + grade (a grade já diz a unidade).
function ModalCriarFuncionario({
  grades,
  onFechar,
  onCriar,
}: {
  grades: Grade[];
  onFechar: () => void;
  onCriar: (dados: { nome: string; email: string; senha: string; papel: PapelFuncionario; gradeIds: string[] }) => Promise<void>;
}) {
  const [papel, setPapel] = useState<PapelFuncionario>("recepcionista");
  const [gradeIds, setGradeIds] = useState<string[]>([]);
  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [criando, setCriando] = useState(false);

  const precisaDeGrade = papel === "atendente";

  async function criar(evento: React.FormEvent) {
    evento.preventDefault();
    if (precisaDeGrade && gradeIds.length === 0) return;
    setCriando(true);
    try {
      await onCriar({ nome: nome.trim(), email: email.trim(), senha, papel, gradeIds });
    } finally {
      setCriando(false);
    }
  }

  return (
    <div className="modal-fundo" onClick={onFechar}>
      <div className="modal-cartao" onClick={(e) => e.stopPropagation()}>
        <div className="modal-cartao-topo">
          <h2>Criar funcionário</h2>
          <button className="botao-icone" onClick={onFechar} title="Fechar"><X size={18} /></button>
        </div>
        <form onSubmit={criar} className="form-vertical">
          <label>
            Cargo
            <select value={papel} onChange={(e) => setPapel(e.target.value as PapelFuncionario)}>
              <option value="recepcionista">Recepcionista</option>
              <option value="atendente">Atendente</option>
            </select>
          </label>
          <label>
            {precisaDeGrade ? "Grades" : "Grades (opcional)"}
            <SelecaoGrades grades={grades} selecionadas={gradeIds} onMudar={setGradeIds} />
          </label>
          {precisaDeGrade && grades.length === 0 && <p className="aviso">Nenhuma grade cadastrada ainda — importe uma planilha primeiro.</p>}
          {!precisaDeGrade && <p className="dica">Sem nenhuma marcada, a recepção vê/cadastra encaixe em qualquer grade da unidade.</p>}
          <label>
            Nome
            <input value={nome} onChange={(e) => setNome(e.target.value)} required />
          </label>
          <label>
            Email
            <input type="email" value={email} onChange={(e) => setEmail(e.target.value)} required />
          </label>
          <label>
            Senha
            <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)} required />
          </label>
          <button type="submit" disabled={criando || (precisaDeGrade && grades.length === 0)}>{criando ? "Criando…" : "Criar"}</button>
        </form>
      </div>
    </div>
  );
}

type AbaUsuario = "ativos" | "inativos";

export default function Usuarios() {
  const { usuario } = useOutletContext<ContextoGestor>();
  const [lista, setLista] = useState<UsuarioInterno[]>([]);
  const [grades, setGrades] = useState<Grade[]>([]);
  const [editando, setEditando] = useState<UsuarioInterno | null>(null);
  const [criandoAberto, setCriandoAberto] = useState(false);
  const [aba, setAba] = useState<AbaUsuario>("ativos");
  const notificar = useNotificar();

  // Grades excluídas nunca são uma opção válida de alocação (23/09, auditoria de
  // reativação) — `api.grades()` agora inclui excluídas pra tela "Grade de horário" poder
  // reativá-las, mas aqui (formulários de criar/editar usuário) elas precisam ficar de fora.
  const gradesAlocaveis = grades.filter((g) => !g.excluida);

  async function carregar() {
    const [u, g] = await Promise.all([api.usuarios(usuario.secretariaId), api.grades(usuario.secretariaId)]);
    setLista(u);
    setGrades(g);
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [usuario.unidadeId]);

  async function salvarUsuario(dados: { nome: string; email: string; papel: string; senha?: string; gradeIds: string[] }) {
    if (!editando) return;
    try {
      await api.atualizarUsuario(editando.id, dados);
      notificar("Usuário atualizado.");
      setEditando(null);
      await carregar();
    } catch (err) {
      notificar(err instanceof Error ? err.message : "Erro ao salvar.", "erro");
    }
  }

  async function criarFuncionario(dados: { nome: string; email: string; senha: string; papel: PapelFuncionario; gradeIds: string[] }) {
    // A unidade vem da primeira grade escolhida (todo mundo aqui é da mesma secretaria, mas
    // uma grade pertence a UMA unidade só) — pra recepcionista sem nenhuma grade marcada,
    // não dá pra criar por aqui sem saber a unidade; nesse caso pede pelo menos uma mesmo
    // sendo "opcional" pro filtro de encaixe (ela só filtra DEPOIS de já saber a unidade).
    const unidadeID = grades.find((g) => dados.gradeIds.includes(g.id))?.unidadeId ?? grades[0]?.unidadeId;
    if (!unidadeID) {
      notificar("Selecione ao menos uma grade pra saber a unidade.", "erro");
      return;
    }
    try {
      await api.criarFuncionario(unidadeID, {
        nome: dados.nome, email: dados.email, senha: dados.senha, papel: dados.papel,
        gradeIds: dados.gradeIds,
      });
      notificar("Funcionário criado.");
      setCriandoAberto(false);
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao criar funcionário.", "erro");
    }
  }

  async function excluirUsuario(u: UsuarioInterno) {
    if (!window.confirm(`Excluir ${u.nome}? Ele pode ser reativado depois na aba "Inativos" — o histórico de atendimentos continua guardado.`)) return;
    try {
      await api.excluirUsuario(u.id);
      notificar("Usuário excluído.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao excluir.", "erro");
    }
  }

  async function reativarUsuario(u: UsuarioInterno) {
    try {
      await api.reativarUsuario(u.id);
      notificar("Usuário reativado.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao reativar.", "erro");
    }
  }

  const ativos = lista.filter((u) => u.ativo);
  const inativos = lista.filter((u) => !u.ativo);
  const listaFiltrada = aba === "ativos" ? ativos : inativos;

  return (
    <div className="pagina-gestor">
      <div className="secao-cabecalho">
        <h1>Usuários internos</h1>
        <button onClick={() => setCriandoAberto(true)}>
          <Plus size={16} /> Criar funcionário
        </button>
      </div>

      <div className="abas">
        <button className={`aba ${aba === "ativos" ? "aba-ativa" : ""}`} onClick={() => setAba("ativos")}>
          Ativos ({ativos.length})
        </button>
        <button className={`aba ${aba === "inativos" ? "aba-ativa" : ""}`} onClick={() => setAba("inativos")}>
          Inativos ({inativos.length})
        </button>
      </div>

      <div className="tabela-cartao" style={{ marginTop: 16 }}>
        <table>
          <thead>
            <tr>
              <th>Nome</th>
              <th>Cargo</th>
              <th>Grades</th>
              <th>Status</th>
              <th className="coluna-acao"></th>
            </tr>
          </thead>
          <tbody>
            {listaFiltrada.map((u) => (
              <tr key={u.id}>
                <td><strong>{u.nome}</strong><div className="chamado-meta">{u.email}</div></td>
                <td style={{ textTransform: "capitalize" }}>{u.papel}</td>
                <td>
                  {u.grades.length === 0 ? "—" : (
                    <div className="etiquetas-grade">
                      {u.grades.map((g) => <span key={g.id} className="etiqueta-grade">{g.nome}</span>)}
                    </div>
                  )}
                </td>
                <td>
                  {!u.ativo ? (
                    <span className="chip ausente">inativo</span>
                  ) : u.pendente ? (
                    <span className="chip atrasado">pendente</span>
                  ) : (
                    <span className="chip no_horario">acesso em {formatarUltimoAcesso(u.ultimoAcessoEm)}</span>
                  )}
                </td>
                <td className="coluna-acao">
                  <div className="acoes-icones">
                    {!u.ativo ? (
                      <button className="botao-icone" onClick={() => reativarUsuario(u)} title="Reativar usuário">
                        <RotateCcw size={18} />
                      </button>
                    ) : (
                      <>
                        <button className="botao-icone" onClick={() => setEditando(u)} title="Configurações do usuário">
                          <Settings size={18} />
                        </button>
                        {u.id !== usuario.id && (
                          <button className="botao-icone botao-icone-perigo" onClick={() => excluirUsuario(u)} title="Excluir usuário">
                            <Trash2 size={18} />
                          </button>
                        )}
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {listaFiltrada.length === 0 && (
              <tr><td colSpan={5}>{aba === "ativos" ? "Nenhum usuário nesta unidade ainda." : "Nenhum usuário inativo."}</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {editando && (
        <ModalEdicaoUsuario
          usuario={editando}
          grades={gradesAlocaveis}
          ehEuMesmo={editando.id === usuario.id}
          onFechar={() => setEditando(null)}
          onSalvar={salvarUsuario}
        />
      )}

      {criandoAberto && (
        <ModalCriarFuncionario grades={gradesAlocaveis} onFechar={() => setCriandoAberto(false)} onCriar={criarFuncionario} />
      )}
    </div>
  );
}
