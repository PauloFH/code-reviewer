// Cálculo dos Web Vitals a partir das entradas cruas do PerformanceObserver (sem dependência externa).

export interface ShiftEntry {
  value: number
  startTime: number
  hadRecentInput: boolean
}

export interface InteractionEntry {
  interactionId: number
  duration: number
}

// CLS: a maior janela de sessão (deslocamentos a menos de 1 s um do outro, janela de até 5 s);
// deslocamento logo após input do usuário não conta.
export function clsFromShifts(entries: ShiftEntry[]): number {
  let max = 0
  let session = 0
  let first = 0
  let last = 0
  for (const e of entries) {
    if (e.hadRecentInput) continue
    if (session && e.startTime - last < 1000 && e.startTime - first < 5000) {
      session += e.value
    } else {
      session = e.value
      first = e.startTime
    }
    last = e.startTime
    max = Math.max(max, session)
  }
  return max
}

// shortcut: INP é a pior interação; o oficial usa o p98 acima de 50 interações, irrelevante num painel admin.
export function inpFromEvents(entries: InteractionEntry[]): number {
  let max = 0
  for (const e of entries) if (e.interactionId) max = Math.max(max, e.duration)
  return max
}
