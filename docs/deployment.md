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
sudo nano /etc/systemd/system/code-reviewer.service
```

```ini
[Unit]
Description=Code Reviewer
After=network.target

[Service]
Type=simple
User=ubuntu
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

    # Cloudflare envia o IP real do cliente neste header
    real_ip_header    CF-Connecting-IP;
    set_real_ip_from  0.0.0.0/0;

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

As portas 80 e 443 já estão abertas na security list da rede.  
Configure o `ufw` interno para consistência:

```bash
sudo ufw allow 22/tcp
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
sudo ufw enable
sudo ufw status
```

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

### Atualizar o binário

```bash
cd /opt/code-reviewer
git pull

# Reinstalar Node temporariamente para rebuild do frontend
curl -fsSL https://deb.nodesource.com/setup_22.x | sudo -E bash -
sudo apt install -y nodejs
source /etc/profile.d/go.sh

cd web && npm ci && npm run build && cd ..
CGO_ENABLED=1 go build -ldflags="-s -w" -o code-reviewer ./cmd/server
sudo systemctl restart code-reviewer

# Remover Node novamente
sudo apt remove -y nodejs && sudo apt autoremove -y
```

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
