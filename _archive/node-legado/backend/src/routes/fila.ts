import type { FastifyInstance } from "fastify";
import { PrismaClient, StatusAgendamento, Papel } from "@prisma/client";

const prisma = new PrismaClient();

const STATUS_VALIDOS: StatusAgendamento[] = ["aguardando", "chamado", "atendido", "ausente", "cancelado"];

// CPF e telefone são dado sensível (LGPD) — só gestor/admin enxergam. Atendente e o
// painel de TV público nunca recebem esses campos (ver decisão em CLAUDE.md).
function mascararCamposSensiveis<T extends { cpf?: string | null; telefone?: string | null }>(
  agendamento: T,
  papel: Papel
): T {
  if (papel === "gestor" || papel === "admin") return agendamento;
  return { ...agendamento, cpf: null, telefone: null };
}

export default async function filaRoutes(app: FastifyInstance) {
  // Lista a fila do dia de uma secretaria (por padrão, só quem está aguardando)
  app.get("/secretarias/:secretariaId/agendamentos", async (request, reply) => {
    const { secretariaId } = request.params as { secretariaId: string };
    const { status } = request.query as { status?: string };
    const usuario = request.usuarioAtual!;

    if (usuario.secretariaId !== secretariaId) {
      return reply.code(403).send({ erro: "Usuário não pertence a esta secretaria." });
    }

    if (status && !STATUS_VALIDOS.includes(status as StatusAgendamento)) {
      return reply.code(400).send({ erro: `Status inválido. Use um de: ${STATUS_VALIDOS.join(", ")}.` });
    }

    const agendamentos = await prisma.agendamento.findMany({
      where: { secretariaId, status: (status as StatusAgendamento) ?? "aguardando" },
      orderBy: { horarioPrevisto: "asc" },
      include: { guiche: true, local: true },
    });

    return agendamentos.map((a) => mascararCamposSensiveis(a, usuario.papel));
  });

  // Atendente de um guichê chama o próximo da fila (prioriza quem já está atribuído
  // àquele guichê, depois quem ainda não tem guichê definido).
  // Usa FOR UPDATE SKIP LOCKED pra dois atendentes chamando ao mesmo tempo nunca
  // pegarem o mesmo agendamento.
  app.post("/secretarias/:secretariaId/guiches/:guicheId/chamar-proximo", async (request, reply) => {
    const { secretariaId, guicheId } = request.params as { secretariaId: string; guicheId: string };
    const usuario = request.usuarioAtual!;

    if (usuario.secretariaId !== secretariaId) {
      return reply.code(403).send({ erro: "Usuário não pertence a esta secretaria." });
    }

    const guiche = await prisma.guiche.findFirst({ where: { id: guicheId, secretariaId } });
    if (!guiche) {
      return reply.code(404).send({ erro: "Guichê não encontrado nesta secretaria." });
    }
    if (!guiche.ativo) {
      return reply.code(400).send({ erro: "Guichê está inativo." });
    }

    const resultado = await prisma.$transaction(async (tx) => {
      const proximos = await tx.$queryRaw<{ id: string }[]>`
        SELECT id FROM agendamentos
        WHERE secretaria_id = ${secretariaId}
          AND status = 'aguardando'::"StatusAgendamento"
          AND (guiche_id = ${guicheId} OR guiche_id IS NULL)
        ORDER BY horario_previsto ASC
        LIMIT 1
        FOR UPDATE SKIP LOCKED
      `;

      const proximo = proximos[0];
      if (!proximo) return null;

      const agendamento = await tx.agendamento.update({
        where: { id: proximo.id },
        data: { status: "chamado", guicheId },
      });

      const chamada = await tx.chamada.create({
        data: { agendamentoId: agendamento.id, guicheId, usuarioId: usuario.id },
      });

      return { agendamento, chamada };
    });

    if (!resultado) {
      return reply.code(404).send({ erro: "Fila vazia — nenhum agendamento aguardando." });
    }

    return reply.code(201).send({
      ...resultado,
      agendamento: mascararCamposSensiveis(resultado.agendamento, usuario.papel),
    });
  });

  // Rechama (repete o aviso) um agendamento que já está com status 'chamado',
  // sem mudar o status — só registra uma nova Chamada pro histórico/painel.
  app.post("/agendamentos/:agendamentoId/rechamar", async (request, reply) => {
    const { agendamentoId } = request.params as { agendamentoId: string };
    const usuario = request.usuarioAtual!;

    const agendamento = await prisma.agendamento.findUnique({ where: { id: agendamentoId } });
    if (!agendamento || agendamento.secretariaId !== usuario.secretariaId) {
      return reply.code(404).send({ erro: "Agendamento não encontrado nesta secretaria." });
    }
    if (agendamento.status !== "chamado") {
      return reply.code(400).send({ erro: "Só é possível rechamar um agendamento com status 'chamado'." });
    }

    const ultimaChamada = await prisma.chamada.findFirst({
      where: { agendamentoId },
      orderBy: { chamadoEm: "desc" },
    });
    if (!ultimaChamada) {
      return reply.code(409).send({ erro: "Agendamento está 'chamado' mas não tem chamada registrada — estado inconsistente." });
    }

    const chamada = await prisma.chamada.create({
      data: { agendamentoId, guicheId: ultimaChamada.guicheId, usuarioId: usuario.id },
    });

    return reply.code(201).send(chamada);
  });

  // Marca ausência: só é uma transição válida a partir de 'chamado'.
  app.post("/agendamentos/:agendamentoId/ausente", async (request, reply) => {
    const { agendamentoId } = request.params as { agendamentoId: string };
    const usuario = request.usuarioAtual!;

    const { count } = await prisma.agendamento.updateMany({
      where: { id: agendamentoId, secretariaId: usuario.secretariaId, status: "chamado" },
      data: { status: "ausente" },
    });

    if (count === 0) {
      return reply.code(409).send({ erro: "Agendamento não existe nesta secretaria ou não está com status 'chamado'." });
    }

    return reply.code(200).send({ ok: true });
  });

  // Marca atendimento concluído: só é uma transição válida a partir de 'chamado'.
  app.post("/agendamentos/:agendamentoId/atendido", async (request, reply) => {
    const { agendamentoId } = request.params as { agendamentoId: string };
    const usuario = request.usuarioAtual!;

    const { count } = await prisma.agendamento.updateMany({
      where: { id: agendamentoId, secretariaId: usuario.secretariaId, status: "chamado" },
      data: { status: "atendido" },
    });

    if (count === 0) {
      return reply.code(409).send({ erro: "Agendamento não existe nesta secretaria ou não está com status 'chamado'." });
    }

    return reply.code(200).send({ ok: true });
  });

  // Últimas chamadas da secretaria — base pro painel de TV (chamada atual + histórico).
  app.get("/secretarias/:secretariaId/chamadas/recentes", async (request, reply) => {
    const { secretariaId } = request.params as { secretariaId: string };
    const usuario = request.usuarioAtual!;

    if (usuario.secretariaId !== secretariaId) {
      return reply.code(403).send({ erro: "Usuário não pertence a esta secretaria." });
    }

    const chamadas = await prisma.chamada.findMany({
      where: { agendamento: { secretariaId } },
      orderBy: { chamadoEm: "desc" },
      take: 10,
      include: { agendamento: true, guiche: true },
    });

    return chamadas.map((c) => ({ ...c, agendamento: mascararCamposSensiveis(c.agendamento, usuario.papel) }));
  });
}
