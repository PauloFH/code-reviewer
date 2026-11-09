import { clsFromShifts, inpFromEvents, type InteractionEntry, type ShiftEntry } from '@/brand/vitals'

// Medidas de um carregamento de página, em ms (CLS é adimensional). Ausente = o navegador não mediu.
export interface WebVitals {
  path: string
  ttfb?: number
  fcp?: number
  lcp?: number
  cls?: number
  inp?: number
}

function observe(type: string, onEntries: (list: PerformanceEntry[]) => void, extra: object = {}) {
  try {
    new PerformanceObserver((l) => onEntries(l.getEntries())).observe({ type, buffered: true, ...extra })
  } catch {
    // tipo não suportado neste navegador (ex.: Safari sem layout-shift): a medida fica ausente
  }
}

// Coleta TTFB, FCP, LCP, CLS e INP do carregamento atual e envia uma vez, quando a aba some.
// getToken é lido só no envio: a página de login começa sem token e o envio é ignorado.
export function startVitals(getToken: () => string | null): void {
  if (typeof PerformanceObserver === 'undefined') return
  const path = location.pathname
  const shifts: ShiftEntry[] = []
  const events: InteractionEntry[] = []
  let lcp: number | undefined
  let fcp: number | undefined

  observe('paint', (es) => {
    for (const e of es) if (e.name === 'first-contentful-paint') fcp = e.startTime
  })
  observe('largest-contentful-paint', (es) => {
    lcp = es[es.length - 1]?.startTime
  })
  observe('layout-shift', (es) => shifts.push(...(es as unknown as ShiftEntry[])))
  observe('event', (es) => events.push(...(es as unknown as InteractionEntry[])), { durationThreshold: 40 })

  let sent = false
  function send() {
    const token = getToken()
    if (sent || !token) return
    sent = true
    const nav = performance.getEntriesByType('navigation')[0] as PerformanceNavigationTiming | undefined
    const inp = inpFromEvents(events)
    const body: WebVitals = {
      path,
      ttfb: nav ? Math.round(nav.responseStart) : undefined,
      fcp: fcp === undefined ? undefined : Math.round(fcp),
      lcp: lcp === undefined ? undefined : Math.round(lcp),
      // sem suporte a layout-shift (Firefox, Safari) o CLS fica ausente, não zero
      cls: PerformanceObserver.supportedEntryTypes?.includes('layout-shift')
        ? Number(clsFromShifts(shifts).toFixed(4))
        : undefined,
      inp: inp ? Math.round(inp) : undefined, // sem interação real não há INP
    }
    // keepalive sobrevive ao fechamento da aba e, ao contrário do sendBeacon, aceita o header de auth.
    fetch('/api/v1/vitals', {
      method: 'POST',
      keepalive: true,
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${token}` },
      body: JSON.stringify(body),
    }).catch(() => {})
  }

  addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') send()
  })
  addEventListener('pagehide', send)
}
