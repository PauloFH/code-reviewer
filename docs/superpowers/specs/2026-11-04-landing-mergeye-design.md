# Landing page do Mergeye — design

## Intenção

O Mergeye ainda não tem página pública: quem abre `https://reviewer.pauloholanda.com.br` (o endereço que o portfólio aponta em `.portfolio/project.json`) cai direto no login. Este projeto cria essa página, no mesmo espírito da landing do pad (`~/projetos/pad`, bloco `landing` de `templates.html`): uma página que ajuda quem pode usar o sistema e que, vista por um recrutador, funciona como portfólio.

Quem chega deve sair sabendo o que o Mergeye faz e como funciona, e convencido de que o autor sabe construir sistemas robustos com IA integrados a terceiros (intenção já registrada na spec da identidade).

A ideia herdada do pad é a página ser feita do próprio assunto. Lá ela imita um documento sendo editado. Aqui, o símbolo do Mergeye já é um merge, e a página nasce dele: a logo se forma, seus ramos se abrem em dois fios, os fios se juntam em uma linha amarela e essa linha atravessa a página ligando as seções.

## Objetivos

- Em `/`, para quem não está logado, a landing. Para quem está logado, o painel, como hoje.
- A abertura animada descrita abaixo, tocando de ponta a ponta sem corte visual.
- Conteúdo verdadeiro: tudo o que a página afirma existe no sistema hoje.
- Temas escuro e claro, seguindo a identidade Mergeye.

## Fora de escopo

- Demonstração pública ou modo demo do produto. O painel é visto por capturas (decisão da spec da identidade).
- Hospedagem separada, domínio novo ou pré-renderização no servidor. A página é uma rota do SPA já embutido no binário Go.
- Mudanças no backend. O `frontendHandler` já devolve o `index.html` em `/`.
- Mudança de comportamento do painel, do login ou das permissões, fora o que a nova rota `/` exige.
- Formulário de contato, captura de e-mail, analytics, preços e cadastro.

## O que não pode acontecer

A página falha, mesmo funcionando, se:

- parecer um template genérico de produto de IA (as proibições da spec da identidade valem aqui: índigo ou roxo, Geist, ícones de traço fino, rótulos em mono maiúsculo, brilhos na interface, sparkles, cérebros);
- houver degrau, vão ou mudança de espessura visível entre o fio da abertura e a linha que desce pela página. Paulo pediu essa continuidade explicitamente;
- prometer o que o sistema não faz, como esconder que Relacionado e Agêntica hoje são só GitLab. Quem lê o código percebe, e isso derruba o objetivo de portfólio;
- obrigar o visitante a esperar a animação para ler. Ela sempre pode ser pulada;
- depender só de vermelho e verde para comunicar algo.

## Decisões de Paulo

- **Referência:** a landing do pad, com o duplo papel de ajudar o usuário e servir de portfólio.
- **Direção:** conceito B, "O merge comanda", em vez de imitar um MR (A) ou vitrine clássica do painel (C).
- **Abertura:** a logo se forma e depois segue para os fios. O protótipo mostrado foi aprovado ("gostei, pode usar ela mesmo").
- **Fio amarelo:** é a continuação exata da linha que desce pela página.
- **Endereço:** `/` do próprio serviço, para quem não está logado. Quem está logado vai para o painel.
- **Seções e ordem:** as do protótipo, listadas abaixo. Paulo não pediu mudança, mas elas não foram aprovadas item a item; os textos seguem provisórios.

## A abertura

Referência executável, a seguir de perto: `docs/superpowers/specs/assets/mergeye/landing-referencia.html`. É um fragmento HTML do protótipo aprovado. Os caminhos `/files/...` apontam para `web/src/assets/brand/` e `docs/media/`. Ele serve de referência de geometria e tempo, não de código a colar. O protótipo só foi visto dentro de um quadro: onde ele divergir deste documento sobre a primeira dobra e o topo, vale este documento.

Sequência, em cerca de 4 segundos:

1. **A logo se monta no centro da primeira dobra (0 a 2,2 s).** Linha e nó da esquerda, depois os ramos vermelho e verde com seus nós, a linha central, o bloco amarelo, `</>`, o check, o nó de merge e a saída da direita. O topo e o texto ainda não aparecem.
2. **Pausa com a logo pronta (até 2,7 s).**
3. **Os ramos viram os fios (2,7 a 3,7 s).** Os mesmos traços vermelho e verde se abrem e afinam até a forma final. A saída da direita amarela e vira o fio que desce. O resto do símbolo encolhe até a logo do topo, que aparece no lugar dele, junto com o restante do topo (links, tema e `Entrar`).
4. **A página aparece (3,25 a 4,2 s).** Título e botões entram, e a linha das seções se revela de cima para baixo.

**Primeira dobra.** Topo e herói juntos ocupam a primeira dobra da janela, e a logo se monta no centro dela. Nada de "Como funciona" aparece durante a montagem.

Regras de geometria que garantem a continuidade, já provadas no protótipo:

- Os dois ramos e o fio amarelo compartilham o mesmo ponto desde o começo, escondido sob o nó de merge. Assim nunca há vão durante a abertura.
- O desenho é calculado em pixels reais, sem esticar com `preserveAspectRatio="none"`, para o traço manter a espessura.
- O fim do fio amarelo cai exatamente no centro da linha das seções (largura do herói − 31,5 px no protótipo) e na borda inferior do herói. A diferença medida é 0 px em 1280 px de largura, e isso deve continuar valendo em qualquer largura.
- Os fios mantêm a proporção aprovada em qualquer altura de janela: não esticam para preencher um herói alto. O que sobra de altura vira trecho reto do fio amarelo até a borda inferior do herói.
- O herói é redesenhado ao redimensionar a janela.

**Logo em SVG.** O símbolo é redesenhado em SVG só para ser animado, reproduzindo o desenho oficial sem criar outro, como a spec da identidade permite. Fora da abertura, a marca continua sendo `BrandLogo`.

## Seções

Nesta ordem. As seções 2 a 5 têm um nó cada na linha amarela, como no protótipo: quatro nós. A linha nasce no herói e termina no rodapé. O topo tem a logo, links âncora para as seções, troca de tema e `Entrar` (vai para `/login`).

1. **Herói.** "Todo MR merece uma segunda leitura." Uma frase do que é o serviço. Botões `Ver como funciona` (âncora) e `Código no GitHub`.
2. **Como funciona.** Marcar o usuário como reviewer ou assignee dispara o resto: webhook (GitLab, GitHub ou Gitea), diff, IA (Claude, OpenAI, Gemini ou DeepSeek) e comentário no MR, uma vez só.
3. **Quatro níveis de revisão.** Diff; Relacionado (issues, MRs e discussão, GitLab); Agêntica (a IA navega pelo repositório, GitLab); Execução (testes em Docker sem rede nem segredos, GitLab, quando o sandbox está habilitado). Os níveis se escolhem por perfil. O comando `/review <preset>` pode ser citado, mas como recurso do GitLab: só o webhook do GitLab o trata, e nenhum preset liga a Execução (`ProfileFromPreset`, em `internal/domain/review_profile.go`). O texto do protótipo sobre isso está impreciso e não deve ser copiado.
4. **O painel.** Prompts, modelos e perfis por cliente e projeto, com custo e erros à vista. Capturas de Painel e Revisões. Deixar claro que não há demonstração aberta.
5. **Como é feito.** A seção do recrutador: não repete (idempotência), falha com aviso (retentativa com backoff e erro explicado no MR), segredos cifrados (AES-256-GCM) e fácil de estender (Clean Architecture e Strategy por plataforma).
6. **Rodapé.** "Feito por Paulo Holanda", `Código no GitHub` (`https://github.com/PauloFH/code-reviewer`, público) e `Site pessoal` (`https://www.pauloholanda.com.br`).

O texto final é do construtor, em português, e cada afirmação deve ser conferida contra o código ou o `CLAUDE.md`. Mantêm-se as ressalvas "GitLab".

## Decisões do autor deste documento (podem ser revistas)

- **Quando a abertura toca:** uma vez por sessão do navegador. Depois disso, a página abre direto no estado final.
- **Como pular:** clique na cena, rolagem, toque ou qualquer tecla vão direto ao estado final.
- **`prefers-reduced-motion`:** sem abertura, direto ao estado final, e o commit que percorre a linha fica parado.
- **Idioma:** só português, como a landing do pad e o resto da interface.
- **Tema:** segue o `useTheme()` atual, com escuro como padrão. O tema claro troca os traços claros do símbolo pela tinta escura, como a logo clara oficial, e usa `mergeye-painel-claro.png`. A captura de Revisões só existe no escuro: gerar a clara com `web/tests/mock-api.mjs`.
- **Primeira dobra e topo:** o herói ocupa a primeira dobra e o topo inteiro só entra na transição, como descrito em "A abertura". O protótipo mantinha links e `Entrar` visíveis e um herói mais baixo.
- **Nós da linha:** quatro, como no protótipo, sem nó no herói nem no rodapé.
- **Fundo animado:** `BackgroundMotion` fica fora da landing, porque roda `requestAnimationFrame` sem parar.
- **Commit percorrendo a linha:** mantido do protótipo, depois da abertura.
- **Prévia ao compartilhar o link:** meta tags de título, descrição e imagem (`og:*`) no `index.html`, usando `.portfolio/capa.png` ou equivalente.

## Ponto de partida no código

- **Roteamento (`web/src/router/index.ts`).** `/` hoje é o pai de `AppLayout`, com filho `''` redirecionando para `/dashboard`. A landing entra como rota pública em `/`. Quem tem sessão (`auth.isAuthenticated`) é redirecionado para `/dashboard`, sem a landing piscar antes. Rotas filhas do painel não mudam de URL. O guard de rotas restritas e o `router.push('/dashboard')` do login continuam valendo.
- **Token expirado.** Quem volta com token vencido vai ao painel e o interceptor de 401 leva a `/login`, como já acontece hoje. Isso é aceitável.
- **Sem chamadas de API.** Com visitante anônimo, a landing não faz requisição autenticada nem abre o feed ao vivo.
- **Regras do frontend (`web/CLAUDE.md`).** Cor literal só em `tokens.css`: as cores da logo e dos fios viram tokens, inclusive os gradientes. Texto em PT-BR.
- **Varredura `web/tests/sweep.test.ts`.** Ela falha com `<svg` fora de `AppIcon.vue`. O SVG animado precisa de uma exceção explícita para o componente da abertura na própria varredura, com um comentário dizendo por quê. Não contornar criando o SVG por script só para escapar do teste.
- **Capturas.** Estão em `docs/media/`, fora de `src/`. Devem passar pelo build do Vite (cópia ou import em `web/src/assets`), sem depender de caminho do repositório em produção.
- **Build.** `internal/transport/http/frontend/dist/index.html` é versionado e precisa acompanhar o build.
- **Referência dos fatos do produto.** `CLAUDE.md`, `README.md` e `.portfolio/project.json`.

## Restrições

- Contraste de texto em nível WCAG AA nos dois temas (o teste de contraste dos tokens deve cobrir os novos que vão sob texto).
- Sem nova dependência: CSS, SVG e `requestAnimationFrame` resolvem.
- Corpo da página sem rolagem horizontal em nenhuma largura. Em tela de celular (cerca de 400 px) o texto do herói fica sempre legível e sem fios passando por cima, e a continuidade do fio amarelo vale também ali.
- `requestAnimationFrame` só roda durante a abertura.
- O conteúdo não depende da animação terminar para existir no DOM, e o SVG decorativo fica fora da árvore de acessibilidade (`aria-hidden`).
- Sem regressão: `npm run type-check`, `npm test`, `npm run lint` e `npm run build` em `web/`, e `go test ./...`, continuam passando.
- Verificação visual em navegador real: quadros da abertura, estado final nos dois temas, celular e movimento reduzido.

## Deixado para quem construir

- Estrutura de componentes e nomes de arquivos, seguindo as convenções de `web/CLAUDE.md`.
- Como os fios se acomodam no herói em telas estreitas, dentro das restrições acima.
- Tamanho final das colunas e o vazio entre título e fios em telas muito largas.
- Ajuste fino de tempos e curvas da animação, sem alterar a sequência.

## Git

O trabalho fica na branch `feat/landing-mergeye`, criada a partir de `origin/main`, que já contém a identidade Mergeye, em uma pasta própria (`git worktree`). A pasta principal do repositório segue em uso na branch `feat/deploy-github`, ainda não integrada. Mensagens de commit em uma linha, `tipo: mensagem`, sem escopo, corpo ou coautor, inclusive em commits de subagentes.
