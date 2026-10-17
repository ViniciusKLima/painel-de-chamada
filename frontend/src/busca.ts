// Normalização de texto pra busca local (minúsculas, sem pontuação/espaço) — extraído pra cá
// em 16/10 porque passou a ter uma terceira cópia idêntica (Recepcao.tsx, SalaDeEspera.tsx,
// Grades.tsx), conforme a própria regra já documentada na skill identidade-visual ("se
// aparecer uma terceira vez, vale extrair pra um util compartilhado").
export function normalizarBusca(texto: string): string {
  return texto.toLowerCase().replace(/[.\-/\s]/g, "");
}
