-- CreateEnum
CREATE TYPE "Papel" AS ENUM ('admin', 'gestor', 'atendente');

-- CreateEnum
CREATE TYPE "StatusAgendamento" AS ENUM ('aguardando', 'chamado', 'atendido', 'ausente');

-- CreateEnum
CREATE TYPE "StatusLote" AS ENUM ('processando', 'concluido', 'erro');

-- CreateEnum
CREATE TYPE "TipoMapeamento" AS ENUM ('department', 'unit');

-- CreateTable
CREATE TABLE "prefeituras" (
    "id" TEXT NOT NULL,
    "nome" TEXT NOT NULL,
    "slug" TEXT NOT NULL,
    "issuer_jwt" TEXT,
    "criado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "prefeituras_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "secretarias" (
    "id" TEXT NOT NULL,
    "prefeitura_id" TEXT NOT NULL,
    "nome" TEXT NOT NULL,
    "sigla" TEXT NOT NULL,
    "criado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "secretarias_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "usuarios" (
    "id" TEXT NOT NULL,
    "secretaria_id" TEXT NOT NULL,
    "nome" TEXT NOT NULL,
    "email" TEXT NOT NULL,
    "papel" "Papel" NOT NULL,
    "external_sub" TEXT,
    "criado_via_sso" BOOLEAN NOT NULL DEFAULT false,
    "criado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "usuarios_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "guiches" (
    "id" TEXT NOT NULL,
    "secretaria_id" TEXT NOT NULL,
    "nome" TEXT NOT NULL,
    "ativo" BOOLEAN NOT NULL DEFAULT true,

    CONSTRAINT "guiches_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "lotes_importacao" (
    "id" TEXT NOT NULL,
    "secretaria_id" TEXT NOT NULL,
    "usuario_id" TEXT NOT NULL,
    "arquivo_nome" TEXT NOT NULL,
    "data_upload" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,
    "status" "StatusLote" NOT NULL DEFAULT 'processando',
    "total_linhas" INTEGER,
    "total_erros" INTEGER,

    CONSTRAINT "lotes_importacao_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "agendamentos" (
    "id" TEXT NOT NULL,
    "secretaria_id" TEXT NOT NULL,
    "lote_id" TEXT NOT NULL,
    "protocolo" TEXT NOT NULL,
    "nome_cidadao" TEXT NOT NULL,
    "horario_previsto" TIMESTAMP(3) NOT NULL,
    "guiche_id" TEXT,
    "status" "StatusAgendamento" NOT NULL DEFAULT 'aguardando',
    "criado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "agendamentos_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "chamadas" (
    "id" TEXT NOT NULL,
    "agendamento_id" TEXT NOT NULL,
    "guiche_id" TEXT NOT NULL,
    "usuario_id" TEXT NOT NULL,
    "chamado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "chamadas_pkey" PRIMARY KEY ("id")
);

-- CreateTable
CREATE TABLE "mapeamento_externo" (
    "id" TEXT NOT NULL,
    "tipo" "TipoMapeamento" NOT NULL,
    "external_id" TEXT NOT NULL,
    "secretaria_id" TEXT,
    "guiche_id" TEXT,

    CONSTRAINT "mapeamento_externo_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "prefeituras_slug_key" ON "prefeituras"("slug");

-- CreateIndex
CREATE UNIQUE INDEX "prefeituras_issuer_jwt_key" ON "prefeituras"("issuer_jwt");

-- CreateIndex
CREATE UNIQUE INDEX "secretarias_prefeitura_id_sigla_key" ON "secretarias"("prefeitura_id", "sigla");

-- CreateIndex
CREATE UNIQUE INDEX "usuarios_external_sub_key" ON "usuarios"("external_sub");

-- CreateIndex
CREATE UNIQUE INDEX "usuarios_secretaria_id_email_key" ON "usuarios"("secretaria_id", "email");

-- CreateIndex
CREATE UNIQUE INDEX "guiches_secretaria_id_nome_key" ON "guiches"("secretaria_id", "nome");

-- CreateIndex
CREATE INDEX "agendamentos_secretaria_id_status_idx" ON "agendamentos"("secretaria_id", "status");

-- CreateIndex
CREATE UNIQUE INDEX "agendamentos_secretaria_id_protocolo_key" ON "agendamentos"("secretaria_id", "protocolo");

-- CreateIndex
CREATE INDEX "chamadas_chamado_em_idx" ON "chamadas"("chamado_em");

-- CreateIndex
CREATE UNIQUE INDEX "mapeamento_externo_tipo_external_id_key" ON "mapeamento_externo"("tipo", "external_id");

-- AddForeignKey
ALTER TABLE "secretarias" ADD CONSTRAINT "secretarias_prefeitura_id_fkey" FOREIGN KEY ("prefeitura_id") REFERENCES "prefeituras"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "usuarios" ADD CONSTRAINT "usuarios_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "guiches" ADD CONSTRAINT "guiches_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "lotes_importacao" ADD CONSTRAINT "lotes_importacao_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "lotes_importacao" ADD CONSTRAINT "lotes_importacao_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "usuarios"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "agendamentos" ADD CONSTRAINT "agendamentos_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "agendamentos" ADD CONSTRAINT "agendamentos_lote_id_fkey" FOREIGN KEY ("lote_id") REFERENCES "lotes_importacao"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "agendamentos" ADD CONSTRAINT "agendamentos_guiche_id_fkey" FOREIGN KEY ("guiche_id") REFERENCES "guiches"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chamadas" ADD CONSTRAINT "chamadas_agendamento_id_fkey" FOREIGN KEY ("agendamento_id") REFERENCES "agendamentos"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chamadas" ADD CONSTRAINT "chamadas_guiche_id_fkey" FOREIGN KEY ("guiche_id") REFERENCES "guiches"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "chamadas" ADD CONSTRAINT "chamadas_usuario_id_fkey" FOREIGN KEY ("usuario_id") REFERENCES "usuarios"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "mapeamento_externo" ADD CONSTRAINT "mapeamento_externo_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE SET NULL ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "mapeamento_externo" ADD CONSTRAINT "mapeamento_externo_guiche_id_fkey" FOREIGN KEY ("guiche_id") REFERENCES "guiches"("id") ON DELETE SET NULL ON UPDATE CASCADE;
