const num = (v: number, min: number, max: number) =>
  v.toLocaleString('pt-BR', { minimumFractionDigits: min, maximumFractionDigits: max })

const finite = (v: number) => (Number.isFinite(v) ? v : 0)

export function formatUsd(value: number): string {
  const v = finite(value)
  // Custos minúsculos (< US$ 0,01) mostram até 5 casas para não virarem 0,00.
  return 'US$ ' + (Math.abs(v) < 0.01 ? num(v, 2, 5) : num(v, 2, 2))
}

export function formatCount(value: number): string {
  const v = finite(value)
  if (Math.abs(v) >= 1_000_000) return num(v / 1_000_000, 0, 1) + ' mi'
  if (Math.abs(v) >= 10_000) return num(v / 1_000, 0, 1) + ' mil'
  return num(v, 0, 0)
}

export function formatRelative(iso: string, now: Date = new Date()): string {
  const t = new Date(iso).getTime()
  if (Number.isNaN(t)) return ''
  const sec = (now.getTime() - t) / 1000
  if (sec < 60) return 'agora' // inclui datas no futuro (relógio dessincronizado)
  const min = Math.floor(sec / 60)
  if (min < 60) return `há ${min} min`
  const h = Math.floor(min / 60)
  if (h < 24) return `há ${h} h`
  const d = Math.floor(h / 24)
  return d === 1 ? 'há 1 dia' : `há ${d} dias`
}
