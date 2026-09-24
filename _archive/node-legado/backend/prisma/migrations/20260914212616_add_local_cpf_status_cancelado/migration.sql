/*
  Warnings:

  - Added the required column `local_id` to the `guiches` table without a default value. This is not possible if the table is not empty.

*/
-- AlterEnum
ALTER TYPE "StatusAgendamento" ADD VALUE 'cancelado';

-- AlterTable
ALTER TABLE "agendamentos" ADD COLUMN     "cancelado_em" TIMESTAMP(3),
ADD COLUMN     "cpf" TEXT,
ADD COLUMN     "local_id" TEXT,
ADD COLUMN     "motivo_cancelamento" TEXT,
ADD COLUMN     "servico" TEXT,
ADD COLUMN     "telefone" TEXT;

-- AlterTable
ALTER TABLE "guiches" ADD COLUMN     "local_id" TEXT NOT NULL;

-- CreateTable
CREATE TABLE "locais" (
    "id" TEXT NOT NULL,
    "secretaria_id" TEXT NOT NULL,
    "nome" TEXT NOT NULL,
    "ativo" BOOLEAN NOT NULL DEFAULT true,
    "criado_em" TIMESTAMP(3) NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT "locais_pkey" PRIMARY KEY ("id")
);

-- CreateIndex
CREATE UNIQUE INDEX "locais_secretaria_id_nome_key" ON "locais"("secretaria_id", "nome");

-- AddForeignKey
ALTER TABLE "locais" ADD CONSTRAINT "locais_secretaria_id_fkey" FOREIGN KEY ("secretaria_id") REFERENCES "secretarias"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "guiches" ADD CONSTRAINT "guiches_local_id_fkey" FOREIGN KEY ("local_id") REFERENCES "locais"("id") ON DELETE RESTRICT ON UPDATE CASCADE;

-- AddForeignKey
ALTER TABLE "agendamentos" ADD CONSTRAINT "agendamentos_local_id_fkey" FOREIGN KEY ("local_id") REFERENCES "locais"("id") ON DELETE SET NULL ON UPDATE CASCADE;
