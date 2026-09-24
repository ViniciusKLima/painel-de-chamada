import type { FastifyInstance } from "fastify";
import { PrismaClient, StatusLote } from "@prisma/client";
import { parse as parseCsv } from "csv-parse/sync";
import ExcelJS from "exceljs";

const prisma = new PrismaClient();

// Baseado no export real da plataforma Jornada (planilha de agendamentos). "Serviço" e
// "Local" são conceitos diferentes: Serviço é o tipo de atendimento (ex: CadÚnico) —
// hoje só guardado como texto solto; Local é a unidade física (ex: "CRAS - Vila Rica") e
// ainda não existe como coluna no export real, então é opcional aqui até existir.
const COLUNAS: Record<string, string[]> = {
  protocolo: ["protocolo"],
  data: ["data"],
  horario: ["horario", "hora"],
  nome: ["cidadao responsavel", "nome", "nome cidadao", "cidadao"],
  dependente: ["agendamento para dependente", "dependente"],
  cpf: ["cpf"],
  telefone: ["telefone"],
  servico: ["servico"],
  status: ["status"],
  canceladoEm: ["cancelado em"],
  motivoCancelamento: ["motivo do cancelamento", "motivo cancelamento"],
  local: ["local", "unidade"],
  guiche: ["guiche"],
};

const MESES: Record<string, number> = {
  janeiro: 1, fevereiro: 2, marco: 3, abril: 4, maio: 5, junho: 6,
  julho: 7, agosto: 8, setembro: 9, outubro: 10, novembro: 11, dezembro: 12,
};

function normalizarTexto(valor: string): string {
  return valor
    .normalize("NFD")
    .replace(/[̀-ͯ]/g, "")
    .replace(/[^a-zA-Z0-9\s]/g, " ")
    .replace(/\s+/g, " ")
    .trim()
    .toLowerCase();
}

function mapearColunas(cabecalhos: string[]): Record<string, number> {
  const normalizados = cabecalhos.map(normalizarTexto);
  const indices: Record<string, number> = {};

  for (const [campo, aliases] of Object.entries(COLUNAS)) {
    const idx = normalizados.findIndex((c) => aliases.includes(c));
    if (idx !== -1) indices[campo] = idx;
  }

  return indices;
}

// "15 de Setembro" (sem ano — assume o ano da data do upload) + "10:20"
function parseDataHorario(dataTexto: unknown, horarioTexto: unknown): Date | null {
  const data = String(dataTexto ?? "").trim();
  const horario = String(horarioTexto ?? "").trim();

  const matchData = data.match(/^(\d{1,2})\s+de\s+([a-zA-ZçÇ]+)$/i);
  const matchHorario = horario.match(/^(\d{1,2}):(\d{2})$/);
  if (!matchData || !matchHorario) return null;

  const mes = MESES[normalizarTexto(matchData[2])];
  if (!mes) return null;

  const dia = Number(matchData[1]);
  const ano = new Date().getFullYear();
  const resultado = new Date(ano, mes - 1, dia, Number(matchHorario[1]), Number(matchHorario[2]));
  return Number.isNaN(resultado.getTime()) ? null : resultado;
}

// "01 de Setembro de 2026, 06:08" — usado em "Confirmado em" / "Cancelado em"
function parseDataHoraCompleta(texto: unknown): Date | null {
  const valor = String(texto ?? "").trim();
  if (!valor) return null;

  const match = valor.match(/^(\d{1,2})\s+de\s+([a-zA-ZçÇ]+)\s+de\s+(\d{4}),?\s*(\d{1,2}):(\d{2})$/i);
  if (!match) return null;

  const mes = MESES[normalizarTexto(match[2])];
  if (!mes) return null;

  const resultado = new Date(
    Number(match[3]),
    mes - 1,
    Number(match[1]),
    Number(match[4]),
    Number(match[5])
  );
  return Number.isNaN(resultado.getTime()) ? null : resultado;
}

interface LinhaBruta {
  [campo: string]: unknown;
}

async function extrairLinhas(buffer: Buffer, filename: string): Promise<LinhaBruta[]> {
  const ehExcel = /\.xlsx?$/i.test(filename);

  if (ehExcel) {
    const workbook = new ExcelJS.Workbook();
    // exceljs traz sua própria versão de @types/node que colide com a do projeto — cast é seguro aqui.
    await workbook.xlsx.load(buffer as unknown as ArrayBuffer);
    const sheet = workbook.worksheets[0];
    if (!sheet) return [];

    const linhas: LinhaBruta[] = [];
    const cabecalhos: string[] = [];

    sheet.eachRow((row, rowNumber) => {
      const valores = (row.values as unknown[]).slice(1); // ExcelJS usa índice 1-based
      if (rowNumber === 1) {
        valores.forEach((v) => cabecalhos.push(String(v ?? "")));
        return;
      }
      const indices = mapearColunas(cabecalhos);
      const linha: LinhaBruta = {};
      for (const [campo, idx] of Object.entries(indices)) {
        linha[campo] = valores[idx];
      }
      linhas.push(linha);
    });

    return linhas;
  }

  const registros: Record<string, string>[] = parseCsv(buffer, {
    columns: (cabecalhos: string[]) => cabecalhos.map(normalizarTexto),
    skip_empty_lines: true,
    trim: true,
  });

  return registros.map((registro) => {
    const linha: LinhaBruta = {};
    for (const [campo, aliases] of Object.entries(COLUNAS)) {
      const chave = aliases.find((a) => a in registro);
      if (chave) linha[campo] = registro[chave];
    }
    return linha;
  });
}

interface ErroLinha {
  linha: number;
  motivo: string;
}

interface LinhaValidada {
  protocolo: string;
  nomeCidadao: string;
  cpf: string | null;
  telefone: string | null;
  servico: string | null;
  horarioPrevisto: Date;
  localId: string | null;
  guicheId: string | null;
  status: "aguardando" | "cancelado";
  canceladoEm: Date | null;
  motivoCancelamento: string | null;
}

export default async function lotesRoutes(app: FastifyInstance) {
  app.post("/secretarias/:secretariaId/lotes", async (request, reply) => {
    const { secretariaId } = request.params as { secretariaId: string };
    const usuario = request.usuarioAtual!;

    if (usuario.secretariaId !== secretariaId) {
      return reply.code(403).send({ erro: "Usuário não pertence a esta secretaria." });
    }
    if (usuario.papel !== "gestor" && usuario.papel !== "admin") {
      return reply.code(403).send({ erro: "Apenas gestor ou admin podem subir planilhas." });
    }

    const arquivo = await request.file();
    if (!arquivo) {
      return reply.code(400).send({ erro: "Nenhum arquivo enviado (campo multipart esperado)." });
    }

    const buffer = await arquivo.toBuffer();
    let linhasBrutas: LinhaBruta[];
    try {
      linhasBrutas = await extrairLinhas(buffer, arquivo.filename);
    } catch (err) {
      return reply.code(400).send({ erro: "Não foi possível ler o arquivo. Confira se é um .csv ou .xlsx válido." });
    }

    if (linhasBrutas.length === 0) {
      return reply.code(400).send({ erro: "Planilha vazia ou colunas não reconhecidas (esperado: protocolo, cidadão responsável, data, horário)." });
    }

    const [guiches, locais, existentes] = await Promise.all([
      prisma.guiche.findMany({ where: { secretariaId } }),
      prisma.local.findMany({ where: { secretariaId } }),
      prisma.agendamento.findMany({ where: { secretariaId }, select: { protocolo: true } }),
    ]);

    const guichesPorNome = new Map(guiches.map((g) => [normalizarTexto(g.nome), g.id]));
    const locaisPorNome = new Map(locais.map((l) => [normalizarTexto(l.nome), l.id]));
    const protocolosExistentes = new Set(existentes.map((a) => a.protocolo));
    const protocolosNoArquivo = new Set<string>();

    const erros: ErroLinha[] = [];
    const validas: LinhaValidada[] = [];

    linhasBrutas.forEach((linha, idx) => {
      const numeroLinha = idx + 2; // +1 pelo header, +1 pra base 1
      const protocolo = String(linha.protocolo ?? "").trim();
      const dependente = String(linha.dependente ?? "").trim();
      const responsavel = String(linha.nome ?? "").trim();
      const nomeCidadao = dependente || responsavel;
      const horarioPrevisto = parseDataHorario(linha.data, linha.horario);
      const statusTexto = normalizarTexto(String(linha.status ?? ""));
      const localTexto = String(linha.local ?? "").trim();
      const guicheTexto = String(linha.guiche ?? "").trim();

      if (!protocolo) {
        erros.push({ linha: numeroLinha, motivo: "Protocolo ausente." });
        return;
      }
      if (!nomeCidadao) {
        erros.push({ linha: numeroLinha, motivo: "Nome do cidadão (ou dependente) ausente." });
        return;
      }
      if (!horarioPrevisto) {
        erros.push({ linha: numeroLinha, motivo: "Data/horário ausente ou em formato não reconhecido." });
        return;
      }
      if (protocolosExistentes.has(protocolo) || protocolosNoArquivo.has(protocolo)) {
        erros.push({ linha: numeroLinha, motivo: `Protocolo '${protocolo}' duplicado.` });
        return;
      }

      let localId: string | null = null;
      if (localTexto) {
        localId = locaisPorNome.get(normalizarTexto(localTexto)) ?? null;
        if (!localId) {
          erros.push({ linha: numeroLinha, motivo: `Local '${localTexto}' não encontrado.` });
          return;
        }
      }

      let guicheId: string | null = null;
      if (guicheTexto) {
        guicheId = guichesPorNome.get(normalizarTexto(guicheTexto)) ?? null;
        if (!guicheId) {
          erros.push({ linha: numeroLinha, motivo: `Guichê '${guicheTexto}' não encontrado.` });
          return;
        }
      }

      const cancelado = statusTexto.startsWith("cancelado");

      protocolosNoArquivo.add(protocolo);
      validas.push({
        protocolo,
        nomeCidadao,
        cpf: String(linha.cpf ?? "").trim() || null,
        telefone: String(linha.telefone ?? "").trim() || null,
        servico: String(linha.servico ?? "").trim() || null,
        horarioPrevisto,
        localId,
        guicheId,
        status: cancelado ? "cancelado" : "aguardando",
        canceladoEm: cancelado ? parseDataHoraCompleta(linha.canceladoEm) : null,
        motivoCancelamento: cancelado ? String(linha.motivoCancelamento ?? "").trim() || null : null,
      });
    });

    const status: StatusLote = validas.length === 0 ? "erro" : "concluido";

    const resultado = await prisma.$transaction(async (tx) => {
      const lote = await tx.loteImportacao.create({
        data: {
          secretariaId,
          usuarioId: usuario.id,
          arquivoNome: arquivo.filename,
          status,
          totalLinhas: linhasBrutas.length,
          totalErros: erros.length,
        },
      });

      if (validas.length > 0) {
        await tx.agendamento.createMany({
          data: validas.map((v) => ({
            secretariaId,
            loteId: lote.id,
            protocolo: v.protocolo,
            nomeCidadao: v.nomeCidadao,
            cpf: v.cpf,
            telefone: v.telefone,
            servico: v.servico,
            horarioPrevisto: v.horarioPrevisto,
            localId: v.localId,
            guicheId: v.guicheId,
            status: v.status,
            canceladoEm: v.canceladoEm,
            motivoCancelamento: v.motivoCancelamento,
          })),
        });
      }

      return lote;
    });

    return reply.code(201).send({
      loteId: resultado.id,
      status: resultado.status,
      totalLinhas: linhasBrutas.length,
      totalCriados: validas.length,
      totalCancelados: validas.filter((v) => v.status === "cancelado").length,
      totalErros: erros.length,
      erros,
    });
  });
}
