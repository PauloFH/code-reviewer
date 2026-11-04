# Identidade visual e repaginada do painel — design

## Intenção

O Mergeye (repositório `code-reviewer`) é um projeto de exemplo público de Paulo Holanda. Quem chega vem do site pessoal ou do GitHub, quer entender o que é o sistema e o que ele tem, e deve sair convencido de que o autor sabe desenvolver sistemas robustos com IA integrados a terceiros.

O painel atual funciona, mas incomoda: ícones, cores e interface seguem um padrão visual característico de produto gerado por IA, sem identidade própria. Este projeto cria uma identidade individual, nascida do universo do code review, e a aplica a todo o painel interno. A referência de atitude é <https://deepbook-v2-wip.webflow.io/>: visual limpo, movimento e identidade ligada ao tema. O princípio a aproveitar é o motivo gráfico sair do assunto; o site não deve ser copiado.

## Objetivos

- Uma identidade reconhecível, baseada em marca-texto, marcações vermelha e verde (diff) e sistema de merge.
- Todas as telas do painel com essa identidade, nos temas escuro e claro.
- A logo do Mergeye aplicada no painel, nos dois temas.
- Animações pequenas e translúcidas de commits subindo e branches sendo puxadas.
- Uma base (tokens, componente de ícones, logo) que a landing page possa reaproveitar depois.
- Capturas de tela e um vídeo curto do painel, com dados de exemplo, nos dois temas, publicados no README. É assim que os visitantes verão o painel: não haverá demo pública.

## Fora de escopo

- **Landing page.** É um projeto separado, pausado, feito depois deste e herdando esta identidade.
- **Comportamento do sistema.** Rotas, chamadas de API, permissões, formulários e regras continuam iguais. Mudam só aparência, textos e animações. Exceção aprovada por Paulo: o feed "Ao vivo" passa a mostrar publicadas, erros e ignoradas, e não só publicadas.
- **Backend em Go**, exceto o `index.html` gerado em `internal/transport/http/frontend/dist/`.
- **Renomear repositório, módulo Go, binário ou imagem Docker.** Continuam como `code-reviewer`; o nome novo vale para a interface.
- **Compra de domínio e registro de marca.**
- **Demo pública ou modo de demonstração no produto.** Os dados de exemplo das capturas são preparados só para a captura, sem código novo no sistema.

## O que não pode acontecer

A identidade falha, mesmo funcionando, se o painel:

- continuar parecendo um template genérico de produto de IA: índigo ou roxo, Geist, bibliotecas de ícones de traço fino, rótulos em mono maiúsculo, brilhos, sparkles ou cérebros;
- ficar preso a um único tema de grafo com cor chapada, direção testada e rejeitada;
- tiver animações que atrapalhem a leitura de tabelas e formulários;
- depender só de vermelho e verde para comunicar estado.

## Decisões de Paulo

- **Nome:** Mergeye (merge + eye), com essa grafia. Substitui "Code Reviewer" em tudo que o usuário vê.
- **Direção:** composição "C. O merge comanda". Fios vermelhos e verdes convergem, passam pelo amarelo e viram uma linha só.
- **Ingredientes:** marca-texto, marcações vermelha e verde e merge, usados em conjunto. Branches são um elemento entre outros, não o tema único.
- **Cor:** com gradiente, sem depender de cor chapada.
- **Temas:** escuro e claro.
- **Logo:** criada por Paulo e oficial, com o olho formado por branches vermelha e verde, um bloco de código aprovado no centro e o nome Mergeye. Original em `~/Downloads/mergeEye.png`.
- **Animações:** pequenas, de commits subindo e branches sendo puxadas, como fundos translúcidos.
- **Como o visitante vê o painel:** por capturas de tela e vídeo.
- **Aparência geral:** aprovada a partir do exemplo `painel-novo-design.html`.
- **Intensidade:** movimento forte na landing page e no login; contido no uso diário do painel.

## Decisões do autor deste documento (podem ser revistas)

**Cor.** Assinatura em gradiente de vermelho, amarelo e verde: o que sai, o que foi marcado e o que entra. Valores de partida, a ajustar por contraste:

| Papel | Escuro | Claro |
|---|---|---|
| Fundo | `#0b0c10` | `#fbf8ef` a `#efe9da` |
| Texto | `#f6f6f0` | `#16140e` |
| Vermelho (saiu, erro) | `#ff4d3d` | `#dc2a18` |
| Verde (entrou, publicada) | `#1fd97c` | `#07924f` |
| Marca-texto | `#fff15a` a `#ffc917` | `#fff15a` a `#ffd21f` |

O tema escuro continua como padrão, e o botão de troca atual permanece.

**Uso dos elementos.**
- **Marca-texto:** item ativo do menu e destaques pontuais, sempre com texto escuro por cima.
- **Gradiente:** detalhes como barras e divisórias, nunca sob texto corrido ou em grandes áreas do painel.
- **Status:** sinal mais cor. `+` para publicada (`success`), `−` para erro (`error`) e `~` para ignorada (`skipped`).

**Animações de fundo no painel.** Uma camada permanente no `AppLayout`, atrás do conteúdo, em todas as telas autenticadas, com baixa opacidade e movimento lento. Cartões e tabelas usam superfícies quase opacas (cerca de 90%), para o fundo aparecer sem reduzir o contraste do texto. A camada pausa com a aba oculta e fica estática com `prefers-reduced-motion`. No login, ela pode ser mais presente. Interações comuns, como trocar de tela e abrir modais, usam só transições curtas.

**Textos.** Todo o texto do frontend fica em português, eliminando a mistura atual (`Sign in`, `Events`, `Clients`, `Users`, `Total Reviews`, `Live feed`). Termos técnicos consagrados, como `merge`, `branch`, `commit` e `token`, podem ficar em inglês. Mensagens de erro vindas do backend continuam como estão, porque o backend está fora de escopo. O rótulo fixo `v 2.4 · prod` sai, porque não reflete o ambiente real. Os atalhos de teclado continuam funcionando: `P` Painel, `R` Revisões, `C` Clientes, `O` Prompts, `M` Modelos, `F` Perfis de revisão e `U` Usuários.

**Aprovação em etapas.** A implementação para depois de logo, tokens, primeiros ícones e duas telas (login e Painel) para Paulo aprovar. As demais telas só migram depois disso, e as decisões desta seção ainda podem mudar nesse ponto.

**Logo.** Os arquivos sem fundo estão em `docs/superpowers/specs/assets/mergeye/`, para cada tema, em versão empilhada, só símbolo e horizontal, com o nome ao lado. A horizontal é a usada no menu, em tamanho pequeno. A versão escura é fiel ao original. Na clara, o branco virou tinta escura e as cores foram escurecidas um pouco, para ler sobre papel. A logo é um PNG de cerca de 900px e pode ser usada assim. Se precisar de mais nitidez, uma versão em SVG deve reproduzir este desenho, sem criar outro. Abaixo de 32px, os detalhes internos se perdem, então o favicon usa um símbolo simplificado derivado dela. O brilho da logo é parte do desenho de Paulo e não contraria a regra contra brilhos, que vale para a interface.

## Telas

Login, Painel (`DashboardView`), Revisões (`EventsView`), Clientes, Prompts, Modelos (`AiConfigView`), Perfis de revisão e Usuários, além de layout, modais, formulários, toasts, feed ao vivo, tabela de eventos, editor de texto e gráfico de custo.

## Ponto de partida no código

- Frontend em `web/`, com Vue 3 e CSS próprio, sem biblioteca de UI. Não precisa de nova biblioteca de componentes.
- `web/src/assets/tokens.css` concentra variáveis e classes base, e a maior parte das cores já passa por ele.
- 16 arquivos `.vue` têm cores fixas fora dos tokens, incluindo o índigo do gráfico em `CostChart.vue`.
- 49 ícones SVG estão colados em 11 arquivos, sem componente central.
- As fontes vêm do Google Fonts, referenciadas em `web/index.html`, junto com a cor de fundo inicial.
- `internal/transport/http/frontend/dist/index.html` é versionado e precisa acompanhar o build.
- `web/CLAUDE.md` descreve uma paleta Slate que já está desatualizada e deve ser corrigido.

## Restrições

- Contraste de texto em nível WCAG AA nos dois temas, inclusive sobre o marca-texto.
- Sem nova dependência para o que CSS, SVG ou Canvas resolvam.
- Painel utilizável em telas pequenas, como hoje.
- Sem regressão: `npm run type-check` e `npm run build` em `web/` e `go test ./...` continuam passando.

## Referências visuais

Rascunhos locais, ainda não versionados, em `.superpowers/brainstorm/637840-1793801627/content/`:

- `painel-novo-design.html`: exemplo do Painel na nova identidade, nos dois temas e com as animações de fundo. É a principal referência de aplicação.
- `identidade-v2.html`: direção escolhida, opção C. Serve de referência de atitude, não de layout final.
- `direcoes-identidade.html`, `painel-grafo.html` e `logo-olho-*.png`: direções e conceitos substituídos, úteis só como histórico.

## Deixado para quem construir

- Tipografia final, desde que tenha personalidade e não seja Geist. Bricolage Grotesque e DM Mono foram usadas nos rascunhos só como provisórias.
- Layout de cada tela e desenho dos ícones, seguindo a nova linguagem.
- Técnica e posição exatas das animações, dentro das regras acima.
- Ordem de migração das telas.
