<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useTheme } from '@/composables/useTheme'
import BrandLogo from '@/components/BrandLogo.vue'
import AppIcon from '@/components/AppIcon.vue'
import LandingAbertura from '@/components/LandingAbertura.vue'

const { theme, toggleTheme } = useTheme()

const raiz = ref<HTMLElement>()
const heroi = ref<HTMLElement>()
const topo = ref<HTMLElement>()
const logoTopo = ref<HTMLElement>()
const texto = ref<HTMLElement>()
const montado = ref(false)
onMounted(() => (montado.value = true))

// URLs literais para o Vite resolver em build.
const capturas = {
  'painel-dark': new URL('../assets/landing/painel-escuro.png', import.meta.url).href,
  'painel-light': new URL('../assets/landing/painel-claro.png', import.meta.url).href,
  'revisoes-dark': new URL('../assets/landing/revisoes-escuro.png', import.meta.url).href,
  'revisoes-light': new URL('../assets/landing/revisoes-claro.png', import.meta.url).href,
}
const painel = computed(() => capturas[`painel-${theme.value}`])
const revisoes = computed(() => capturas[`revisoes-${theme.value}`])

const GITHUB = 'https://github.com/PauloFH/code-reviewer'
const SITE = 'https://www.pauloholanda.com.br'

const passos = [
  { titulo: '1. Webhook', texto: 'GitLab, GitHub ou Gitea avisam' },
  { titulo: '2. Diff', texto: 'Busca o MR pela API' },
  { titulo: '3. IA', texto: 'Claude, OpenAI, Gemini ou DeepSeek' },
  { titulo: '4. Comentário', texto: 'Publicado no MR, uma vez só' },
]
const niveis = [
  { titulo: 'Diff', texto: 'Título, descrição e alterações' },
  { titulo: 'Relacionado', texto: 'Issues, MRs e discussão (GitLab)' },
  { titulo: 'Agêntica', texto: 'A IA navega pelo repositório (GitLab)' },
  { titulo: 'Execução', texto: 'Testes em Docker sem rede nem segredos, com o sandbox habilitado (GitLab)' },
]
const pontos = [
  { titulo: 'Não repete', texto: 'Reenvio de webhook não gera segundo review' },
  { titulo: 'Falha com aviso', texto: 'Retenta com backoff e explica o erro no MR' },
  { titulo: 'Segredos cifrados', texto: 'Credenciais de IA em AES-256-GCM' },
  { titulo: 'Fácil de estender', texto: 'Clean Architecture e Strategy por plataforma' },
]
</script>

<template>
  <div ref="raiz" class="landing">
    <div ref="heroi" class="heroi">
      <LandingAbertura
        v-if="montado"
        :raiz="raiz!"
        :heroi="heroi!"
        :topo="topo!"
        :logo-topo="logoTopo!"
        :texto="texto!"
      />
      <header ref="topo" class="topo">
        <span ref="logoTopo" class="marca"><BrandLogo /></span>
        <span class="espaco"></span>
        <nav class="links" aria-label="Seções">
          <a href="#como-funciona">como funciona</a>
          <a href="#modos">modos</a>
          <a href="#painel">painel</a>
          <a href="#como-e-feito">como é feito</a>
        </nav>
        <button
          class="btn ghost"
          type="button"
          :aria-label="theme === 'dark' ? 'Tema claro' : 'Tema escuro'"
          :title="theme === 'dark' ? 'Tema claro' : 'Tema escuro'"
          @click="toggleTheme"
        >
          <AppIcon :name="theme === 'dark' ? 'sun' : 'moon'" :size="16" />
        </button>
        <router-link to="/login" class="btn primary">Entrar</router-link>
      </header>

      <div ref="texto" class="texto">
        <h1>Todo MR merece uma <mark>segunda leitura</mark>.</h1>
        <p>Um serviço em Go que recebe o webhook, lê o diff com IA e comenta no próprio MR, uma única vez.</p>
        <div class="acoes">
          <a href="#como-funciona" class="btn primary">Ver como funciona</a>
          <a :href="GITHUB" class="btn" target="_blank" rel="noopener">Código no GitHub</a>
        </div>
      </div>
    </div>

    <main class="trilho">
      <span class="commit" aria-hidden="true"></span>

      <section id="como-funciona">
        <h2>Como funciona</h2>
        <p>Você marca o usuário configurado como reviewer ou assignee, e o resto acontece sozinho.</p>
        <ol class="grade">
          <li v-for="p in passos" :key="p.titulo" class="caixa"><b>{{ p.titulo }}</b>{{ p.texto }}</li>
        </ol>
      </section>

      <section id="modos">
        <h2>Quatro níveis de revisão</h2>
        <p>
          As camadas se somam por perfil, definido por cliente ou por projeto. No GitLab, um comentário
          <code>/review deep</code> escolhe o nível daquele MR.
        </p>
        <ul class="grade">
          <li v-for="n in niveis" :key="n.titulo" class="caixa">
            <b><span class="sinal">+</span> {{ n.titulo }}</b>{{ n.texto }}
          </li>
        </ul>
      </section>

      <section id="painel">
        <h2>O painel</h2>
        <p>
          Prompts, modelos e perfis por cliente e projeto, com custo e erros à vista. Não há demonstração aberta:
          você vê o painel por estas capturas.
        </p>
        <div class="telas">
          <img :src="painel" alt="Captura do Painel" width="1440" height="900" loading="lazy" />
          <img :src="revisoes" alt="Captura das Revisões" width="1440" height="900" loading="lazy" />
        </div>
      </section>

      <section id="como-e-feito">
        <h2>Como é feito</h2>
        <p>A parte para quem quer ver como o sistema foi pensado.</p>
        <ul class="grade">
          <li v-for="p in pontos" :key="p.titulo" class="caixa"><b>{{ p.titulo }}</b>{{ p.texto }}</li>
        </ul>
      </section>
    </main>

    <footer class="fim">
      <span class="autor">Feito por Paulo Holanda</span>
      <span class="espaco"></span>
      <a :href="GITHUB" class="btn" target="_blank" rel="noopener">Código no GitHub</a>
      <a :href="SITE" class="btn" target="_blank" rel="noopener">Site pessoal</a>
    </footer>
  </div>
</template>

<style scoped>
/* Variáveis --abertura-*: a abertura animada as altera; sem ela valem os valores do estado final. */
.landing {
  overflow-x: clip;
  min-height: 100vh;
  background: var(--bg);
  color: var(--text);
  font-size: 15px;
}

.espaco {
  flex: 1;
}

mark {
  background: var(--highlight);
  color: var(--ink);
  padding: 0 0.15em;
}

code {
  font-family: var(--font-mono);
  font-size: 0.92em;
  color: var(--text-2);
}

.heroi {
  position: relative;
  display: flex;
  align-items: center;
  min-height: max(100svh, 560px);
  padding: 84px 64px 44px 22px;
}

.topo {
  position: absolute;
  inset: 0 0 auto 0;
  display: flex;
  align-items: center;
  gap: 18px;
  padding: 16px 22px;
  opacity: var(--abertura-topo, 1);
}

.marca {
  display: block;
}

.marca :deep(.brand-logo) {
  height: 28px;
  width: auto;
}

.links {
  display: flex;
  gap: 18px;
  font-size: 13px;
}

.links a {
  color: var(--text-2);
  text-decoration: none;
}

.links a:hover {
  color: var(--text);
}

.topo .btn {
  text-decoration: none;
}

.texto {
  position: relative;
  max-width: min(540px, 52%);
  opacity: var(--abertura-texto, 1);
  transform: translateY(var(--abertura-texto-y, 0px));
}

h1 {
  font-size: clamp(30px, 4.6vw, 52px);
  line-height: 1.02;
  font-weight: 800;
  margin-bottom: 14px;
}

.texto p {
  color: var(--text-2);
  max-width: 44ch;
  margin-bottom: 18px;
}

.acoes {
  display: flex;
  gap: 10px;
  flex-wrap: wrap;
}

.acoes .btn,
.fim .btn {
  padding: 10px 16px;
  font-size: 14px;
  text-decoration: none;
}

.trilho {
  position: relative;
  padding: 0 64px 8px 22px;
}

.trilho::before {
  content: '';
  position: absolute;
  top: 0;
  bottom: 0;
  right: 30px;
  width: 3px;
  background: var(--merge-mid);
  transform: scaleY(var(--abertura-trilho, 1));
  transform-origin: top;
}

.commit {
  display: none;
  position: absolute;
  top: 0;
  right: 26px;
  width: 11px;
  height: 11px;
  border-radius: 50%;
  background: var(--hl-1);
  opacity: var(--abertura-trilho, 1);
}

@media (prefers-reduced-motion: no-preference) {
  .commit {
    display: block;
    animation: desce 9s linear infinite;
  }
}

@keyframes desce {
  to {
    top: 98%;
  }
}

section {
  position: relative;
  padding: 26px 0;
  border-top: 1px solid var(--hairline);
  scroll-margin-top: 8px;
}

section::after {
  content: '';
  position: absolute;
  top: 34px;
  right: -40px;
  width: 15px;
  height: 15px;
  border-radius: 50%;
  background: var(--bg);
  border: 3px solid var(--merge-mid);
  opacity: var(--abertura-trilho, 1);
}

h2 {
  font-size: clamp(20px, 2.6vw, 27px);
  font-weight: 800;
  line-height: 1.15;
  margin-bottom: 6px;
}

section > p {
  color: var(--text-2);
  max-width: 60ch;
  margin-bottom: 14px;
}

.grade {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(135px, 1fr));
  gap: 10px;
  list-style: none;
}

.caixa {
  border: 1px solid var(--hairline);
  background: var(--panel-solid);
  padding: 10px 12px;
  font-size: 13px;
  color: var(--text-2);
}

.caixa b {
  display: block;
  color: var(--text);
  font-size: 14px;
  margin-bottom: 2px;
}

.sinal {
  color: var(--ok);
  font-family: var(--font-mono);
}

.telas {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 10px;
}

.telas img {
  display: block;
  width: 100%;
  height: auto;
  border: 1px solid var(--hairline);
}

.fim {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  padding: 22px;
  border-top: 3px solid;
  border-image: var(--spectrum) 1;
}

.autor {
  font-weight: 600;
}

@media (max-width: 699px) {
  .heroi {
    align-items: flex-start;
    min-height: max(100svh, 680px);
  }

  .texto {
    max-width: 100%;
  }

  .links {
    display: none;
  }
}
</style>
