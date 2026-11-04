export type StatusMark = { sign: '+' | '−' | '~'; label: string; tone: 'ok' | 'danger' | 'warn' }

// Estado nunca só por cor: cada situação tem sinal e rótulo.
export function statusMark(status: string): StatusMark {
  if (status === 'success') return { sign: '+', label: 'publicada', tone: 'ok' }
  if (status === 'error') return { sign: '−', label: 'erro', tone: 'danger' }
  if (status === 'skipped') return { sign: '~', label: 'ignorada', tone: 'warn' }
  return { sign: '~', label: status, tone: 'warn' }
}
