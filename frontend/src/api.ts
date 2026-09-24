// Deriva a URL da API do MESMO host que serviu a página, em vez de um IP fixo no .env —
// achado 20/09: se o frontend é acessado por um host (ex. "localhost") e a API por outro
// (ex. o IP da rede), o navegador trata como cross-site e bloqueia o cookie de sessão
// (SameSite=Lax), fazendo o login "funcionar" e imediatamente devolver pra tela de login
// na primeira checagem autenticada seguinte. Mantendo o mesmo hostname sempre, o front
// funciona igual acessando por localhost, 127.0.0.1 ou o IP da rede local, sem precisar
// lembrar de atualizar o .env se o IP mudar. VITE_API_URL continua disponível como
// override manual, se um dia backend e frontend precisarem ficar em hosts diferentes.
const API_URL = import.meta.env.VITE_API_URL || `${window.location.protocol}//${window.location.hostname}:3333`;

export class ApiError extends Error {
  status: number;
  codigo: string;

  constructor(status: number, codigo: string, mensagem: string) {
    super(mensagem);
    this.status = status;
    this.codigo = codigo;
  }
}

// Vigia de identidade de sessão (23/09) — corrige um bug real reproduzido de ponta a ponta:
// o cookie de sessão é por ORIGEM, não por aba. Cada tela busca `api.sessao()` só UMA VEZ ao
// montar e guarda o usuário em estado React pro resto da vida da página, sem nunca revalidar
// — então se OUTRA aba/janela do mesmo navegador faz login como outra pessoa (comum ao testar
// papéis diferentes, ou um computador compartilhado que troca de usuário sem logout explícito
// primeiro), o cookie da aba antiga é substituído nos bastidores, mas a tela antiga continua
// mostrando a UI de quem estava logado antes. Só quando uma ação de verdade é enviada é que a
// identidade REAL da sessão atual aparece — e a checagem de papel daquele endpoint falha pro
// papel de quem está logado agora, não pro papel que a tela ainda exibe. Reproduzido via curl
// (login A → ação ok → login B no mesmo cookie jar → mesma ação → 403 "papel_sem_acesso"),
// exatamente o sintoma relatado ("Apenas recepção ou gestor" depois de ficar ocioso, nas 4
// telas). Como cada tela já faz polling frequente (2-4s) mesmo sem interação do usuário, a
// abordagem reativa abaixo detecta a troca de identidade dentro de poucos segundos, sem
// precisar de um watchdog/intervalo novo.
let usuarioIdConhecido: string | null = null;
const CODIGOS_POSSIVEL_TROCA_DE_SESSAO = new Set([
  "papel_sem_acesso",
  "unidade_incorreta",
  "secretaria_incorreta",
  "nao_e_dono",
]);

function irParaLogin(motivo: "expirada" | "alterada") {
  usuarioIdConhecido = null;
  if (window.location.pathname === "/login") return; // evita loop de redirecionamento
  window.location.href = `/login?sessao=${motivo}`;
}

// Confirma se a sessão atual (o que o cookie REALMENTE identifica agora) ainda é a mesma
// pessoa que a tela acha que é. Usa `fetch` direto (não `request`) de propósito, pra não
// disparar este mesmo interceptor recursivamente.
async function confirmarIdentidadeAindaValida() {
  try {
    const resposta = await fetch(`${API_URL}/api/v1/sessao`, { credentials: "include" });
    if (!resposta.ok) {
      irParaLogin("expirada");
      return;
    }
    const usuario = await resposta.json().catch(() => null);
    if (!usuario?.id || usuario.id !== usuarioIdConhecido) {
      irParaLogin("alterada");
    }
    // Mesmo id: a sessão não mudou, então o 403 era uma restrição de permissão genuína —
    // deixa o erro original seguir seu fluxo normal (toast de erro na tela).
  } catch {
    // Falha de rede ao revalidar não é motivo pra deslogar — deixa o erro original passar.
  }
}

async function request<T>(path: string, options: RequestInit = {}): Promise<T> {
  const resposta = await fetch(`${API_URL}${path}`, {
    ...options,
    credentials: "include", // cookie de sessão HttpOnly — nunca lido em JS, só enviado
    headers: { "Content-Type": "application/json", ...(options.headers as Record<string, string>) },
  });
  const corpo = await resposta.json().catch(() => null);
  if (!resposta.ok) {
    if (resposta.status === 401 && path !== "/api/v1/sessao") {
      // 401 em qualquer chamada (não só na busca inicial de sessão) = sessão realmente
      // ausente/expirada/desativada — antes disso só a busca inicial de cada tela tratava
      // 401, deixando qualquer ação disparada pelo usuário depois disso sem nenhum
      // redirecionamento (só um toast de erro cru, a tela ficava travada precisando de
      // logout manual).
      irParaLogin("expirada");
    } else if (resposta.status === 403 && CODIGOS_POSSIVEL_TROCA_DE_SESSAO.has(corpo?.error_code) && path !== "/api/v1/sessao") {
      void confirmarIdentidadeAindaValida();
    }
    throw new ApiError(resposta.status, corpo?.error_code ?? "erro", corpo?.message ?? `Erro ${resposta.status}`);
  }
  if (path === "/api/v1/sessao" && options.method === "DELETE") {
    usuarioIdConhecido = null; // logout explícito
  } else if (path === "/api/v1/login" || path === "/api/v1/sessao") {
    usuarioIdConhecido = (corpo as { id?: string } | null)?.id ?? usuarioIdConhecido;
  }
  return corpo as T;
}

// Vínculos são DIFERENTES por papel (22/09, ver skill modelo-dados) — nem todo campo vem
// preenchido pra todo mundo: admin não tem nenhum (acesso master); gestor tem secretariaId
// (gerencia a secretaria inteira, todas as unidades/grades dela) mas não unidadeId;
// recepcionista tem unidadeId (trabalha o local físico inteiro); atendente tem unidadeId +
// gradeId (a fila específica que ele chama). corDestaque/corClara são a identidade visual
// da prefeitura do usuário (configurável pelo admin) — nulas pra admin.
// GradeResumo — só o essencial pra popular seletor/etiqueta, sem trazer o objeto Grade
// inteiro (SLA, painel etc.) onde só o nome importa. unidadeId/unidadeNome (23/09,
// auditoria de multi-grade) vêm por GRADE, não da unidade fixa do usuário — um atendente
// pode ter grades de unidades físicas DIFERENTES, e é isso que permite o frontend agrupar
// visualmente por local sem uma requisição extra (ver SalaDeEspera.tsx).
export interface GradeResumo {
  id: string;
  nome: string;
  unidadeId: string;
  unidadeNome: string;
}

export interface Usuario {
  id: string;
  nome: string;
  email: string;
  papel: "admin" | "gestor" | "atendente" | "recepcionista";
  unidadeId: string | null;
  unidadeNome: string | null;
  secretariaId: string | null;
  // Grades alocadas (23/09, "atendente ou recepção podem ser alocados em mais de uma
  // grade") — substitui o antigo gradeId/gradeNome único. Atendente escolhe em qual delas
  // trabalhar ao entrar (junto com o guichê, ver SalaDeEspera.tsx); recepcionista sem
  // nenhuma continua vendo/cadastrando em qualquer grade da própria unidade.
  grades: GradeResumo[];
  corDestaque: string | null;
  corClara: string | null;
  // Nome da prefeitura do usuário (22/09, agora que o admin gerencia várias) — nulo pra
  // admin, que não pertence a nenhuma específica. Substitui o antigo texto fixo do
  // cabeçalho/sidebar (ver skill identidade-visual).
  prefeituraNome: string | null;
}

export type TipoGuiche = "guiche" | "sala";

// ocupadoPorUsuarioId/ocupadoPorNome (22/09) — quem está atendendo nesse guichê agora, em
// tempo real (heartbeat, ver skill regras-negocio-fila). Nulo = livre. Um valor "velho"
// (sem heartbeat recente) já vem nulo do backend — o frontend nunca precisa calcular isso.
export interface Guiche {
  id: string;
  gradeId: string;
  nome: string;
  ativo: boolean;
  tipo: TipoGuiche;
  andar: string | null;
  capacidade: number;
  ocupadoPorUsuarioId: string | null;
  ocupadoPorNome: string | null;
  // excluida (23/09, auditoria de reativação) — soft-delete distinto de `ativo` (reversível
  // desde sempre). Só `ListarTodosGuiches` (tela de gestão) devolve excluídos; em qualquer
  // outra listagem isso vem sempre `false`.
  excluida: boolean;
}

// Unidade é só o local físico agora (22/09) — SLA/guichês/painel viraram atributo da Grade,
// não da unidade (uma unidade pode ter várias grades, uma por serviço).
export interface Unidade {
  id: string;
  secretariaId: string;
  nome: string;
}

// Grade é a fila de um serviço específico dentro de uma unidade (ex. "CadÚnico" e
// "Bolsa Família" na mesma unidade seriam duas grades distintas).
export interface Grade {
  id: string;
  unidadeId: string;
  servico: string;
  nome: string;
  slaChegadaMinutos: number;
  slaAtendimentoMinutos: number;
  duracaoChamadaPainelSegundos: number;
  duracaoChamadaPainelFilaSegundos: number;
  repeticoesChamada: number;
  intervaloRepeticaoSegundos: number;
  cooldownRechamadaMinutos: number;
  // 23/09 — recepcionista só pode cadastrar encaixe nesta grade se true; gestor/admin
  // sempre podem, independente deste campo (checado no backend).
  permiteEncaixeRecepcao: boolean;
  ativo: boolean;
  // excluida (23/09, auditoria de reativação) — soft-delete distinto de `ativo`. Só
  // `GET /secretarias/{id}/grades` (tela de gestão) devolve excluídas; em qualquer outra
  // listagem (recepção, painel, sessão) isso nunca aparece, porque elas são filtradas antes.
  excluida: boolean;
}

export interface ConfiguracaoGrade {
  grade: Grade;
  guiches: Guiche[];
}

// Linha da tela "Usuários internos" do gestor (22/09). `grades` (23/09) substitui o antigo
// gradeId/gradeNome único — cada uma vira uma "etiqueta" na tabela.
export interface UsuarioInterno {
  id: string;
  nome: string;
  email: string;
  papel: "gestor" | "atendente" | "recepcionista";
  ativo: boolean;
  grades: GradeResumo[];
  ultimoAcessoEm: string | null;
  pendente: boolean;
}

export interface Agendamento {
  id: string;
  unidadeId: string;
  gradeId: string;
  protocolo: string;
  nomeCidadao: string;
  cpf: string | null;
  telefone: string | null;
  servico: string | null;
  gradeHorario: string | null;
  tipo: "agendado" | "encaixe";
  horarioPrevisto: string | null;
  chegadaEm: string | null;
  prioridade: "no_horario" | "atrasado" | null;
  guicheId: string | null;
  status: "aguardando_chegada" | "sala_espera" | "chamado" | "atendido" | "ausente" | "cancelado";
  ausenteEm: string | null;
}

// Uma linha do bloco compartilhado "Chamando agora" — decisão 20/09 (ver skill
// regras-negocio-fila): TODOS os atendentes da grade veem a mesma lista; "souDono" indica se
// ESTE usuário pode agir (rechamar/marcar presente); "ehOrfao" indica se qualquer atendente
// ocioso pode assumir (o dono original já rechamou e seguiu em frente pra outra pessoa).
export interface ChamadoCompartilhado extends Agendamento {
  guicheNome: string;
  chamadoPorUsuarioId: string;
  chamadoPorNome: string;
  souDono: boolean;
  ehOrfao: boolean;
  totalChamadasDoDono: number;
  podeRechamarEm: string;
  prazoAusenciaEm: string;
}

// ehOrfao (23/09): distingue, dentro do status "chamado", quem ainda está normalmente
// sendo esperado pelo próprio atendente (false — ícone cinza "aguardando") de quem já foi
// abandonado pelo dono e precisa que outro atendente assuma (true — ícone âmbar de sempre).
export interface ChamadaPainel {
  id: string;
  protocolo: string;
  nome: string;
  guiche: string;
  status: string;
  ehOrfao: boolean;
  chamadoEm: string;
}

// Item do "modo aguardando" (23/09) — pessoa já chamada, ainda não atendida/ausente/órfã,
// mas fora da exibição de destaque normal (a janela de 30s/15s já passou). Só narração e
// countdown pertencem à `chamadaAtual`; isto aqui é puramente visual.
export interface ItemAguardando {
  protocolo: string;
  nome: string;
  guiche: string;
}

export interface RespostaPainel {
  // Chave "unidade" mantida por compatibilidade — na prática, desde a grade (22/09), é a
  // grade que está sendo exibida (nome já vem combinado "Unidade — Grade" quando não é a
  // única grade da unidade, ver handlers.GetPainel).
  unidade: { id: string; nome: string; repeticoesChamada: number; intervaloRepeticaoSegundos: number };
  // Cores de identidade da prefeitura (22/09, configuráveis pelo admin) — nulas só se a
  // cadeia unidade->secretaria->prefeitura falhar por algum motivo (não deveria acontecer
  // em uso normal).
  corDestaque: string | null;
  corClara: string | null;
  chamadaAtual: {
    protocolo: string;
    nome: string;
    guiche: string;
    // expiraEm/duracaoTotalSegundos (22/09) — só pra desenhar a barra de contagem
    // regressiva; a decisão de quando trocar de exibição continua só do backend.
    expiraEm: string;
    duracaoTotalSegundos: number;
  } | null;
  // Só vem preenchido quando chamadaAtual é nulo (ver handlers.GetPainel) — até 3 pessoas.
  aguardando: ItemAguardando[];
  ultimasChamadas: ChamadaPainel[];
}

export interface RespostaImportacao {
  loteId: string;
  status: string;
  totalLinhas: number;
  totalCriados: number;
  totalCancelados: number;
  totalErros: number;
  erros: { linha: number; motivo: string }[];
}

// Resultado da análise prévia da planilha (22/09) — primeira etapa do fluxo de importação
// em duas etapas: mostra o que aconteceria sem gravar nada, pro gestor confirmar.
export interface RespostaPreview {
  totalLinhas: number;
  totalValidas: number;
  totalCancelados: number;
  totalErros: number;
  erros: { linha: number; motivo: string }[];
  porDia: { data: string; total: number }[];
  unidadesNovas: string[];
  gradesNovas: { unidade: string; servico?: string }[];
}

export interface ResumoDashboard {
  totalDia: number;
  atendidos: number;
  ausentes: number;
  porAtendente: { usuarioId: string; nome: string; total: number }[] | null;
  serieDiaria: { data: string; atendidos: number; ausentes: number }[];
  porGrade: { gradeId: string; nome: string; atendidos: number; ausentes: number }[] | null;
}

// Admin (22/09) — hierarquia Admin -> Prefeitura -> Secretaria -> Usuários (ver skill
// modelo-dados). CorDestaque/corClara mapeiam pros tokens --cor-primaria/--cor-info-fundo
// do frontend (ver skill identidade-visual) — configuráveis por prefeitura.
export interface Prefeitura {
  id: string;
  nome: string;
  slug: string;
  corDestaque: string;
  corClara: string;
}

export interface Secretaria {
  id: string;
  prefeituraId: string;
  nome: string;
  sigla: string;
}

export interface DetalhePrefeitura {
  prefeitura: Prefeitura;
  secretarias: Secretaria[];
}

// Linha enriquecida da tela "Usuários internos" consolidada do admin (22/09) — gestor,
// atendente e recepcionista numa lista só, já com os nomes de prefeitura/secretaria/unidade/
// grade resolvidos, sem precisar de N+1 no frontend.
export interface UsuarioAdmin {
  id: string;
  nome: string;
  email: string;
  papel: "gestor" | "atendente" | "recepcionista";
  ativo: boolean;
  ultimoAcessoEm: string | null;
  pendente: boolean;
  prefeituraNome: string | null;
  secretariaId: string | null;
  secretariaNome: string | null;
  unidadeId: string | null;
  unidadeNome: string | null;
  grades: GradeResumo[];
}

export interface ResumoAdmin {
  totalPrefeituras: number;
  totalSecretarias: number;
  totalGestores: number;
  totalAtendentes: number;
  totalRecepcionistas: number;
}

export const api = {
  login: (email: string, senha: string) => request<Usuario>("/api/v1/login", { method: "POST", body: JSON.stringify({ email, senha }) }),
  sessao: () => request<Usuario>("/api/v1/sessao"),
  sair: () => request<void>("/api/v1/sessao", { method: "DELETE" }),

  // Recepção trabalha a UNIDADE inteira (todas as grades daquele local físico).
  recepcao: (unidadeId: string) => request<Agendamento[]>(`/api/v1/unidades/${unidadeId}/recepcao`),
  confirmarChegada: (agendamentoId: string) => request(`/api/v1/agendamentos/${agendamentoId}/confirmar-chegada`, { method: "POST" }),
  desfazerChegada: (agendamentoId: string) => request(`/api/v1/agendamentos/${agendamentoId}/desfazer-chegada`, { method: "POST" }),
  criarEncaixe: (unidadeId: string, nomeCidadao: string, gradeId: string, telefone?: string) =>
    request(`/api/v1/unidades/${unidadeId}/encaixe`, { method: "POST", body: JSON.stringify({ nomeCidadao, gradeId, telefone }) }),
  // Escopadas pela SECRETARIA inteira (22/09) — gestor deixou de ser vinculado a uma
  // unidade só, ver skill modelo-dados.
  grades: (secretariaId: string) => request<Grade[]>(`/api/v1/secretarias/${secretariaId}/grades`),
  // Só as grades de UMA unidade (recepção — o formulário de encaixe precisa saber pra qual
  // serviço é o walk-in dentro do local físico onde a pessoa chegou), distinto de
  // `grades()` acima (secretaria inteira, tela de gestão do gestor).
  gradesDaUnidade: (unidadeId: string) => request<Grade[]>(`/api/v1/unidades/${unidadeId}/grades`),
  usuarios: (secretariaId: string) => request<UsuarioInterno[]>(`/api/v1/secretarias/${secretariaId}/usuarios`),
  unidadesDaSecretaria: (secretariaId: string) => request<Unidade[]>(`/api/v1/secretarias/${secretariaId}/unidades`),
  atualizarUsuario: (usuarioId: string, dados: { nome: string; email: string; papel: string; senha?: string; gradeIds: string[] }) =>
    request<UsuarioInterno>(`/api/v1/usuarios/${usuarioId}`, { method: "PATCH", body: JSON.stringify(dados) }),
  excluirUsuario: (usuarioId: string) => request(`/api/v1/usuarios/${usuarioId}`, { method: "DELETE" }),
  reativarUsuario: (usuarioId: string) => request<UsuarioInterno>(`/api/v1/usuarios/${usuarioId}/reativar`, { method: "POST" }),

  // Consolidado de TODAS as grades do atendente logado (23/09, auditoria de multi-grade —
  // "não quero uma interface em que o atendente precise selecionar uma grade"): a lista de
  // grades vem sempre pronta em `usuario.grades` (ver sessao()); estes endpoints derivam a
  // lista de IDs no próprio backend a partir da sessão, sem o cliente precisar enviá-la nem
  // fazer N requisições (uma por grade) pra montar a visão consolidada.
  minhasGradesGuiches: () => request<Guiche[]>("/api/v1/atendente/grades/guiches"),
  minhasGradesSalaEspera: () => request<Agendamento[]>("/api/v1/atendente/grades/sala-espera"),
  minhasGradesChamados: () => request<ChamadoCompartilhado[]>("/api/v1/atendente/grades/chamados"),
  minhasGradesAtendidosHoje: () => request<Agendamento[]>("/api/v1/atendente/grades/atendidos-hoje"),
  minhasGradesAusentesHoje: () => request<Agendamento[]>("/api/v1/atendente/grades/ausentes-hoje"),

  // Atendente e fila trabalham por GRADE (um serviço específico dentro da unidade).
  guiches: (gradeId: string) => request<Guiche[]>(`/api/v1/grades/${gradeId}/guiches`),
  // Ocupação em tempo real (22/09): reivindicar um guichê ao escolhê-lo e reafirmar
  // periodicamente (heartbeat) enquanto o atendente continua nele; liberar ao sair/trocar.
  ocuparGuiche: (gradeId: string, guicheId: string) =>
    request<Guiche>(`/api/v1/grades/${gradeId}/guiches/${guicheId}/ocupar`, { method: "POST" }),
  liberarGuiche: (gradeId: string, guicheId: string) =>
    request(`/api/v1/grades/${gradeId}/guiches/${guicheId}/liberar`, { method: "POST" }),
  salaDeEspera: (gradeId: string) => request<Agendamento[]>(`/api/v1/grades/${gradeId}/sala-espera`),
  chamados: (gradeId: string) => request<ChamadoCompartilhado[]>(`/api/v1/grades/${gradeId}/chamados`),
  atendidosHoje: (gradeId: string) => request<Agendamento[]>(`/api/v1/grades/${gradeId}/atendidos-hoje`),
  ausentesHoje: (gradeId: string) => request<Agendamento[]>(`/api/v1/grades/${gradeId}/ausentes-hoje`),
  chamarProximo: (gradeId: string, guicheId: string) =>
    request(`/api/v1/grades/${gradeId}/guiches/${guicheId}/chamar-proximo`, { method: "POST" }),
  rechamar: (agendamentoId: string) => request(`/api/v1/agendamentos/${agendamentoId}/rechamar`, { method: "POST" }),
  marcarAtendido: (agendamentoId: string) => request(`/api/v1/agendamentos/${agendamentoId}/atendido`, { method: "POST" }),
  marcarAusencia: (agendamentoId: string) => request(`/api/v1/agendamentos/${agendamentoId}/ausencia`, { method: "POST" }),
  assumirChamada: (agendamentoId: string, guicheId: string) =>
    request(`/api/v1/agendamentos/${agendamentoId}/assumir`, { method: "POST", body: JSON.stringify({ guicheId }) }),

  dashboard: (secretariaId: string) => request<ResumoDashboard>(`/api/v1/secretarias/${secretariaId}/dashboard`),

  configuracaoGrade: (gradeId: string) => request<ConfiguracaoGrade>(`/api/v1/grades/${gradeId}/configuracao`),
  atualizarConfiguracaoGrade: (gradeId: string, config: Omit<Grade, "id" | "unidadeId" | "servico" | "excluida">) =>
    request<Grade>(`/api/v1/grades/${gradeId}/configuracao`, { method: "PATCH", body: JSON.stringify(config) }),
  criarGuiche: (gradeId: string, dados: { nome: string; tipo: TipoGuiche; andar: string | null; capacidade: number }) =>
    request<Guiche>(`/api/v1/grades/${gradeId}/guiches`, { method: "POST", body: JSON.stringify(dados) }),
  atualizarAtivoGuiche: (gradeId: string, guicheId: string, ativo: boolean) =>
    request(`/api/v1/grades/${gradeId}/guiches/${guicheId}`, { method: "PATCH", body: JSON.stringify({ ativo }) }),
  atualizarGuiche: (gradeId: string, guicheId: string, dados: Pick<Guiche, "nome" | "tipo" | "andar" | "capacidade" | "ativo">) =>
    request<Guiche>(`/api/v1/grades/${gradeId}/guiches/${guicheId}`, { method: "PUT", body: JSON.stringify(dados) }),
  excluirGuiche: (gradeId: string, guicheId: string) =>
    request(`/api/v1/grades/${gradeId}/guiches/${guicheId}`, { method: "DELETE" }),
  reativarGuiche: (gradeId: string, guicheId: string) =>
    request<Guiche>(`/api/v1/grades/${gradeId}/guiches/${guicheId}/reativar`, { method: "POST" }),
  excluirGrade: (gradeId: string) => request(`/api/v1/grades/${gradeId}`, { method: "DELETE" }),
  reativarGrade: (gradeId: string) => request<Grade>(`/api/v1/grades/${gradeId}/reativar`, { method: "POST" }),

  // Fluxo de importação em duas etapas (22/09): preview analisa sem gravar nada; importar
  // de fato só acontece quando o gestor confirma, reenviando o mesmo arquivo.
  previewPlanilha: async (secretariaId: string, arquivo: File) => {
    const form = new FormData();
    form.append("file", arquivo);
    const resposta = await fetch(`${API_URL}/api/v1/secretarias/${secretariaId}/lotes/preview`, {
      method: "POST",
      credentials: "include",
      body: form,
    });
    const corpo = await resposta.json().catch(() => null);
    if (!resposta.ok) throw new ApiError(resposta.status, corpo?.error_code ?? "erro", corpo?.message ?? "Erro ao analisar planilha");
    return corpo as RespostaPreview;
  },
  importarPlanilha: async (secretariaId: string, arquivo: File) => {
    const form = new FormData();
    form.append("file", arquivo);
    const resposta = await fetch(`${API_URL}/api/v1/secretarias/${secretariaId}/lotes`, {
      method: "POST",
      credentials: "include",
      body: form,
    });
    const corpo = await resposta.json().catch(() => null);
    if (!resposta.ok) throw new ApiError(resposta.status, corpo?.error_code ?? "erro", corpo?.message ?? "Erro ao importar");
    return corpo as RespostaImportacao;
  },

  // Painel de TV é público — sem credentials, sem cookie. Escopo por grade (22/09).
  painel: (gradeId: string) =>
    fetch(`${API_URL}/api/v1/painel/${gradeId}`).then((r) => {
      if (!r.ok) throw new ApiError(r.status, "erro", "Erro ao carregar painel");
      return r.json() as Promise<RespostaPainel>;
    }),
  // Central de painéis PÚBLICA de uma unidade (23/09) — pra acessar pela TV do local sem
  // precisar de login de gestor, ver skill painel-tv.
  paineisPublicoDaUnidade: (unidadeId: string) =>
    fetch(`${API_URL}/api/v1/unidades/${unidadeId}/paineis-publico`).then((r) => {
      if (!r.ok) throw new ApiError(r.status, "erro", "Erro ao carregar painéis");
      return r.json() as Promise<{ unidadeNome: string; grades: GradeResumo[] }>;
    }),

  // Admin — hierarquia Admin -> Prefeitura -> Secretaria -> Usuários (22/09).
  resumoAdmin: () => request<ResumoAdmin>("/api/v1/admin/resumo"),
  prefeituras: () => request<Prefeitura[]>("/api/v1/prefeituras"),
  criarPrefeitura: (nome: string) => request<Prefeitura>("/api/v1/prefeituras", { method: "POST", body: JSON.stringify({ nome }) }),
  prefeitura: (prefeituraId: string) => request<DetalhePrefeitura>(`/api/v1/prefeituras/${prefeituraId}`),
  atualizarPrefeitura: (prefeituraId: string, dados: { nome: string; corDestaque: string; corClara: string }) =>
    request<Prefeitura>(`/api/v1/prefeituras/${prefeituraId}`, { method: "PATCH", body: JSON.stringify(dados) }),
  criarSecretaria: (prefeituraId: string, nome: string, sigla: string) =>
    request<Secretaria>(`/api/v1/prefeituras/${prefeituraId}/secretarias`, { method: "POST", body: JSON.stringify({ nome, sigla }) }),
  criarGestor: (secretariaId: string, dados: { nome: string; email: string; senha: string }) =>
    request<UsuarioInterno>(`/api/v1/secretarias/${secretariaId}/gestores`, { method: "POST", body: JSON.stringify(dados) }),
  criarFuncionario: (unidadeId: string, dados: { nome: string; email: string; senha: string; papel: "atendente" | "recepcionista"; gradeIds: string[] }) =>
    request<UsuarioInterno>(`/api/v1/unidades/${unidadeId}/funcionarios`, { method: "POST", body: JSON.stringify(dados) }),
  usuariosInternosAdmin: () => request<UsuarioAdmin[]>("/api/v1/admin/usuarios"),

  // Abas da configuração de uma prefeitura, no admin (23/09) — dashboard/usuários/painéis
  // escopados a essa prefeitura só (substitui as antigas telas globais "Dashboard"/
  // "Usuários internos" do admin, que misturavam todas as prefeituras).
  dashboardPrefeitura: (prefeituraId: string) => request<ResumoDashboard>(`/api/v1/prefeituras/${prefeituraId}/dashboard`),
  usuariosPrefeitura: (prefeituraId: string) => request<UsuarioAdmin[]>(`/api/v1/prefeituras/${prefeituraId}/usuarios`),
  paineisPrefeitura: (prefeituraId: string) =>
    request<{ id: string; nome: string; unidadeId: string; unidadeNome: string }[]>(`/api/v1/prefeituras/${prefeituraId}/paineis`),
};
