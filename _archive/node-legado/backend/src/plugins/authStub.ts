import fp from "fastify-plugin";
import type { FastifyInstance, FastifyRequest, FastifyReply } from "fastify";
import { PrismaClient, Papel } from "@prisma/client";

const prisma = new PrismaClient();

// STUB TEMPORÁRIO enquanto o middleware de JWT real (ver CLAUDE.md > roadmap item 2)
// não pode ser implementado — falta alinhar o `aud` dedicado com a equipe da plataforma
// Jornada. NÃO usar em produção: qualquer chamador pode se passar por qualquer usuário
// só informando o header `x-usuario-id`. Trocar por validação JWKS assim que possível.
export interface UsuarioAutenticado {
  id: string;
  secretariaId: string;
  papel: Papel;
  nome: string;
}

declare module "fastify" {
  interface FastifyRequest {
    usuarioAtual?: UsuarioAutenticado;
  }
}

async function authStubPlugin(app: FastifyInstance) {
  app.decorateRequest("usuarioAtual", undefined);

  app.addHook("preHandler", async (request: FastifyRequest, reply: FastifyReply) => {
    const usuarioId = request.headers["x-usuario-id"];

    if (!usuarioId || Array.isArray(usuarioId)) {
      return reply.code(401).send({
        erro: "Header x-usuario-id ausente ou inválido (auth stub de dev, ver CLAUDE.md).",
      });
    }

    const usuario = await prisma.usuario.findUnique({ where: { id: usuarioId } });

    if (!usuario) {
      return reply.code(401).send({ erro: "Usuário não encontrado para o x-usuario-id informado." });
    }

    request.usuarioAtual = {
      id: usuario.id,
      secretariaId: usuario.secretariaId,
      papel: usuario.papel,
      nome: usuario.nome,
    };
  });
}

export default fp(authStubPlugin);
