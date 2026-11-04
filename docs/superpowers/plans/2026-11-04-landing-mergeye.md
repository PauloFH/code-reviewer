# Landing page do Mergeye: plano de implementação

> **Para agentes:** SUB-SKILL OBRIGATÓRIA: use superpowers:subagent-driven-development (recomendada) ou superpowers:executing-plans para implementar este plano tarefa a tarefa. Os passos usam caixas (`- [ ]`) para acompanhamento.

**Objetivo:** servir em `/`, para quem não está logado, uma landing em que a logo se forma, abre-se nos fios e vira a linha amarela que liga as seções.

**Arquitetura:** a geometria e o tempo da abertura ficam em um módulo TypeScript puro e testado (`brand/abertura.ts`). Um componente fino (`LandingAbertura.vue`) aplica cada quadro no SVG com `requestAnimationFrame`, e uma view (`LandingView.vue`) traz o conteúdo. O backend não muda: o SPA embutido já responde em `/`.

**Stack:** Vue 3, TypeScript, Vite, `node --test`, CSS à mão sobre `tokens.css`, SVG.

**Spec:** `docs/superpowers/specs/2026-11-04-landing-mergeye-design.md`. Referência executável aprovada: `docs/superpowers/specs/assets/mergeye/landing-referencia.html`. Onde o protótipo divergir da spec sobre a primeira dobra, o topo e os textos, vale a spec.

## Restrições globais

- Trabalhar em `/home/paulorh/projetos/code-reviewer-landing`, branch `feat/landing-mergeye`. Não fazer push nem merge.
- Commit em uma linha, `tipo: mensagem`, sem escopo, corpo ou `Co-Authored-By`, inclusive em subagentes.
- Sem nova dependência. Cor literal só em `web/src/assets/tokens.css`; `.vue` usa `var(--x)`. Texto em PT-BR com acentuação.
- `requestAnimationFrame` só roda durante a abertura. SVG decorativo com `aria-hidden="true"`.
- Corpo da página sem rolagem horizontal em nenhuma largura.
- Continuidade: fim do fio amarelo em `x = largura do herói − 31,5` e `y = altura do herói`, com traço de 3 px, igual à linha das seções.
- Não copiar do protótipo o texto sobre `/review` nem sobre a Execução: use a seção de textos da Tarefa 2.

## Pontos de atenção da revisão

1. Link direto para uma seção (`/#modos`): não toca a abertura e rola até a seção.
2. `sessionStorage` bloqueado: nada lança erro; a abertura apenas toca de novo a cada carga.
3. Janela redimensionada ou celular girado, durante ou depois da abertura: fios e linha seguem alinhados e sem cobrir o texto.
4. Aba em segundo plano no meio da abertura, ou janela muito baixa ou de tamanho zero: ao voltar, a página está no estado final, sem repetir nem gerar `NaN` no SVG.
5. Sessão expirada em `/`: vai ao painel e o 401 leva a `/login`, sem página em branco nem laço de redirecionamento.

---

### Tarefa 1: Geometria e tempo da abertura

**Arquivos:**
- Criar: `web/src/brand/abertura.ts`
- Teste: `web/tests/abertura.test.ts`

**Interfaces:**
- Consome: nada.
- Produz, em `web/src/brand/abertura.ts`:

```ts
export const ABERTURA = { FIM: 4.2, ABRE: 2.7, DUR: 1 } as const
export const TRILHO_CENTRO = 31.5

export interface Medidas {
  largura: number
  altura: number
  topoLivre: number // y a partir do qual os fios podem ser desenhados
  logoTopo: { x: number; y: number; altura: number } // relativo ao herói
}

export interface Quadro {
  ramoV: number[]        // M + 3 cúbicas: 20 números
  ramoG: number[]        // idem
  tronco: number[]       // M + 1 cúbica + L: 10 números
  riscoRamos: number
  riscoTronco: number
  amarelo: number        // 0..1: quanto o tronco já é amarelo
  tracoRamos: number     // 0..1: quanto dos ramos já foi desenhado
  tracoTronco: number    // 0..1
  resto: { x: number; y: number; escala: number; opacidade: number }
  topo: number           // opacidade do topo
  texto: number          // opacidade do texto
  textoY: number         // deslocamento em px
  trilho: number         // 0..1: revelação da linha das seções
}

export function progresso(t: number, de: number, ate: number): number
export function quadroAbertura(t: number, m: Medidas): Quadro
export function caminhoRamo(v: number[]): string
export function caminhoTronco(v: number[]): string
export function deveTocarAbertura(o: { reduzMovimento: boolean; jaTocou: boolean; temHash: boolean }): boolean
```

- [ ] **Passo 1: Preparar a pasta e medir a base**

Run: `cd web && npm ci && npm test`
Expected: todos os testes passam antes de qualquer mudança.

- [ ] **Passo 2: Escrever os testes que falham**

Em `web/tests/abertura.test.ts`, com o mesmo cabeçalho de `web/tests/brand.test.ts`:

```ts
import { ABERTURA, TRILHO_CENTRO, quadroAbertura, caminhoRamo, deveTocarAbertura, type Medidas } from '../src/brand/abertura.ts'

const med = (largura: number, altura: number, topoLivre = 62): Medidas =>
  ({ largura, altura, topoLivre, logoTopo: { x: 22, y: 19, altura: 24 } })
const TAMANHOS = [[400, 700], [800, 360], [1000, 800], [1280, 800], [1600, 900], [2560, 1440]]

test('fim do fio amarelo cai no centro da linha, na borda do herói, com traço de 3', () => {
  for (const [w, h] of TAMANHOS) {
    const q = quadroAbertura(ABERTURA.FIM, med(w, h))
    assert.ok(Math.abs(q.tronco[8] - (w - TRILHO_CENTRO)) < 1e-6)
    assert.ok(Math.abs(q.tronco[9] - h) < 1e-6)
    assert.ok(Math.abs(q.riscoTronco - 3) < 1e-6)
    assert.ok(Math.abs(q.riscoRamos - 3) < 1e-6)
  }
})

test('ramos e tronco compartilham o ponto de junção em toda a abertura', () => {
  for (let t = 0; t <= ABERTURA.FIM; t += 0.1) {
    const q = quadroAbertura(t, med(1280, 800))
    for (const i of [0, 1]) {
      assert.ok(Math.abs(q.ramoV[18 + i] - q.tronco[i]) < 1e-6, `V em t=${t}`)
      assert.ok(Math.abs(q.ramoG[18 + i] - q.tronco[i]) < 1e-6, `G em t=${t}`)
    }
  }
})

test('fios mantêm a proporção aprovada em janelas altas', () => {
  for (const h of [700, 1400, 2200]) {
    const q = quadroAbertura(ABERTURA.FIM, med(1280, h))
    const larg = q.ramoV[18] - q.ramoV[0], alt = q.ramoG[1] - q.ramoV[1]
    assert.ok(Math.abs(alt / larg - 276 / 220) < 1e-6, `altura ${h}`)
  }
})

test('fios nunca sobem acima de topoLivre nem saem do herói', () => {
  for (const [w, h] of TAMANHOS) {
    const q = quadroAbertura(ABERTURA.FIM, med(w, h, w < 700 ? 300 : 62))
    assert.ok(q.ramoV[1] >= (w < 700 ? 300 : 62))
    assert.ok(q.ramoV[0] >= 0)
    assert.ok(q.ramoG[1] <= h)
  }
})

test('a logo se forma centrada no herói, antes de abrir', () => {
  const q = quadroAbertura(ABERTURA.ABRE, med(1280, 800))
  const centroX = q.resto.x + (866 / 2) * q.resto.escala
  const centroY = q.resto.y + (450 / 2) * q.resto.escala
  assert.ok(Math.abs(centroX - 640) < 1e-6)
  assert.ok(Math.abs(centroY - 400) < 1e-6)
})

test('estado inicial esconde a página e estado final a mostra inteira', () => {
  const a = quadroAbertura(0, med(1280, 800)), z = quadroAbertura(ABERTURA.FIM, med(1280, 800))
  assert.deepEqual([a.topo, a.texto, a.trilho, a.tracoRamos], [0, 0, 0, 0])
  assert.deepEqual([z.topo, z.texto, z.trilho, z.textoY, z.resto.opacidade, z.amarelo], [1, 1, 1, 0, 0, 1])
})

test('tempo além do fim equivale ao fim, como ao voltar de uma aba oculta', () => {
  assert.deepEqual(quadroAbertura(9999, med(1280, 800)), quadroAbertura(ABERTURA.FIM, med(1280, 800)))
})

test('tamanho zero não gera NaN', () => {
  const q = quadroAbertura(3.2, med(0, 0, 0))
  for (const n of [...q.ramoV, ...q.ramoG, ...q.tronco, q.riscoRamos, q.resto.escala]) assert.ok(Number.isFinite(n))
  assert.ok(!caminhoRamo(q.ramoV).includes('NaN'))
})

test('abertura só toca sem movimento reduzido, sem hash e na primeira vez da sessão', () => {
  const base = { reduzMovimento: false, jaTocou: false, temHash: false }
  assert.equal(deveTocarAbertura(base), true)
  assert.equal(deveTocarAbertura({ ...base, reduzMovimento: true }), false)
  assert.equal(deveTocarAbertura({ ...base, jaTocou: true }), false)
  assert.equal(deveTocarAbertura({ ...base, temHash: true }), false)
})
```

- [ ] **Passo 3: Confirmar que falham**

Run: `cd web && node --test tests/abertura.test.ts`
Expected: FAIL, porque `src/brand/abertura.ts` não existe.

- [ ] **Passo 4: Implementar `web/src/brand/abertura.ts`**

Porte de `landing-referencia.html`: `faixa` (aqui `progresso`), `suave`, `mix`, `divide`, `emTres`, `caminho`, e as constantes `LOGO_V`, `LOGO_G` e `LOGO_T`. Mantenha os tempos do `desenha` de lá: traço dos ramos `0,5–1,5`; traço do tronco `1,65–1,95`; `m = suave(progresso(t, ABRE, ABRE + DUR))`; `resto.opacidade = 1 − progresso(t, ABRE + 0,35, ABRE + 0,95)`; `topo` em `ABRE + 0,45..ABRE + 1`; `texto` em `ABRE + 0,55..ABRE + 1,15`, com `textoY = (1 − suave(texto)) * 14`; `trilho` em `ABRE + DUR − 0,05..ABRE + DUR + 0,45`; `amarelo = min(1, m * 2,5)`; riscos `mix(36s, 3, m)` e `mix(16s, 3, m)`.

O destino de `resto` é `logoTopo`, com `escala = logoTopo.altura / 450`.

A geometria muda em relação ao protótipo, para atender à spec. Sendo `W`, `H` e `T = topoLivre`:

```ts
const estreita = W < 700
const s = Math.min(W * (estreita ? 0.86 : 0.6) / 866, H * 0.62 / 450) // logo
const livre = Math.max(0, H - T)
const e = Math.min(W * (estreita ? 0.72 : 0.4) / 340, livre / 330)    // escala única dos fios
const oy = T + (livre - 330 * e) / 2
const railX = W - TRILHO_CENTRO, x0 = railX - 308.5 * e, mx = x0 + 220 * e
// fioV:  (x0, oy+24e) (x0+120e, oy+24e) (x0+130e, oy+140e) (mx, oy+160e), dividido com emTres
// fioG:  (x0, oy+300e) (x0+120e, oy+300e) (x0+130e, oy+180e) (mx, oy+160e), dividido com emTres
// tronco: (mx, oy+160e) (mx+60e, oy+160e) (railX, oy+180e) (railX, oy+230e) L (railX, H)
```

- [ ] **Passo 5: Confirmar que passam**

Run: `cd web && npm test`
Expected: todos passam.

- [ ] **Passo 6: Commit**

```bash
git add web/src/brand/abertura.ts web/tests/abertura.test.ts
git commit -m "feat: geometria da abertura da landing"
```

### Tarefa 2: Rota e conteúdo da landing

**Arquivos:**
- Criar: `web/src/views/LandingView.vue`
- Criar: `web/src/assets/landing/{painel-escuro,painel-claro,revisoes-escuro,revisoes-claro}.png`
- Criar: `docs/media/mergeye-revisoes-claro.png`
- Modificar: `web/src/router/index.ts`, `web/index.html`

**Interfaces:**
- Consome: `useTheme()`, `BrandLogo`, `AppIcon` (`sun`, `moon`), classes `.btn`, `.primary` e `.ghost`.
- Produz: rota `landing` em `/`. Na view, `.landing` como raiz, com as variáveis CSS `--abertura-topo`, `--abertura-texto`, `--abertura-texto-y` e `--abertura-trilho`, todas com fallback do estado final (`1`, `1`, `0px`, `1`). Também `.heroi`, `.topo`, `.texto`, um elemento que envolve `BrandLogo` no topo e `.trilho`, com a linha em `::before` a `right: 30px; width: 3px; background: var(--merge-mid)`.

- [ ] **Passo 1: Ajustar as rotas**

Em `web/src/router/index.ts`:
- Acrescentar, antes do pai `AppLayout`: `{ path: '/', name: 'landing', component: () => import('@/views/LandingView.vue'), meta: { public: true } }`.
- Remover o filho `{ path: '', redirect: '/dashboard' }`.
- Em `createRouter`, acrescentar `scrollBehavior: (to) => (to.hash ? { el: to.hash } : undefined)`.
- No início do `beforeEach`: `if (to.name === 'landing' && auth.isAuthenticated) return { name: 'dashboard' }`. O componente é preguiçoso e só carrega depois do guard, então a landing não pisca.

- [ ] **Passo 2: Gerar a captura clara de Revisões**

Com `node tests/mock-api.mjs` e `npm run dev` em `web/`, faça login, ative o tema claro e capture `/events` a 1440 × 900 em `docs/media/mergeye-revisoes-claro.png`. Copie para `web/src/assets/landing/`: `docs/media/mergeye-painel-escuro.png` → `painel-escuro.png`, `mergeye-painel-claro.png` → `painel-claro.png`, `mergeye-revisoes.png` → `revisoes-escuro.png`, `mergeye-revisoes-claro.png` → `revisoes-claro.png`.

- [ ] **Passo 3: Escrever `web/src/views/LandingView.vue`**

Estrutura: `.landing` > `.heroi` (com `<header class="topo">` absoluto e `<div class="texto">`) > `<main class="trilho">` com as seções > `<footer>`. Use `new URL(...)` para as imagens, como em `BrandLogo.vue`, com `width="1440" height="900" loading="lazy"` e a versão do tema atual. Âncoras são `<a href="#id">`; `Entrar` é `<router-link to="/login" class="btn primary">`; links externos usam `rel="noopener"`.

Layout, com a referência para o restante:
- `.heroi`: `position: relative; display: flex; align-items: center; min-height: max(100svh, 560px); padding: 84px 64px 44px 22px`.
- `.texto`: `position: relative; max-width: min(540px, 52%)`, com `opacity: var(--abertura-texto, 1)` e `transform: translateY(var(--abertura-texto-y, 0px))`.
- `.topo`: `position: absolute; inset: 0 0 auto 0`, com `opacity: var(--abertura-topo, 1)`.
- `.trilho`: `position: relative; padding: 0 64px 8px 22px`. A linha usa `transform: scaleY(var(--abertura-trilho, 1)); transform-origin: top`. Nó em `section::after`, como no protótipo. O commit percorre a linha só sem `prefers-reduced-motion`.
- Abaixo de 700 px: `.heroi` com `align-items: flex-start; min-height: max(100svh, 680px)`; `.texto` com `max-width: 100%`; os links âncora do topo somem, mas tema e `Entrar` ficam.
- `.landing` com `overflow-x: clip`.

Textos exatos:

| Parte | Texto |
|---|---|
| `h1` | Todo MR merece uma `<mark>`segunda leitura`</mark>`. |
| Linha do herói | Um serviço em Go que recebe o webhook, lê o diff com IA e comenta no próprio MR, uma única vez. |
| Botões do herói | Ver como funciona (`#como-funciona`) · Código no GitHub (`https://github.com/PauloFH/code-reviewer`) |
| Links do topo | como funciona · modos (`#modos`) · painel (`#painel`) · como é feito (`#como-e-feito`) |
| `#como-funciona` | **Como funciona.** Você marca o usuário configurado como reviewer ou assignee, e o resto acontece sozinho. |
| Passos | 1. Webhook: GitLab, GitHub ou Gitea avisam · 2. Diff: busca o MR pela API · 3. IA: Claude, OpenAI, Gemini ou DeepSeek · 4. Comentário: publicado no MR, uma vez só |
| `#modos` | **Quatro níveis de revisão.** As camadas se somam por perfil, definido por cliente ou por projeto. No GitLab, um comentário `/review deep` escolhe o nível daquele MR. |
| Níveis | `+` Diff: título, descrição e alterações · `+` Relacionado: issues, MRs e discussão (GitLab) · `+` Agêntica: a IA navega pelo repositório (GitLab) · `+` Execução: testes em Docker sem rede nem segredos, com o sandbox habilitado (GitLab) |
| `#painel` | **O painel.** Prompts, modelos e perfis por cliente e projeto, com custo e erros à vista. Não há demonstração aberta: você vê o painel por estas capturas. |
| `#como-e-feito` | **Como é feito.** A parte para quem quer ver como o sistema foi pensado. |
| Pontos | Não repete: reenvio de webhook não gera segundo review · Falha com aviso: retenta com backoff e explica o erro no MR · Segredos cifrados: credenciais de IA em AES-256-GCM · Fácil de estender: Clean Architecture e Strategy por plataforma |
| Rodapé | Feito por Paulo Holanda · Código no GitHub · Site pessoal (`https://www.pauloholanda.com.br`) |

- [ ] **Passo 4: Acrescentar as meta tags em `web/index.html`**

Acrescente `<meta name="description">` e `og:title`, `og:description`, `og:type` (`website`), `og:url` (`https://reviewer.pauloholanda.com.br/`) e `og:image` (`https://raw.githubusercontent.com/PauloFH/code-reviewer/main/.portfolio/capa.png`). Descrição: "Serviço em Go que revisa MRs e PRs com IA e comenta direto no GitLab, no GitHub e no Gitea." A URL do GitHub evita publicar a imagem pelo binário.

- [ ] **Passo 5: Testes e verificação no navegador**

Run: `cd web && npm test && npm run type-check && npm run lint`
Expected: tudo passa.

Com a API fictícia e o `npm run dev`, confira no navegador:
- Sem `jwt`, `/` mostra a landing, sem chamadas a `/api/`. As rotas do painel continuam indo para `/login`.
- Com login, `/` vai para `/dashboard`, sem a landing aparecer.
- Com `jwt` inválido na API real ou numa resposta 401 simulada, `/` termina em `/login`, sem laço.
- `/#modos` abre rolado até a seção.
- O logout continua levando a `/login`.
- A troca de tema muda as capturas.

- [ ] **Passo 6: Commit**

```bash
git add web/src/views/LandingView.vue web/src/assets/landing web/src/router/index.ts web/index.html docs/media/mergeye-revisoes-claro.png
git commit -m "feat: rota e conteúdo da landing"
```

### Tarefa 3: Abertura animada

**Arquivos:**
- Criar: `web/src/components/LandingAbertura.vue`
- Modificar: `web/src/views/LandingView.vue`, `web/src/assets/tokens.css`, `web/src/brand/storage.ts`, `web/tests/storage.test.ts`, `web/tests/sweep.test.ts`

**Interfaces:**
- Consome: `ABERTURA`, `quadroAbertura`, `caminhoRamo`, `caminhoTronco`, `progresso` e `deveTocarAbertura` (Tarefa 1); `.landing`, `.heroi`, `.topo`, `.texto`, o elemento da logo do topo e as variáveis `--abertura-*` (Tarefa 2).
- Produz:
  - `defineProps<{ raiz: HTMLElement; heroi: HTMLElement; topo: HTMLElement; logoTopo: HTMLElement; texto: HTMLElement }>()` em `LandingAbertura.vue`.
  - `readSessionFlag(key: string): boolean` e `writeSessionFlag(key: string): void` em `storage.ts`.

- [ ] **Passo 1: Escrever os testes que falham**

Em `web/tests/storage.test.ts`:

```ts
test('flag de sessão: grava, lê e não lança com sessionStorage bloqueado', () => {
  const m = new Map<string, string>()
  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, writable: true, value: {
    getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v),
  } })
  assert.equal(readSessionFlag('abertura'), false)
  writeSessionFlag('abertura')
  assert.equal(readSessionFlag('abertura'), true)

  Object.defineProperty(globalThis, 'sessionStorage', { configurable: true, get() { throw new Error('bloqueado') } })
  assert.equal(readSessionFlag('abertura'), false)
  assert.doesNotThrow(() => writeSessionFlag('abertura'))
})
```

No teste de detectores de `web/tests/sweep.test.ts`, acrescente:

```ts
assert.equal(achados(`components${sep}LandingAbertura.vue`, '<svg></svg>').length, 0)
```

- [ ] **Passo 2: Confirmar que falham**

Run: `cd web && node --test tests/storage.test.ts tests/sweep.test.ts`
Expected: FAIL, porque `readSessionFlag` não existe e `LandingAbertura.vue` ainda dispara o detector.

- [ ] **Passo 3: Implementar as flags e a exceção da varredura**

Em `storage.ts`, siga o padrão de `readStorage` com `sessionStorage`. Em `sweep.test.ts`, a condição do `<svg` passa a aceitar também arquivos terminados em `${sep}LandingAbertura.vue`, com o comentário: `// símbolo animado: AppIcon só desenha ícones estáticos de um traço`.

- [ ] **Passo 4: Acrescentar os tokens em `tokens.css`**

Decorativos: nenhum fica sob texto, então o teste de contraste não muda.

| Token | Escuro | Claro |
|---|---|---|
| `--logo-traco` | `#f4f2e8` | `#16140e` |
| `--logo-v-1` / `--logo-v-2` | `#ff4d1f` / `#ff8a1f` | `#dc2a18` / `#e8730f` |
| `--logo-g-1` / `--logo-g-2` | `#1fd97c` / `#6fe03a` | `#07924f` / `#4aa91f` |
| `--logo-no-v` / `--logo-no-g` | `#e6410f` / `#14913c` | `#dc2a18` / `#07924f` |
| `--logo-bloco` | `#ffbf1f` | `#e0a100` |
| `--logo-check` | `#2fd94a` | `#07924f` |
| `--logo-brilho` | `rgba(255, 190, 30, 0.45)` | `rgba(224, 161, 0, 0.30)` |

Os gradientes dos ramos terminam em `var(--merge-mid)`: o fio amarelo, a linha e a junção têm a mesma cor nos dois temas.

- [ ] **Passo 5: Escrever `LandingAbertura.vue`**

Template: porte o `<svg>` da referência (`#ramoV`, `#ramoG`, `#tronco`, `#resto` e `[data-de]`), trocando cor literal por classes com `var(--x)` e `id`s de gradiente únicos. O traço do tronco é `stroke: color-mix(in srgb, var(--merge-mid) calc(var(--amarelo) * 100%), var(--logo-traco))`. O fundo dos nós escuros é `var(--bg)`.

Script:
- `desenha(t)`: mede `largura` e `altura` por `heroi.clientWidth/Height`. `topoLivre` é `topo.offsetHeight` a partir de 700 px e `texto.offsetTop + texto.offsetHeight + 16` abaixo disso. `logoTopo` vem de `getBoundingClientRect` relativo ao herói. Aplica o `quadroAbertura` no SVG e define `--abertura-topo`, `--abertura-texto`, `--abertura-texto-y` e `--abertura-trilho` em `raiz`. Aplica `progresso(t, de, ate)` em cada `[data-de]`.
- `onMounted`: chama `deveTocarAbertura({ reduzMovimento: matchMedia('(prefers-reduced-motion: reduce)').matches, jaTocou: readSessionFlag('abertura-tocou'), temHash: !!location.hash })`. Se for tocar, grava a flag, desenha `0` no mesmo tique e inicia o `requestAnimationFrame`. Se não, desenha `FIM`.
- Pular: enquanto toca, `pointerdown`, `wheel`, `touchstart` e `keydown` em `window`, com `{ passive: true }`, vão direto a `FIM`. Remova os ouvintes ao terminar.
- `ResizeObserver` no herói redesenha o `t` atual.
- `onBeforeUnmount`: cancela o quadro, remove os ouvintes, desconecta o observer e limpa as variáveis `--abertura-*`.

Em `LandingView.vue`: pegue as referências com `ref`, ligue um `montado` em `onMounted` e renderize `<LandingAbertura v-if="montado" ...>` como primeiro filho de `.heroi`.

- [ ] **Passo 6: Testes e verificação da abertura**

Run: `cd web && npm test && npm run type-check && npm run lint`
Expected: tudo passa.

No navegador, confira:
- O primeiro carregamento toca a abertura, sem o texto piscar antes, e o segundo abre direto. Limpando o `sessionStorage`, toca de novo.
- Clique, rolagem e tecla pulam para o estado final.
- Com `prefers-reduced-motion`, a página abre pronta e dois quadros com 2 s de intervalo são idênticos.
- Em 400, 1000, 1280, 1600 e 2560 px, o fim de `#tronco` difere 0 px do centro da linha, medido com `getScreenCTM`, mesmo redimensionando durante e depois da abertura.

- [ ] **Passo 7: Commit**

```bash
git add web/src/components/LandingAbertura.vue web/src/views/LandingView.vue web/src/assets/tokens.css web/src/brand/storage.ts web/tests/storage.test.ts web/tests/sweep.test.ts
git commit -m "feat: abertura animada da landing"
```

### Tarefa 4: Revisão visual, build e documentação

**Arquivos:**
- Modificar: `web/CLAUDE.md`, `web/src/views/CLAUDE.md`, `web/src/components/CLAUDE.md`, `README.md`, `internal/transport/http/frontend/dist/index.html`

**Interfaces:**
- Consome: a página pronta das Tarefas 1 a 3.
- Produz: branch pronta para revisão.

- [ ] **Passo 1: Conferir o estado final**

Compare com a spec, nos dois temas e em 400 × 800, 800 × 360, 1280 × 800 e 2560 × 1440:
- Nenhum elemento provoca rolagem horizontal: `document.documentElement.scrollWidth <= innerWidth`.
- Os fios não tocam o texto do herói: a borda esquerda de `#ramoV` fica à direita de `.texto`, ou, abaixo de 700 px, abaixo dele.
- `.landing` não usa os itens proibidos da spec da identidade.
- No tema claro, os traços do símbolo aparecem em tinta escura.

Corrija o que falhar em `LandingView.vue` ou `abertura.ts`, mantendo os testes passando.

- [ ] **Passo 2: Atualizar a documentação**

- `web/CLAUDE.md`: na tabela de rotas, `/` passa a ser `landing`, pública, `LandingView.vue`, com redirecionamento a `/dashboard` para quem tem sessão. Em "Ícones", registre a exceção de `LandingAbertura.vue`.
- `web/src/views/CLAUDE.md` e `web/src/components/CLAUDE.md`: acrescente uma linha para cada arquivo novo, se listarem arquivos.
- `README.md`: em uma frase, diga que `/` mostra a landing para quem não está logado.

- [ ] **Passo 3: Rodar a suíte completa e gerar o build**

Run: `cd web && npm test && npm run lint && npm run build && cd .. && GOPRIVATE='git.pauloholanda.com.br/*' go test ./...`
Expected: tudo passa, e `dist/index.html` é atualizado com as meta tags e os novos hashes. Se o `go test` falhar por falta de acesso ao módulo privado, registre isso no relatório: não é falha da landing.

- [ ] **Passo 4: Conferir o binário**

Run: `go run ./cmd/server`, com `.env` e `clients.json` locais, e abra `http://localhost:8080/`.
Expected: a landing abre, as imagens carregam e `/login` continua funcionando. Se o módulo privado impedir a compilação, faça a mesma conferência com `npm run preview` em `web/` e registre a limitação.

- [ ] **Passo 5: Commit**

```bash
git add web/CLAUDE.md web/src/views/CLAUDE.md web/src/components/CLAUDE.md README.md internal/transport/http/frontend/dist/index.html
git commit -m "docs: registra a landing e atualiza o build"
```
