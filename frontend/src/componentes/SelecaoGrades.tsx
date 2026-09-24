import type { Grade } from "../api";

// Seletor de múltiplas grades (23/09, "atendente ou recepção podem ser alocados em mais de
// uma grade") — checkboxes em vez de um <select multiple> nativo (péssima UX: precisa de
// ctrl+clique, não é óbvio pra maioria das pessoas). Usado na criação e edição de usuário,
// tanto no gestor quanto no admin.
export default function SelecaoGrades({
  grades,
  selecionadas,
  onMudar,
}: {
  grades: Grade[];
  selecionadas: string[];
  onMudar: (gradeIds: string[]) => void;
}) {
  function alternar(gradeId: string) {
    if (selecionadas.includes(gradeId)) {
      onMudar(selecionadas.filter((id) => id !== gradeId));
    } else {
      onMudar([...selecionadas, gradeId]);
    }
  }

  return (
    <div className="lista-selecao-grades">
      {grades.map((g) => (
        <label key={g.id} className="linha-selecao-grade">
          <input type="checkbox" checked={selecionadas.includes(g.id)} onChange={() => alternar(g.id)} />
          {g.nome}
        </label>
      ))}
      {grades.length === 0 && <p className="dica">Nenhuma grade cadastrada ainda.</p>}
    </div>
  );
}
