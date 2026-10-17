// Aplica em runtime as duas cores de identidade configuráveis pelo admin por prefeitura
// (22/09) — "a principal que é o destaque, e a mais clarinha... que serve como background e
// cabeçalho de tabelas", nas palavras do dono do produto. Mapeiam direto pros tokens que já
// existiam (ver skill identidade-visual): destaque -> --cor-primaria (+ uma variante mais
// escura calculada, pro hover dos botões), clara -> --cor-info-fundo/--cor-info (fundo +
// cabeçalho de tabela). Aplicado como custom property no elemento raiz, sem precisar de
// build separado por prefeitura.
function escurecer(hex: string, fator: number): string {
  const num = parseInt(hex.replace("#", ""), 16);
  const r = Math.max(0, Math.round(((num >> 16) & 0xff) * (1 - fator)));
  const g = Math.max(0, Math.round(((num >> 8) & 0xff) * (1 - fator)));
  const b = Math.max(0, Math.round((num & 0xff) * (1 - fator)));
  return `#${((1 << 24) + (r << 16) + (g << 8) + b).toString(16).slice(1)}`;
}

// Bug real corrigido (23/09, achado pedido do dono do produto: "a cor de Jaboatão só
// interfere apenas Jaboatão, e assim com cada cidade" — mas o admin estava herdando a cor
// de quem quer que tivesse logado por último naquela aba). `aplicarTemaPrefeitura` grava a
// cor como estilo INLINE em `:root`, que nunca era desfeito ao trocar de contexto — numa
// SPA, navegar de uma tela de gestor/atendente/recepção (com cor aplicada) pra uma tela sem
// prefeitura nenhuma (login, admin) deixava a cor antiga "grudada", porque nada revertia o
// inline style de volta pro CSS padrão. `resetarTema` remove essas propriedades inline,
// deixando o `:root` do CSS (a cor NEUTRA da própria plataforma, não de nenhuma cidade)
// voltar a valer — chamado no mount de Login e AdminLayout, as duas telas que nunca têm
// (e nunca devem ter) a cor de uma prefeitura específica.
export function resetarTema() {
  const raiz = document.documentElement.style;
  for (const prop of ["--cor-primaria", "--cor-primaria-forte", "--cor-info", "--cor-info-fundo", "--cor-marca"]) {
    raiz.removeProperty(prop);
  }
}

export function aplicarTemaPrefeitura(corDestaque: string | null | undefined, corClara: string | null | undefined) {
  if (!corDestaque || !corClara) return;
  const raiz = document.documentElement.style;
  raiz.setProperty("--cor-primaria", corDestaque);
  raiz.setProperty("--cor-primaria-forte", escurecer(corDestaque, 0.18));
  raiz.setProperty("--cor-info", corDestaque);
  raiz.setProperty("--cor-info-fundo", corClara);
  // Cabeçalho (Cabecalho.tsx) e sidebar do gestor (GestorLayout.tsx) seguem a cor de
  // destaque da prefeitura de quem está logado (23/09) — não faz mais sentido todo mundo ver
  // o mesmo teal-marinho fixo não importa a prefeitura. Alinhamento com o design system do
  // Conecta Cidades (24/09): o fundo do cabeçalho/sidebar virou CHAPADO (sem degradê — "o
  // produto usa cor chapada", proibição explícita deles), então só precisa de UM tom
  // (`--cor-marca` = a própria `corDestaque`) — o segundo tom clareado que alimentava o
  // degradê (`--cor-marca-clara`) não existe mais, nem a função `clarear` que o calculava.
  raiz.setProperty("--cor-marca", corDestaque);
}

// Painel de TV (22/09, pedido explícito do dono do produto): o fundo da tela passa a usar a
// cor clara/secundária da prefeitura — os dois blocos de conteúdo (chamada atual + tabela)
// continuam brancos por cima, pra manter contraste e legibilidade a distância. O acento
// (protocolo, barra de tempo) usa a cor de destaque. Só os tokens --painel-*, não mexe nos
// tokens operacionais --cor-* (o painel mantém seu tema isolado, ver skill identidade-visual).
export function aplicarTemaPainel(corDestaque: string | null | undefined, corClara: string | null | undefined) {
  const raiz = document.documentElement.style;
  if (corClara) raiz.setProperty("--painel-fundo", corClara);
  if (corDestaque) raiz.setProperty("--painel-acento", corDestaque);
}
