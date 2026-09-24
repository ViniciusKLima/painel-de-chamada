// Lista de barras horizontais com porcentagem — usada em "Atendidos por atendente" e
// "Atendimentos por grade" no dashboard do gestor. Cor única (não é dado categórico com
// identidade própria, é a mesma métrica repetida por pessoa/grade — ver skill dataviz).
export default function ListaProgresso({ itens }: { itens: { rotulo: string; total: number }[] }) {
  const maximo = Math.max(1, ...itens.map((i) => i.total));
  const soma = itens.reduce((acc, i) => acc + i.total, 0);

  return (
    <div className="lista-progresso">
      {itens.map((item) => {
        const percentual = soma > 0 ? Math.round((item.total / soma) * 100) : 0;
        return (
          <div key={item.rotulo} className="lista-progresso-item">
            <div className="lista-progresso-cabecalho">
              <span>{item.rotulo}</span>
              <span className="numerico">{item.total} · {percentual}%</span>
            </div>
            <div className="lista-progresso-trilha">
              <div className="lista-progresso-barra" style={{ width: `${(item.total / maximo) * 100}%` }} />
            </div>
          </div>
        );
      })}
      {itens.length === 0 && <p className="aviso">Nenhum atendimento concluído hoje ainda.</p>}
    </div>
  );
}
