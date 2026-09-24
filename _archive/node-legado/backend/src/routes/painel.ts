import type { FastifyInstance } from "fastify";
import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

// Painel de TV é público (sem login, roda em Chromecast/TV) — por isso exibimos nome
// parcial (primeiro nome + inicial do último sobrenome) em vez do nome completo do
// cidadão. Política ainda não confirmada com a SASC (ver CLAUDE.md > Decisões em aberto);
// trocar essa função é a única mudança necessária quando a política for definida.
function nomeParcial(nomeCompleto: string): string {
  const partes = nomeCompleto.trim().split(/\s+/);
  if (partes.length === 1) return partes[0];
  const primeiro = partes[0];
  const ultimo = partes[partes.length - 1];
  return `${primeiro} ${ultimo.charAt(0).toUpperCase()}.`;
}

function formatarChamada(chamada: {
  id: string;
  chamadoEm: Date;
  agendamento: { protocolo: string; nomeCidadao: string };
  guiche: { nome: string };
}) {
  return {
    id: chamada.id,
    protocolo: chamada.agendamento.protocolo,
    nome: nomeParcial(chamada.agendamento.nomeCidadao),
    guiche: chamada.guiche.nome,
    chamadoEm: chamada.chamadoEm,
  };
}

// Rotas públicas pro painel de TV — sem authStub, de propósito (não tem login na TV).
export default async function painelRoutes(app: FastifyInstance) {
  app.get("/painel/:secretariaId/chamadas", async (request, reply) => {
    const { secretariaId } = request.params as { secretariaId: string };

    const secretaria = await prisma.secretaria.findUnique({ where: { id: secretariaId } });
    if (!secretaria) {
      return reply.code(404).send({ erro: "Secretaria não encontrada." });
    }

    const chamadas = await prisma.chamada.findMany({
      where: { agendamento: { secretariaId } },
      orderBy: { chamadoEm: "desc" },
      take: 10,
      include: { agendamento: true, guiche: true },
    });

    const formatadas = chamadas.map(formatarChamada);

    return {
      secretaria: { nome: secretaria.nome, sigla: secretaria.sigla },
      chamadaAtual: formatadas[0] ?? null,
      ultimasChamadas: formatadas,
    };
  });
}
