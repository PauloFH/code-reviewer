# Guia de Implantação

Implantação do Code Reviewer em servidor Linux sem Docker. Recomendado para VMs com recursos limitados (1 GB RAM ou menos).

**Testado em:** Canonical Ubuntu 24.04 Minimal · 1 GHz · 1 GB RAM  
**Runtime:** binário Go autocontido + Nginx + systemd

---

## Arquitetura

```
Usuário / Plataforma Git
        │ HTTPS
        ▼
  Cloudflare (proxy + TLS)
        │ HTTP
        ▼
   VM porta 80 — Nginx (reverse proxy)
        │ HTTP interno
        ▼
  localhost:8080 — code-reviewer (Go)
```

Porta 8080 **não é exposta** para a internet — apenas Nginx acessa internamente.  
Cloudflare encerra o TLS; o tráfego VM-interna é HTTP simples.

---

## Pré-requisitos

- Ubuntu 24.04 (ou derivado)
- Acesso SSH com `sudo`
- Domínio gerenciado no Cloudflare com DNS apontando para o IP da VM
- Portas 80 e 443 abertas no firewall da rede (security list / NSG)

---

## 1. Preparar o servidor

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl ca-certificates gcc musl-dev openssl nginx
```

### Swap (obrigatório com 1 GB RAM)

Protege contra OOM durante o build do frontend.

```bash
sudo fallocate -l 1G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile
echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab
```

---

## 2. Instalar Go

```bash
# x86_64
curl -OL https://go.dev/dl/go1.24.3.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.24.3.linux-amd64.tar.gz

# aarch64 (ARM64) — descomente se necessário
# curl -OL https://go.dev/dl/go1.24.3.linux-arm64.tar.gz
# sudo tar -C /usr/local -xzf go1.24.3.linux-arm64.tar.gz

echo 'export PATH=$PATH:/usr/local/go/bin' | sudo tee /etc/profile.d/go.sh
source /etc/profile.d/go.sh
go version
```

---

## 3. Instalar Node.js 22

Necessário apenas para o build do frontend. Pode ser removido após.

```bash
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs
node -v   # v22.x
```

---

## 4. Clonar o repositório

```bash
sudo mkdir -p /opt/code-reviewer
sudo chown $USER:$USER /opt/code-reviewer

git clone https://github.com/paulofh/code-reviewer.git /opt/code-reviewer
cd /opt/code-reviewer
```

---

## 5. Build do frontend

```bash
cd /opt/code-reviewer/web
npm ci
npm run build
# saída em internal/transport/http/frontend/dist (embutida no binário Go)
cd /opt/code-reviewer
```

---

## 6. Build do binário

```bash
go mod download
CGO_ENABLED=1 go build -ldflags="-s -w" -o code-reviewer ./cmd/server
ls -lh code-reviewer   # ~20–30 MB com SPA embutida
```

---

## 7. Configurar variáveis de ambiente

```bash
cp .env.example .env
nano .env
```

Preencha os campos obrigatórios:

```env
APP_PORT=8080
APP_ENV=production

# Provider de IA: gemini ou chat_completions
AI_PROVIDER=gemini

# API Gemini generateContent
GEMINI_API_KEY=cole-sua-chave-gemini-aqui
GEMINI_MODEL=gemini-flash-latest
GEMINI_BASE_URL=https://generativelanguage.googleapis.com/v1beta

# Para AI_PROVIDER=chat_completions (Claude/OpenAI-compatible proxy):
# AI_BASE_URL=https://sua-api-de-ia.com
# AI_TOKEN=seu-token-aqui
# AI_MODEL=claude-sonnet-4
AI_TIMEOUT_SECONDS=120

# Preço por milhão de tokens (USD)
AI_PRICE_PER_M_INPUT_TOKENS=3.0
AI_PRICE_PER_M_OUTPUT_TOKENS=15.0

DATABASE_PATH=/opt/code-reviewer/data/reviewer.db
CLIENTS_CONFIG_PATH=/opt/code-reviewer/clients.json

DASHBOARD_USER=admin
DASHBOARD_PASSWORD=senha-forte-aqui
DASHBOARD_JWT_SECRET=cole-aqui-o-resultado-do-openssl

LOG_LEVEL=info
```

Gere o `DASHBOARD_JWT_SECRET`:

```bash
openssl rand -hex 32
```

---

## 8. Configurar clients de plataforma

```bash
cp clients.example.json clients.json
nano clients.json
```

Mantenha apenas as plataformas que você usa. Exemplo mínimo com GitHub:

```json
[
  {
    "name": "meu-github",
    "platform": "github",
    "base_url": "https://api.github.com",
    "token": "ghp_xxxxxxxxxxxxxxxxxxxx",
    "webhook_secret": "segredo-configurado-no-webhook",
    "username": "seu-usuario-github"
  }
]
```

O campo `name` define a URL do webhook: `POST /webhook/meu-github`.  
Na primeira inicialização com banco vazio, os clients deste arquivo são importados automaticamente.

---

## 9. Criar diretório de dados

```bash
mkdir -p /opt/code-reviewer/data
```

---

## 10. Serviço systemd

```bash
sudo adduser --system --group --home /var/lib/code-reviewer --shell /usr/sbin/nologin code-reviewer
sudo chown -R code-reviewer:code-reviewer /opt/code-reviewer/data
sudo chmod -R u=rwX,go= /opt/code-reviewer/data
sudo chown code-reviewer:code-reviewer /opt/code-reviewer/.env /opt/code-reviewer/clients.json
sudo chmod 600 /opt/code-reviewer/.env /opt/code-reviewer/clients.json
sudo nano /etc/systemd/system/code-reviewer.service
```

```ini
[Unit]
Description=Code Reviewer
After=network.target

[Service]
Type=simple
# Usuário sem sudo, dono de .env, clients.json e data/.
User=code-reviewer
WorkingDirectory=/opt/code-reviewer
EnvironmentFile=/opt/code-reviewer/.env
ExecStart=/opt/code-reviewer/code-reviewer
Restart=on-failure
RestartSec=5
StandardOutput=journal
StandardError=journal
NoNewPrivileges=true
PrivateTmp=true

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl daemon-reload
sudo systemctl enable code-reviewer
sudo systemctl start code-reviewer
sudo systemctl status code-reviewer
```

Verificar:

```bash
curl http://localhost:8080/healthz
# → 200 OK
```

---

## 11. Nginx — reverse proxy

Substitua `reviewer.seu-dominio.com` pelo seu subdomínio real.

```bash
sudo nano /etc/nginx/sites-available/code-reviewer
```

```nginx
server {
    listen 80;
    server_name reviewer.seu-dominio.com;

    # Só aceita tráfego do Cloudflare (conf.d/cloudflare.conf, abaixo)
    if ($is_cloudflare = 0) { return 444; }

    # Tamanho máximo de body — diffs grandes podem chegar a poucos KB
    client_max_body_size 2M;

    location / {
        proxy_pass         http://127.0.0.1:8080;
        proxy_http_version 1.1;

        proxy_set_header Host              $host;
        proxy_set_header X-Real-IP         $remote_addr;
        proxy_set_header X-Forwarded-For   $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;

        # SSE (live feed do dashboard) — sem buffering
        proxy_set_header Connection        '';
        proxy_buffering                    off;
        proxy_cache                        off;
        proxy_read_timeout                 3600s;
    }
}
```

```bash
sudo ln -s /etc/nginx/sites-available/code-reviewer /etc/nginx/sites-enabled/

# Faixas do Cloudflare: o IP real só é aceito delas, e $is_cloudflare identifica quem chega direto no IP.
# O geo usa $realip_remote_addr, o par TCP real, antes da substituição feita pelo real_ip.
IPS="$(curl -fsS https://www.cloudflare.com/ips-v4) $(curl -fsS https://www.cloudflare.com/ips-v6)"
if [ "$(echo $IPS | wc -w)" -ge 20 ]; then
  { for ip in $IPS; do printf 'set_real_ip_from %s;\n' "$ip"; done
    printf 'real_ip_header CF-Connecting-IP;\n\n'
    printf 'geo $realip_remote_addr $is_cloudflare {\n    default 0;\n'
    for ip in $IPS; do printf '    %s 1;\n' "$ip"; done
    printf '}\n'
  } | sudo tee /etc/nginx/conf.d/cloudflare.conf >/dev/null
else
  echo 'lista do Cloudflare incompleta; conf.d/cloudflare.conf não foi alterado'
fi
sudo nginx -t
sudo systemctl reload nginx
```

---

## 12. Cloudflare — DNS e SSL

### 12.1 Registro DNS

No painel Cloudflare do seu domínio:

1. **DNS → Records → Add record**
2. Tipo: `A`
3. Name: `reviewer` (ou o subdomínio que preferir)
4. IPv4 address: IP público da VM
5. Proxy status: **Proxied** (nuvem laranja — obrigatório para o SSL funcionar)

### 12.2 Modo SSL

**SSL/TLS → Overview → Encryption mode:** selecione **Flexible**

> **Flexible** = Cloudflare ↔ usuário em HTTPS; Cloudflare ↔ VM em HTTP.  
> Funciona porque a VM não tem certificado próprio. O tráfego interno à rede da VM já é privado.

### 12.3 Regras de cache (opcional mas recomendado)

**Rules → Cache Rules → Create rule:**

- Expressão: `(http.host eq "reviewer.seu-dominio.com")`
- Cache status: **Bypass**

Isso evita que o Cloudflare cache respostas da API ou do SSE.

### 12.4 Verificar

```bash
curl https://reviewer.seu-dominio.com/healthz
# → 200 OK
```

---

## 13. Firewall interno da VM

As portas 80 e 443 ficam abertas na security list da rede e nas regras `iptables` da imagem. Quem restringe ao Cloudflare é o Nginx, pelo `$is_cloudflare` da seção 11. Não instale o `ufw` em VMs da Oracle: ele pode entrar em conflito com as regras da imagem e bloquear o acesso.

Depois de cadastrar um site novo neste servidor, inclua `if ($is_cloudflare = 0) { return 444; }` no bloco dele. Para conferir de fora, os acessos diretos pelo IP não devem responder (`000`), e o domínio deve dar 200:

```bash
curl -s -m 8 -H 'Host: <dominio-do-painel>' -o /dev/null -w '%{http_code}\n' http://<ip-do-servidor>/healthz
curl -sk -m 8 --resolve <dominio-do-painel>:443:<ip-do-servidor> -o /dev/null -w '%{http_code}\n' https://<dominio-do-painel>/healthz
curl -s -m 8 -o /dev/null -w '%{http_code}\n' http://<ip-do-servidor>:8080/healthz
curl -s -m 8 -o /dev/null -w '%{http_code}\n' https://<dominio-do-painel>/healthz   # 200
```

As faixas do Cloudflare mudam raramente; se mudarem, gere `conf.d/cloudflare.conf` de novo.

A porta 8080 **não deve ser aberta** — o acesso externo passa exclusivamente pelo Nginx.

---

## 14. Liberar espaço após o build

Go e Node não são necessários em runtime.

```bash
sudo apt remove -y nodejs
sudo rm -rf /usr/local/go ~/go /opt/code-reviewer/web/node_modules
sudo apt autoremove -y
```

---

## 15. Configurar webhooks nas plataformas

Use o domínio completo com HTTPS em todas as plataformas.

### GitHub

1. Repositório → **Settings → Webhooks → Add webhook**
2. **Payload URL:** `https://reviewer.seu-dominio.com/webhook/<nome-do-client>`
3. **Content type:** `application/json`
4. **Secret:** valor de `webhook_secret` no `clients.json`
5. **Events:** selecione **Let me select individual events** → marque apenas **Pull requests**

### GitLab

1. Projeto → **Settings → Webhooks**
2. **URL:** `https://reviewer.seu-dominio.com/webhook/<nome-do-client>`
3. **Secret token:** valor de `webhook_secret` no `clients.json`
4. **Trigger:** marque apenas **Merge request events**

### Gitea

1. Repositório → **Settings → Webhooks → Add Webhook**
2. **Target URL:** `https://reviewer.seu-dominio.com/webhook/<nome-do-client>`
3. **Secret:** valor de `webhook_secret` no `clients.json`
4. **Trigger On:** Pull Request

---

## 16. Monitoramento e manutenção

```bash
# Health check
curl https://reviewer.seu-dominio.com/healthz

# Logs em tempo real
sudo journalctl -u code-reviewer -f

# Uso de memória
ps aux | grep code-reviewer

# Tamanho do banco
du -sh /opt/code-reviewer/data/

# Reiniciar após alterar .env
sudo systemctl restart code-reviewer
```

### Deploy automático pelo GitHub Actions

Cada push no `main` do GitHub testa e compila o binário em runners hospedados. O deploy então **espera sua aprovação** no ambiente `production`. Aprovado, o binário vai por SSH para o servidor, onde a chave do CI só consegue executar `/usr/local/bin/code-reviewer-deploy`. O script confere o checksum, recusa o que não for o último commit do `main`, troca o binário e volta ao anterior se o `/healthz` não responder.

O servidor não precisa de token, código, Go nem Node para o deploy, então a seção 14 pode ser aplicada depois do primeiro deploy. A porta SSH precisa aceitar conexões da internet, por um host fora do proxy do Cloudflare.

| Usuário | Papel | Dono de | Pode |
|---|---|---|---|
| `cr-deploy` | Deploy, entra só pela chave do CI | Diretório da aplicação e binário | Reiniciar este serviço |
| `code-reviewer` | Roda o serviço, sem login | `.env`, `clients.json`, `data/` | Executar o binário |

**No servidor**, como `ubuntu`, confira primeiro o que o serviço usa hoje:

```bash
uname -m; . /etc/os-release; echo "$PRETTY_NAME"    # definem BUILD_RUNNER, abaixo
sudo systemctl cat code-reviewer                    # confira User, WorkingDirectory e EnvironmentFile
sudo grep -E '^(APP_PORT|DATABASE_PATH|WORKSPACE_DIR|CLIENTS_CONFIG_PATH|GITLAB_REPOS_PATH|SANDBOX_IMAGE)=' /opt/code-reviewer/.env
grep -hE '^(AllowUsers|AllowGroups)' /etc/ssh/sshd_config /etc/ssh/sshd_config.d/* 2>/dev/null   # se aparecer algo, inclua cr-deploy
command -v git curl flock
```

Antes de continuar, resolva o que o `grep` do `.env` mostrar:

- **Caminho fora de `data/`** (`DATABASE_PATH`, `WORKSPACE_DIR`) ou **`CLIENTS_CONFIG_PATH`**: dê acesso a `code-reviewer` nesses caminhos também.
- **`APP_PORT` diferente de 8080**: ajuste o `HEALTH_URL` no comando da chave, adiante.
- **`GITLAB_REPOS_PATH`**: `code-reviewer` precisa de leitura no `@hashed`; sem isso, a revisão agêntica cai para o clone pela rede.
- **`SANDBOX_IMAGE`**: a camada `execution` usa Docker, e o grupo `docker` equivale a root. Não adicione `code-reviewer` a esse grupo sem aceitar esse risco; sem o grupo, remova `SANDBOX_IMAGE` do `.env`, ou a IA pode ler o erro de acesso ao Docker como teste falho.

Depois, crie o usuário de deploy, ajuste as posses e instale o script. O serviço fica parado por alguns segundos. Este bloco assume que o serviço já roda como `code-reviewer`, com `nologin` e sem outros direitos:

```bash
systemctl show -p User code-reviewer                # User=code-reviewer
sudo adduser --disabled-password --gecos '' cr-deploy

# cr-deploy só pode reiniciar este serviço; confira com sudo -l -U code-reviewer se sobrou regra para o serviço
echo 'cr-deploy ALL=(root) NOPASSWD: /usr/bin/systemctl restart code-reviewer' > /tmp/cr-sudoers
sudo visudo -cf /tmp/cr-sudoers && sudo install -o root -g root -m 440 /tmp/cr-sudoers /etc/sudoers.d/code-reviewer-deploy
rm -f /tmp/cr-sudoers

sudo systemctl stop code-reviewer
cd /opt/code-reviewer
sudo chown cr-deploy:cr-deploy . code-reviewer
sudo chmod 755 . code-reviewer
sudo chown -R code-reviewer:code-reviewer data
sudo chmod -R u=rwX,go= data
sudo chown code-reviewer:code-reviewer .env clients.json
sudo chmod 600 .env clients.json
sudo systemctl start code-reviewer
systemctl show -p User,ActiveState code-reviewer
curl -fsS http://localhost:8080/healthz             # use o APP_PORT do .env, se for outro
cd -

# Instala o script fora do diretório da aplicação (troque <ref> pela branch; depois do merge, é main)
curl -fsSL -o /tmp/cr-deploy.sh https://raw.githubusercontent.com/PauloFH/code-reviewer/<ref>/deploy/code-reviewer-deploy.sh \
  && sudo install -o root -g root -m 755 /tmp/cr-deploy.sh /usr/local/bin/code-reviewer-deploy
rm -f /tmp/cr-deploy.sh
```

Para voltar atrás antes de instalar a chave: `sudo chown code-reviewer:code-reviewer /opt/code-reviewer /opt/code-reviewer/code-reviewer` e devolva a regra de `sudo` a `code-reviewer`.

**Na sua máquina**, gere a chave, crie o ambiente com aprovação e cadastre os segredos. A chave de deploy fica só no ambiente `production`, liberado depois da sua aprovação e só para o `main`:

```bash
HOST=<host-ssh-do-servidor>; PORT=22; R=PauloFH/code-reviewer
D=$(mktemp -d)
ssh-keygen -t ed25519 -N '' -C github-actions-deploy -f "$D/key"
ssh-keyscan -t ed25519 -p "$PORT" "$HOST" > "$D/known_hosts"
ssh-keygen -lf "$D/known_hosts"                     # confira com: sudo ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub

printf '{"reviewers":[{"type":"User","id":%s}],"deployment_branch_policy":{"protected_branches":false,"custom_branch_policies":true}}' "$(gh api user --jq .id)" \
  | gh api -X PUT "repos/$R/environments/production" --input -
gh api -X POST "repos/$R/environments/production/deployment-branch-policies" -f name=main

gh secret set DEPLOY_SSH_KEY -R "$R" -e production < "$D/key"
gh secret set DEPLOY_KNOWN_HOSTS -R "$R" -e production < "$D/known_hosts"
gh secret set DEPLOY_TARGET -R "$R" -e production -b "cr-deploy@$HOST"
gh variable set DEPLOY_PORT -R "$R" -e production -b "$PORT"
gh variable set BUILD_RUNNER -R "$R" -b ubuntu-24.04     # ubuntu-24.04-arm se uname -m for aarch64; acompanhe a versão do Ubuntu
openssl rand -base64 48 | gh secret set ARTIFACT_KEY -R "$R"
gh secret set CI_GIT_TOKEN -R "$R"                       # token de leitura do SDK; o valor é pedido no terminal

# PRs de fork precisam de aprovação para rodar; main sem force-push nem exclusão
gh api -X PUT "repos/$R/actions/permissions/fork-pr-contributor-approval" -f approval_policy=all_external_contributors
printf '{"required_status_checks":null,"enforce_admins":true,"required_pull_request_reviews":null,"restrictions":null,"allow_force_pushes":false,"allow_deletions":false}' \
  | gh api -X PUT "repos/$R/branches/main/protection" --input -
cat "$D/key.pub"
```

**De volta ao servidor**, autorize a chave só para o comando de deploy (cole a chave pública quando o `read` esperar). O arquivo é de root para `cr-deploy` não poder acrescentar outra chave:

```bash
read -r PUB
sudo install -d -m 755 /etc/ssh/authorized_keys.d
printf 'restrict,command="/usr/local/bin/code-reviewer-deploy" %s\n' "$PUB" \
  | sudo install -o root -g root -m 644 /dev/stdin /etc/ssh/authorized_keys.d/cr-deploy
printf 'Match User cr-deploy\n  AuthorizedKeysFile /etc/ssh/authorized_keys.d/cr-deploy\n' \
  | sudo install -o root -g root -m 644 /dev/stdin /etc/ssh/sshd_config.d/10-cr-deploy.conf
sudo sshd -t && sudo systemctl reload ssh
unset PUB
```

Se `APP_PORT` não for 8080, use `command="HEALTH_URL=http://localhost:<porta>/healthz /usr/local/bin/code-reviewer-deploy"`.

Depois, apague a chave local com `rm -rf "$D"`. O script é de root e não se atualiza pelo deploy: se `deploy/code-reviewer-deploy.sh` mudar, reinstale-o com o comando acima. Para voltar a uma versão anterior, use `git revert` no `main`: o script recusa commits superados.

**Corte:** desligue o Actions no Gitea antes de enviar o workflow novo, porque, sem `.gitea/workflows`, o Gitea usaria `.github/workflows`. Em **Settings → Repository → Actions** no Gitea, desmarque Actions; depois, apague o segredo `CI_GIT_TOKEN` de lá, pare e desregistre o `act_runner` e revogue o token antigo, que ficou no `~/.gitconfig` do usuário do runner. O primeiro deploy pelo GitHub é o teste real: se o binário novo não subir, o script volta ao anterior.

Limites:

- **Código aprovado:** ao aprovar um deploy, ele roda como `code-reviewer`, com acesso ao `.env` e ao banco; `retention-days: 1` faz o artefato expirar se a aprovação demorar mais de um dia, e aí basta refazer a execução. A barreira é a sua aprovação. Depois de um deploy malicioso, `git revert` não basta: troque os segredos.
- **Token do SDK e `ARTIFACT_KEY`:** continuam como segredos do CI, ao alcance do código do commit no job de build.
- **Chave de deploy:** quem a obtiver pode instalar qualquer binário, desde que informe o último commit do `main`. Ela só existe no ambiente `production`; o build não tem acesso a ela.

---

## Consumo de recursos esperado

| Recurso | Em repouso |
|---|---|
| RAM (Go) | ~40–80 MB |
| RAM (Nginx) | ~5–10 MB |
| CPU | ~0% (pico durante chamada à IA) |
| Disco — binário | ~25 MB |
| Disco — banco SQLite | cresce com o volume de reviews |

---

## Resolução de problemas

| Sintoma | Verificação |
|---|---|
| `502 Bad Gateway` | `sudo systemctl status code-reviewer` — serviço caiu |
| SSE desconectando | Confirme `proxy_buffering off` e `proxy_read_timeout 3600s` no Nginx |
| Webhook retorna 404 | Nome do client na URL não corresponde ao `name` no `clients.json` |
| Webhook retorna 401 | `webhook_secret` no `clients.json` difere do configurado na plataforma |
| Dashboard sem dados | Verifique `DASHBOARD_PASSWORD` e `DASHBOARD_JWT_SECRET` no `.env` |
| Cloudflare erro 525 | Mude SSL mode para **Flexible** (a VM não tem certificado próprio) |
