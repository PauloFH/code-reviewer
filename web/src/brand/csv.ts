// Campo de texto vindo de fora (título de MR, autor) que começa com = + - @ vira fórmula na planilha: prefixa com '.
export function csvField(v: string | number): string {
  let s = String(v)
  if (typeof v === 'string' && /^[=+\-@\t\r]/.test(s)) s = "'" + s
  return /[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s
}

export function toCsv(rows: (string | number)[][]): string {
  return rows.map((r) => r.map(csvField).join(',')).join('\r\n')
}
