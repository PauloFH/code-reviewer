<script setup lang="ts">
import BrandLogo from '@/components/BrandLogo.vue'

const EMAIL = 'paulorfholanda@gmail.com'
</script>

<template>
  <div class="legal">
    <header class="topo">
      <router-link to="/" class="marca" aria-label="Mergeye, voltar ao início"><BrandLogo /></router-link>
    </header>

    <main class="texto">
      <h1>Política de privacidade</h1>
      <p class="data">Atualizada em 9 de novembro de 2026.</p>

      <p>
        Esta política explica quais dados pessoais esta instância do Mergeye trata, por quê e com quem os compartilha,
        conforme a Lei Geral de Proteção de Dados (Lei nº 13.709/2018, LGPD).
      </p>

      <h2>Quem é o responsável</h2>
      <p>
        O controlador dos dados é Paulo Holanda, pessoa física, que opera esta instância. Para falar sobre privacidade
        ou exercer seus direitos, escreva para <a :href="`mailto:${EMAIL}`">{{ EMAIL }}</a>.
      </p>

      <h2>Visitantes desta página</h2>
      <p>
        A página inicial e esta política não têm formulário, não usam cookies próprios nem scripts de publicidade ou de
        análise de terceiros. A escolha de tema claro ou escuro fica só no seu navegador (<code>localStorage</code>) e
        não é enviada a ninguém.
      </p>
      <p>
        O acesso passa pela Cloudflare, que fornece a conexão segura (HTTPS) e a proteção contra ataques, e por um
        servidor próprio hospedado na Oracle Cloud, em São Paulo. Ambos recebem dados técnicos da requisição, como endereço IP, data e hora, página
        acessada e navegador, e a Cloudflare pode usar cookies estritamente necessários à segurança. Os registros de
        acesso do servidor são guardados por 6 meses, como pede o Marco Civil da Internet (art. 15). Base legal:
        legítimo interesse em manter o serviço seguro e funcionando (art. 7º, IX).
      </p>

      <h2>Usuários do painel</h2>
      <p>
        O painel é restrito a usuários cadastrados por um administrador; não há cadastro aberto. Tratamos o nome de
        usuário, a senha (guardada apenas como hash bcrypt, nunca em texto) e o papel de acesso. Ao entrar, um token de
        sessão válido por 24 horas fica no seu navegador. Enquanto você usa o painel, ele envia métricas de desempenho
        das páginas (tempos de carregamento e de resposta), associadas ao seu nome de usuário, que ficam nos registros do
        serviço por até 6 meses. Base legal: legítimo interesse em controlar
        o acesso e manter o painel rápido (art. 7º, IX).
      </p>

      <h2>Autores de merge requests revisados</h2>
      <p>
        Quando o usuário configurado é marcado como revisor de um merge request ou pull request, o serviço busca na
        plataforma Git (GitLab, GitHub ou Gitea) o título, a descrição, o diff, o número, o link e o nome de usuário do
        autor. Título, descrição, diff, link e nome do projeto são enviados ao provedor de IA configurado para gerar a
        revisão, que é publicada como comentário no próprio merge request. Conforme o perfil de revisão, a IA também
        recebe as issues e os merge requests relacionados e a discussão do merge request, com o nome de usuário de quem
        comentou, e pode ler outros arquivos do repositório.
      </p>
      <p>
        O histórico de revisões (cliente, projeto, título, link, nome de usuário do autor, tokens, custo, situação e
        mensagem de erro) é guardado por 12 meses e apagado automaticamente depois disso. No mesmo prazo é apagada a
        cópia do texto da revisão; fica só o registro de que aquele merge request já foi revisado (cliente, projeto,
        número e usuário do robô), para não revisá-lo duas vezes. Base legal: legítimo interesse do titular do
        repositório em revisar o código (art. 7º, IX).
      </p>

      <h2>Com quem compartilhamos</h2>
      <ul>
        <li>
          <b>Anthropic</b> (Claude) e <b>OpenAI</b>, nos Estados Unidos: recebem o conteúdo do merge request para gerar
          a revisão.
        </li>
        <li><b>Cloudflare</b>: intermedia o acesso ao site, como descrito acima.</li>
        <li><b>Oracle Cloud</b>, em São Paulo: hospeda o servidor e o banco de dados; os dados guardados ficam no Brasil.</li>
        <li><b>A plataforma Git</b> de onde veio o merge request: recebe o comentário com a revisão.</li>
      </ul>
      <p>
        O envio aos provedores de IA é uma transferência internacional de dados (art. 33), feita com
        base nas cláusulas-padrão contratuais aprovadas pela ANPD (Resolução CD/ANPD nº 19/2024).
        Não vendemos dados pessoais nem os usamos para publicidade.
      </p>

      <h2>Segurança</h2>
      <p>
        O tráfego usa HTTPS. As credenciais dos provedores de IA e os tokens de acesso às plataformas Git são cifrados em repouso
        (AES-256-GCM), as senhas
        ficam guardadas só como hash e o painel exige login.
      </p>

      <h2>Seus direitos</h2>
      <p>
        Pelo art. 18 da LGPD, você pode pedir a confirmação de que tratamos seus dados, o acesso a eles, a correção, a
        anonimização, o bloqueio ou a eliminação dos desnecessários, a portabilidade, informações sobre com quem foram
        compartilhados e a revisão de decisões automatizadas, e pode se opor a um tratamento. Faça o pedido pelo contato
        acima; respondemos em até 15 dias. Você também pode reclamar à Autoridade Nacional de Proteção de Dados (ANPD).
      </p>

      <h2>Mudanças</h2>
      <p>Se esta política mudar, a data no topo é atualizada.</p>

      <p class="voltar"><router-link to="/" class="btn">Voltar ao início</router-link></p>
    </main>
  </div>
</template>

<style scoped>
.legal {
  min-height: 100vh;
  background: var(--bg);
  color: var(--text);
  font-size: 15px;
  padding: 16px 22px 40px;
}

.marca {
  display: inline-block;
}

.marca :deep(.brand-logo) {
  height: 28px;
  width: auto;
}

.texto {
  max-width: 68ch;
  margin: 32px auto 0;
  line-height: 1.6;
}

h1 {
  font-size: clamp(26px, 4vw, 40px);
  font-weight: 800;
  line-height: 1.1;
  margin-bottom: 6px;
}

h2 {
  font-size: 19px;
  font-weight: 800;
  margin: 28px 0 8px;
}

p,
li {
  color: var(--text-2);
}

p + p {
  margin-top: 10px;
}

ul {
  padding-left: 20px;
  margin-bottom: 10px;
}

li + li {
  margin-top: 6px;
}

b {
  color: var(--text);
}

a {
  color: var(--text);
}

code {
  font-family: var(--font-mono);
  font-size: 0.92em;
}

.data {
  font-family: var(--font-mono);
  font-size: 13px;
  margin-bottom: 18px;
}

.voltar {
  margin-top: 32px;
}

.voltar .btn {
  text-decoration: none;
}
</style>
