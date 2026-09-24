import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import { Plus, RotateCcw, Settings, Trash2, X } from "lucide-react";
import { api, ApiError, type Grade, type Secretaria, type UsuarioAdmin } from "../../../api";
import type { ContextoPrefeitura } from "./PrefeituraLayout";
import { useNotificar } from "../../../componentes/Notificacoes";
import SelecaoGrades from "../../../componentes/SelecaoGrades";

type Cargo = "gestor" | "atendente" | "recepcionista";
type AbaUsuario = "ativos" | "inativos";

// Modal de edição — mesmo padrão das outras telas de usuário (gestor/admin global), grades
// em checkbox múltiplo (23/09).
function ModalEdicaoUsuario({
  usuario,
  grades,
  onFechar,
  onSalvar,
}: {
  usuario: UsuarioAdmin;
  grades: Grade[];
  onFechar: () => void;
  onSalvar: (dados: { nome: string; email: string; papel: string; senha?: string; gradeIds: string[] }) => Promise<void>;
}) {
  const [nome, setNome] = useState(usuario.nome);
  const [email, setEmail] = useState(usuario.email);
  const [papel, setPapel] = useState<Cargo>(usuario.papel);
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
          <p className="dica">{usuario.secretariaNome}</p>
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
            <select value={papel} onChange={(e) => setPapel(e.target.value as Cargo)}>
              <option value="gestor">Gestor</option>
              <option value="atendente">Atendente</option>
              <option value="recepcionista">Recepcionista</option>
            </select>
          </label>
          <label>
            Nova senha (opcional)
            <input type="password" value={senha} onChange={(e) => setSenha(e.target.value)} placeholder="Deixe em branco pra manter a atual" />
          </label>
          {papel !== "gestor" && (
            <label>
              Grades alocadas
              <SelecaoGrades grades={grades} selecionadas={gradeIds} onMudar={setGradeIds} />
            </label>
          )}
          <button type="submit" disabled={salvando}>{salvando ? "Salvando…" : "Salvar"}</button>
        </form>
      </div>
    </div>
  );
}

// Aba "Usuários" da prefeitura, no admin (23/09) — mesma tela que era global
// (telas/admin/UsuariosInternos.tsx), agora escopada a UMA prefeitura só: não pergunta mais
// Prefeitura no formulário de criação (é sempre esta), só Cargo → Secretaria → Grade.
export default function PrefeituraUsuarios() {
  const { prefeitura } = useOutletContext<ContextoPrefeitura>();
  const [lista, setLista] = useState<UsuarioAdmin[]>([]);
  const [secretarias, setSecretarias] = useState<Secretaria[]>([]);
  const [modalAberto, setModalAberto] = useState(false);
  const notificar = useNotificar();

  const [cargo, setCargo] = useState<Cargo>("gestor");
  const [secretariaId, setSecretariaId] = useState("");
  const [grades, setGrades] = useState<Grade[]>([]);
  const [gradeIds, setGradeIds] = useState<string[]>([]);
  const [nome, setNome] = useState("");
  const [email, setEmail] = useState("");
  const [senha, setSenha] = useState("");
  const [criando, setCriando] = useState(false);

  const [editando, setEditando] = useState<UsuarioAdmin | null>(null);
  const [gradesEdicao, setGradesEdicao] = useState<Grade[]>([]);
  const [aba, setAba] = useState<AbaUsuario>("ativos");

  async function carregar() {
    const [u, d] = await Promise.all([api.usuariosPrefeitura(prefeitura.id), api.prefeitura(prefeitura.id)]);
    setLista(u);
    setSecretarias(d.secretarias);
  }

  useEffect(() => {
    carregar();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [prefeitura.id]);

  async function abrirEdicao(u: UsuarioAdmin) {
    setEditando(u);
    // Grade excluída nunca é uma opção de alocação válida (23/09, auditoria de reativação) —
    // `api.grades()` inclui excluídas pra tela "Grade de horário" do gestor poder reativá-las,
    // mas aqui (formulário de edição) elas precisam ficar de fora.
    const lista = u.secretariaId ? await api.grades(u.secretariaId) : [];
    setGradesEdicao(lista.filter((g) => !g.excluida));
  }

  async function salvarUsuario(dados: { nome: string; email: string; papel: string; senha?: string; gradeIds: string[] }) {
    if (!editando) return;
    try {
      await api.atualizarUsuario(editando.id, dados);
      notificar("Usuário atualizado.");
      setEditando(null);
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao salvar.", "erro");
    }
  }

  async function excluirUsuario(u: UsuarioAdmin) {
    if (!window.confirm(`Excluir ${u.nome}? Ele pode ser reativado depois na aba "Inativos" — o histórico de atendimentos continua guardado.`)) return;
    try {
      await api.excluirUsuario(u.id);
      notificar("Usuário excluído.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao excluir.", "erro");
    }
  }

  async function reativarUsuario(u: UsuarioAdmin) {
    try {
      await api.reativarUsuario(u.id);
      notificar("Usuário reativado.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao reativar.", "erro");
    }
  }

  function abrirModal() {
    setModalAberto(true);
    setCargo("gestor");
    setSecretariaId(secretarias[0]?.id ?? "");
    setGradeIds([]);
    setNome("");
    setEmail("");
    setSenha("");
  }

  useEffect(() => {
    if (!secretariaId || cargo === "gestor") {
      setGrades([]);
      return;
    }
    api.grades(secretariaId).then((lista) => {
      setGrades(lista.filter((g) => !g.excluida));
      setGradeIds([]);
    });
  }, [secretariaId, cargo]);

  async function criar(evento: React.FormEvent) {
    evento.preventDefault();
    if (!secretariaId || !nome.trim() || !email.trim() || !senha) return;
    if (cargo === "atendente" && gradeIds.length === 0) {
      notificar("Escolha ao menos uma grade.", "erro");
      return;
    }
    setCriando(true);
    try {
      if (cargo === "gestor") {
        await api.criarGestor(secretariaId, { nome: nome.trim(), email: email.trim(), senha });
      } else {
        const unidadeID = grades.find((g) => gradeIds.includes(g.id))?.unidadeId ?? grades[0]?.unidadeId;
        if (!unidadeID) {
          notificar("Esta secretaria ainda não tem nenhuma grade cadastrada.", "erro");
          setCriando(false);
          return;
        }
        await api.criarFuncionario(unidadeID, { nome: nome.trim(), email: email.trim(), senha, papel: cargo, gradeIds });
      }
      setModalAberto(false);
      notificar("Usuário criado.");
      await carregar();
    } catch (err) {
      notificar(err instanceof ApiError ? err.message : "Erro ao criar usuário.", "erro");
    } finally {
      setCriando(false);
    }
  }

  function vinculo(u: UsuarioAdmin): string {
    if (u.papel === "gestor") return u.secretariaNome ?? "—";
    return u.unidadeNome ?? "—";
  }

  function rotuloCargo(papel: UsuarioAdmin["papel"]): string {
    if (papel === "gestor") return "Gestor";
    if (papel === "atendente") return "Atendente";
    return "Recepcionista";
  }

  const ativos = lista.filter((u) => u.ativo);
  const inativos = lista.filter((u) => !u.ativo);
  const listaFiltrada = aba === "ativos" ? ativos : inativos;

  return (
    <div>
      <div className="secao-cabecalho">
        <h2>Usuários</h2>
        <button onClick={abrirModal} disabled={secretarias.length === 0}>
          <Plus size={16} /> Criar usuário
        </button>
      </div>
      {secretarias.length === 0 && <p className="aviso">Esta prefeitura ainda não tem secretaria — crie uma na aba Settings primeiro.</p>}

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
              <th>Vínculo</th>
              <th>Grades</th>
              <th>Status</th>
              <th className="coluna-acao"></th>
            </tr>
          </thead>
          <tbody>
            {listaFiltrada.map((u) => (
              <tr key={u.id}>
                <td><strong>{u.nome}</strong><br /><span className="dica">{u.email}</span></td>
                <td>{rotuloCargo(u.papel)}</td>
                <td>{vinculo(u)}</td>
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
                    <span className="chip no_horario">ativo</span>
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
                        <button className="botao-icone" onClick={() => abrirEdicao(u)} title="Editar usuário">
                          <Settings size={18} />
                        </button>
                        <button className="botao-icone botao-icone-perigo" onClick={() => excluirUsuario(u)} title="Excluir usuário">
                          <Trash2 size={18} />
                        </button>
                      </>
                    )}
                  </div>
                </td>
              </tr>
            ))}
            {listaFiltrada.length === 0 && (
              <tr><td colSpan={6}>{aba === "ativos" ? "Nenhum usuário nesta prefeitura ainda." : "Nenhum usuário inativo."}</td></tr>
            )}
          </tbody>
        </table>
      </div>

      {modalAberto && (
        <div className="modal-fundo" onClick={() => setModalAberto(false)}>
          <div className="modal-cartao" onClick={(e) => e.stopPropagation()}>
            <div className="modal-cartao-topo">
              <h2>Criar usuário</h2>
              <button className="botao-icone" onClick={() => setModalAberto(false)}><X size={18} /></button>
            </div>
            <form onSubmit={criar} className="form-vertical">
              <label>
                Cargo
                <select value={cargo} onChange={(e) => setCargo(e.target.value as Cargo)}>
                  <option value="gestor">Gestor</option>
                  <option value="recepcionista">Recepcionista</option>
                  <option value="atendente">Atendente</option>
                </select>
              </label>
              <label>
                Secretaria
                <select value={secretariaId} onChange={(e) => setSecretariaId(e.target.value)} required>
                  {secretarias.map((s) => <option key={s.id} value={s.id}>{s.nome}</option>)}
                </select>
              </label>
              {cargo !== "gestor" && (
                <label>
                  {cargo === "atendente" ? "Grades" : "Grades (opcional)"}
                  <SelecaoGrades grades={grades} selecionadas={gradeIds} onMudar={setGradeIds} />
                </label>
              )}
              {cargo === "atendente" && grades.length === 0 && <p className="aviso">Esta secretaria ainda não tem grade cadastrada — importe uma planilha primeiro.</p>}
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
              <button type="submit" disabled={criando || (cargo === "atendente" && grades.length === 0)}>
                {criando ? "Criando…" : "Criar"}
              </button>
            </form>
          </div>
        </div>
      )}

      {editando && (
        <ModalEdicaoUsuario
          usuario={editando}
          grades={gradesEdicao}
          onFechar={() => setEditando(null)}
          onSalvar={salvarUsuario}
        />
      )}
    </div>
  );
}
