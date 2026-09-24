import { PrismaClient } from "@prisma/client";

const prisma = new PrismaClient();

// Dados fixos pra dev enquanto o SSO/JWT não está disponível (ver CLAUDE.md > Decisões em aberto)
async function main() {
  const prefeitura = await prisma.prefeitura.upsert({
    where: { slug: "jaboatao" },
    update: {},
    create: {
      nome: "Jaboatão dos Guararapes",
      slug: "jaboatao",
      issuerJwt: "https://jaboataomaisfacil.jaboatao.pe.gov.br",
    },
  });

  const secretaria = await prisma.secretaria.upsert({
    where: { prefeituraId_sigla: { prefeituraId: prefeitura.id, sigla: "SASC" } },
    update: {},
    create: {
      prefeituraId: prefeitura.id,
      nome: "Secretaria de Assistência Social e Cidadania",
      sigla: "SASC",
    },
  });

  const localVilaRica = await prisma.local.upsert({
    where: { secretariaId_nome: { secretariaId: secretaria.id, nome: "CRAS - Vila Rica" } },
    update: {},
    create: { secretariaId: secretaria.id, nome: "CRAS - Vila Rica" },
  });

  const localCavaleiro = await prisma.local.upsert({
    where: { secretariaId_nome: { secretariaId: secretaria.id, nome: "CRAS - Cavaleiro" } },
    update: {},
    create: { secretariaId: secretaria.id, nome: "CRAS - Cavaleiro" },
  });

  const guiche1 = await prisma.guiche.upsert({
    where: { secretariaId_nome: { secretariaId: secretaria.id, nome: "Guichê 1" } },
    update: {},
    create: { secretariaId: secretaria.id, localId: localVilaRica.id, nome: "Guichê 1" },
  });

  await prisma.guiche.upsert({
    where: { secretariaId_nome: { secretariaId: secretaria.id, nome: "Guichê 2" } },
    update: {},
    create: { secretariaId: secretaria.id, localId: localCavaleiro.id, nome: "Guichê 2" },
  });

  const gestor = await prisma.usuario.upsert({
    where: { secretariaId_email: { secretariaId: secretaria.id, email: "gestor.sasc@dev.local" } },
    update: {},
    create: {
      secretariaId: secretaria.id,
      nome: "Gestor SASC (dev)",
      email: "gestor.sasc@dev.local",
      papel: "gestor",
    },
  });

  const atendente = await prisma.usuario.upsert({
    where: { secretariaId_email: { secretariaId: secretaria.id, email: "atendente.sasc@dev.local" } },
    update: {},
    create: {
      secretariaId: secretaria.id,
      nome: "Atendente SASC (dev)",
      email: "atendente.sasc@dev.local",
      papel: "atendente",
    },
  });

  console.log("Seed concluído:");
  console.log({ prefeitura: prefeitura.slug, secretaria: secretaria.sigla, guiche1: guiche1.nome });
  console.log("Usuário gestor (usar como x-usuario-id):", gestor.id);
  console.log("Usuário atendente (usar como x-usuario-id):", atendente.id);
}

main()
  .catch((err) => {
    console.error(err);
    process.exit(1);
  })
  .finally(async () => {
    await prisma.$disconnect();
  });
