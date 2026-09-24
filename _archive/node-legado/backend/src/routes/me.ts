import type { FastifyInstance } from "fastify";
import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

export default async function meRoutes(app: FastifyInstance) {
  app.get("/me", async (request) => {
    const usuario = request.usuarioAtual!;
    const secretaria = await prisma.secretaria.findUnique({ where: { id: usuario.secretariaId } });
    return { ...usuario, secretaria };
  });

  app.get("/secretarias/:secretariaId/guiches", async (request, reply) => {
    const { secretariaId } = request.params as { secretariaId: string };
    const usuario = request.usuarioAtual!;

    if (usuario.secretariaId !== secretariaId) {
      return reply.code(403).send({ erro: "Usuário não pertence a esta secretaria." });
    }

    return prisma.guiche.findMany({ where: { secretariaId, ativo: true }, orderBy: { nome: "asc" } });
  });
}
