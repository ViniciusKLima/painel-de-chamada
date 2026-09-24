import { createContext, useCallback, useContext, useRef, useState, type ReactNode } from "react";
import { AlertCircle, CheckCircle2, X } from "lucide-react";

type TipoNotificacao = "sucesso" | "erro";

interface ItemNotificacao {
  id: number;
  texto: string;
  tipo: TipoNotificacao;
}

type Notificar = (texto: string, tipo?: TipoNotificacao) => void;

const NotificarContexto = createContext<Notificar | null>(null);

// Some sozinha depois de um tempo — antes cada tela tinha sua própria mensagem fixa que
// nunca desaparecia até a próxima ação, "ficando presa" na tela (reclamação do dono do
// produto, 21/09). Tempo generoso o bastante pra dar tempo de ler numa lista comprida.
const DURACAO_MS = 4500;

export function NotificacoesProvider({ children }: { children: ReactNode }) {
  const [lista, setLista] = useState<ItemNotificacao[]>([]);
  const proximoId = useRef(0);

  const remover = useCallback((id: number) => {
    setLista((atual) => atual.filter((n) => n.id !== id));
  }, []);

  const notificar = useCallback<Notificar>(
    (texto, tipo = "sucesso") => {
      const id = ++proximoId.current;
      setLista((atual) => [...atual, { id, texto, tipo }]);
      window.setTimeout(() => remover(id), DURACAO_MS);
    },
    [remover],
  );

  return (
    <NotificarContexto.Provider value={notificar}>
      {children}
      <div className="notificacoes-area">
        {lista.map((n) => (
          <div key={n.id} className={`notificacao notificacao-${n.tipo}`}>
            {n.tipo === "sucesso" ? <CheckCircle2 size={18} /> : <AlertCircle size={18} />}
            <span>{n.texto}</span>
            <button className="notificacao-fechar" onClick={() => remover(n.id)} title="Fechar">
              <X size={14} />
            </button>
          </div>
        ))}
      </div>
    </NotificarContexto.Provider>
  );
}

// Hook único usado por todas as telas — antes cada uma reinventava seu próprio estado de
// "mensagem" com estilo e comportamento levemente diferentes (21/09, organizado numa
// funcionalidade só: `notificar(texto)` pra sucesso, `notificar(texto, "erro")` pra erro).
export function useNotificar(): Notificar {
  const ctx = useContext(NotificarContexto);
  if (!ctx) throw new Error("useNotificar precisa ser usado dentro de NotificacoesProvider");
  return ctx;
}
