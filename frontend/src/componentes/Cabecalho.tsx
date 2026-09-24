import type { ReactNode } from "react";

// Nome do produto é fixo — nome da prefeitura passou a ser dinâmico (22/09, "o dia em que o
// sistema atende mais de uma prefeitura" já chegou: o admin agora cria e gerencia várias).
// Mantido como fallback só pro caso raríssimo da sessão não resolver a prefeitura ainda.
const NOME_PRODUTO = "Painel de Chamada";
const NOME_PREFEITURA_PADRAO = "Prefeitura de Jaboatão dos Guararapes";

// Cabeçalho compartilhado entre recepção/atendente/gestor — reorganizado em 2 barras (23/09,
// pedido explícito pra corrigir responsividade mobile): o cabeçalho de 3 blocos original
// (grid `1fr auto 1fr`) empilhava em coluna abaixo de 700px porque a coluna do meio ("auto")
// crescia pelo conteúdo (unidade + papel + nome) sem limite — em telas estreitas isso ou
// estourava a largura ou obrigava a virar coluna, e o dono do produto não queria mais NENHUM
// dos dois (nem coluna, nem estouro). Solução: dividir em duas barras mais simples, cada uma
// só com 2 blocos (nunca 3), o que cabe numa linha só em qualquer largura sem precisar de
// breakpoint pra reorganizar layout — só o texto trunca com reticências se precisar.
// - Barra principal (cor de marca): produto+prefeitura à esquerda, ações da tela à direita.
// - Subcabeçalho (fundo branco, logo abaixo): quem está logado à esquerda, unidade à direita.
export default function Cabecalho({
  nome,
  papel,
  unidadeNome,
  prefeituraNome,
  acoes,
}: {
  nome: string;
  papel: string;
  unidadeNome: string;
  prefeituraNome?: string | null;
  acoes: ReactNode;
}) {
  return (
    <>
      <header className="cabecalho">
        <div className="cabecalho-marca">
          <span className="cabecalho-marca-produto">{NOME_PRODUTO}</span>
          <span className="cabecalho-marca-prefeitura">{prefeituraNome ?? NOME_PREFEITURA_PADRAO}</span>
        </div>
        <div className="cabecalho-acoes">{acoes}</div>
      </header>
      <div className="subcabecalho">
        <span className="subcabecalho-usuario">{papel}: {nome}</span>
        <span className="subcabecalho-unidade">{unidadeNome}</span>
      </div>
    </>
  );
}
