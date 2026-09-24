import Fastify from "fastify";
import cors from "@fastify/cors";
import multipart from "@fastify/multipart";
import { PrismaClient } from "@prisma/client";
import authStub from "./plugins/authStub.js";
import lotesRoutes from "./routes/lotes.js";
import filaRoutes from "./routes/fila.js";
import painelRoutes from "./routes/painel.js";
import meRoutes from "./routes/me.js";

const prisma = new PrismaClient();
const app = Fastify({ logger: true });

await app.register(cors, { origin: true, allowedHeaders: ["Content-Type", "x-usuario-id"] });
await app.register(multipart);

// Rota simples pra confirmar que API + banco estão de pé
app.get("/health", async () => {
  const [{ ok }] = await prisma.$queryRaw<{ ok: number }[]>`SELECT 1 as ok`;
  return { status: "ok", db: ok === 1 };
});

await app.register(painelRoutes);

await app.register(async (protegidas) => {
  await protegidas.register(authStub);
  await protegidas.register(lotesRoutes);
  await protegidas.register(filaRoutes);
  await protegidas.register(meRoutes);
});

const port = Number(process.env.PORT ?? 3333);

app
  .listen({ port, host: "0.0.0.0" })
  .then(() => app.log.info(`Servidor rodando na porta ${port}`))
  .catch((err) => {
    app.log.error(err);
    process.exit(1);
  });
