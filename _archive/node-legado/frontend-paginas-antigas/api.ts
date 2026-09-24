const API_URL = import.meta.env.VITE_API_URL as string;

export class ApiError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

function usuarioIdAtual(): string | null {
  return localStorage.getItem("usuarioId");
}

export function definirUsuarioId(id: string) {
  localStorage.setItem("usuarioId", id);
}

export function limparUsuarioId() {
  localStorage.removeItem("usuarioId");
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const usuarioId = usuarioIdAtual();
  const headers: Record<string, string> = { ...(options.headers as Record<string, string>) };
  if (usuarioId) headers["x-usuario-id"] = usuarioId;

  const resposta = await fetch(`${API_URL}${path}`, { ...options, headers });
  const corpo = await resposta.json().catch(() => null);

  if (!resposta.ok) {
    throw new ApiError(resposta.status, corpo?.erro ?? `Erro ${resposta.status}`);
  }

  return corpo as T;
}

export interface Usuario {
  id: string;
  secretariaId: string;
  papel: "admin" | "gestor" | "atendente";
  nome: string;
  secretaria: { id: string; nome: string; sigla: string };
}

export interface Guiche {
  id: string;
  nome: string;
  ativo: boolean;
}

export interface Agendamento {
  id: string;
  protocolo: string;
  nomeCidadao: string;
  horarioPrevisto: string;
  status: "aguardando" | "chamado" | "atendido" | "ausente";
  guiche: Guiche | null;
}

export interface ChamadaPainel {
  id: string;
  protocolo: string;
  nome: string;
  guiche: string;
  chamadoEm: string;
}

export interface RespostaUpload {
  loteId: string;
  status: string;
  totalLinhas: number;
  totalCriados: number;
  totalErros: number;
  erros: { linha: number; motivo: string }[];
}

export const api = {
  me: () => request<Usuario>("/me"),
  guiches: (secretariaId: string) => request<Guiche[]>(`/secretarias/${secretariaId}/guiches`),
  agendamentos: (secretariaId: string, status = "aguardando") =>
    request<Agendamento[]>(`/secretarias/${secretariaId}/agendamentos?status=${status}`),
  chamarProximo: (secretariaId: string, guicheId: string) =>
    request(`/secretarias/${secretariaId}/guiches/${guicheId}/chamar-proximo`, { method: "POST" }),
  rechamar: (agendamentoId: string) => request(`/agendamentos/${agendamentoId}/rechamar`, { method: "POST" }),
  marcarAusente: (agendamentoId: string) => request(`/agendamentos/${agendamentoId}/ausente`, { method: "POST" }),
  marcarAtendido: (agendamentoId: string) => request(`/agendamentos/${agendamentoId}/atendido`, { method: "POST" }),
  chamadasRecentes: (secretariaId: string) =>
    request<(ChamadaPainel & { agendamento: Agendamento })[]>(`/secretarias/${secretariaId}/chamadas/recentes`),
  uploadLote: async (secretariaId: string, arquivo: File) => {
    const form = new FormData();
    form.append("file", arquivo);
    return request<RespostaUpload>(`/secretarias/${secretariaId}/lotes`, { method: "POST", body: form });
  },
  painel: (secretariaId: string) =>
    fetch(`${API_URL}/painel/${secretariaId}/chamadas`).then((r) => {
      if (!r.ok) throw new ApiError(r.status, "Erro ao carregar painel");
      return r.json() as Promise<{
        secretaria: { nome: string; sigla: string };
        chamadaAtual: ChamadaPainel | null;
        ultimasChamadas: ChamadaPainel[];
      }>;
    }),
};
