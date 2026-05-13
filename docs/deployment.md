# Guia de Implantação

Implantação do Code Reviewer em servidor Linux sem Docker. Recomendado para VMs com recursos limitados (1 GB RAM ou menos).

**Testado em:** Canonical Ubuntu 24.04 Minimal · 1 GHz · 1 GB RAM  
**Runtime:** binário Go autocontido + systemd

---

## Pré-requisitos

- Ubuntu 24.04 (ou derivado)
- Acesso SSH com `sudo`
- IP público ou túnel (Cloudflare Tunnel) para receber webhooks

---

## 1. Preparar o servidor

```bash
sudo apt update && sudo apt upgrade -y
sudo apt install -y git curl ca-certificates gcc musl-dev openssl
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

# aarch64 (ARM64)
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
# saída em internal/transport/http/frontend/dist (usado pelo embed Go)
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

# API de IA compatível com chat/completions (Claude, OpenAI, etc.)
AI_BASE_URL=https://sua-api-de-ia.com
AI_TOKEN=seu-token-aqui
AI_MODEL=claude-sonnet-4
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

Mantenha apenas as plataformas que você usa. Exemplo mínimo com um client GitHub:

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

Na primeira inicialização, se a tabela `clients` estiver vazia, os clients deste arquivo são importados automaticamente para o banco.

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

---

## 11. Firewall

```bash
sudo ufw allow 8080/tcp
sudo ufw enable
sudo ufw status
```

Dashboard disponível em `http://<IP-DA-VM>:8080`.

---

## 12. Verificar funcionamento

```bash
# Liveness
curl http://localhost:8080/healthz

# Logs em tempo real
sudo journalctl -u code-reviewer -f
```

---

## 13. Liberar espaço após o build

Go e Node não são necessários em runtime — o binário é autocontido.

```bash
sudo apt remove -y nodejs
sudo rm -rf /usr/local/go ~/go
sudo apt autoremove -y
```

---

## 14. Expor para a internet (webhooks)

A VM precisa ser acessível publicamente para receber webhooks das plataformas Git.

### Opção A — IP público direto

Configure o DNS ou use o IP diretamente na URL do webhook.

### Opção B — Cloudflare Tunnel (sem IP público)

```bash
# Instalar cloudflared
# x86_64
curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-amd64.deb \
  -o cloudflared.deb
# aarch64
# curl -L https://github.com/cloudflare/cloudflared/releases/latest/download/cloudflared-linux-arm64.deb \
#   -o cloudflared.deb

sudo dpkg -i cloudflared.deb

# Túnel temporário (testes)
cloudflared tunnel --url http://localhost:8080
# Anote a URL gerada: https://xxxx.trycloudflare.com
```

URL do webhook: `https://xxxx.trycloudflare.com/webhook/<nome-do-client>`

Para túnel permanente, consulte a [documentação do Cloudflare Tunnel](https://developers.cloudflare.com/cloudflare-one/connections/connect-networks/).

---

## 15. Configurar webhooks nas plataformas

### GitHub

1. Repositório → **Settings → Webhooks → Add webhook**
2. **Payload URL:** `https://seu-dominio.com/webhook/<nome-do-client>`
3. **Content type:** `application/json`
4. **Secret:** valor de `webhook_secret` no `clients.json`
5. **Events:** selecione **Pull requests** apenas

### GitLab

1. Projeto → **Settings → Webhooks**
2. **URL:** `https://seu-dominio.com/webhook/<nome-do-client>`
3. **Secret token:** valor de `webhook_secret` no `clients.json`
4. **Trigger:** marque apenas **Merge request events**

### Gitea

1. Repositório → **Settings → Webhooks → Add Webhook**
2. **Target URL:** `https://seu-dominio.com/webhook/<nome-do-client>`
3. **Secret:** valor de `webhook_secret` no `clients.json`
4. **Trigger On:** Pull Request

---

## Monitoramento e manutenção

```bash
# Uso de memória do processo
ps aux | grep code-reviewer

# Tamanho do banco de dados
du -sh /opt/code-reviewer/data/

# Últimas 100 linhas de log
sudo journalctl -u code-reviewer -n 100

# Reiniciar após alteração no .env
sudo systemctl restart code-reviewer

# Atualizar o binário
cd /opt/code-reviewer
git pull
cd web && npm ci && npm run build && cd ..
CGO_ENABLED=1 go build -ldflags="-s -w" -o code-reviewer ./cmd/server
sudo systemctl restart code-reviewer
```

---

## Consumo de recursos esperado

| Recurso | Em repouso |
|---|---|
| RAM | ~40–80 MB |
| CPU | ~0% (pico durante chamada à IA) |
| Disco — binário | ~25 MB |
| Disco — banco SQLite | cresce com o volume de reviews |
