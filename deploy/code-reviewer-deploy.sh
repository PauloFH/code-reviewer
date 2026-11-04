#!/usr/bin/env bash
# Único comando da chave SSH do CI (forced command). Recebe "<commit> <sha256>" e o binário pela stdin.
# Instalado fora do diretório da aplicação, em /usr/local/bin, para o deploy não poder reescrevê-lo.
set -euo pipefail
export LC_ALL=C
umask 022

APP_DIR="${APP_DIR:-/opt/code-reviewer}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8080/healthz}"
REPO_URL="${REPO_URL:-https://github.com/PauloFH/code-reviewer.git}"
MAX_BYTES="${MAX_BYTES:-314572800}"

request="${SSH_ORIGINAL_COMMAND:-}"
if [[ ! "$request" =~ ^([0-9a-f]{40})\ ([0-9a-f]{64})$ ]]; then
  echo "erro: esperado '<commit> <sha256>'" >&2
  exit 2
fi
commit="${BASH_REMATCH[1]}"
checksum="${BASH_REMATCH[2]}"

cd "$APP_DIR"
exec 9>.deploy.lock
flock -w 600 9

# Rollback é por git revert: um CI atrasado ou re-run nunca regride a produção.
head_sha="$(git ls-remote "$REPO_URL" refs/heads/main | cut -f1)"
if [ -z "$head_sha" ]; then
  echo "erro: não foi possível ler o main em $REPO_URL" >&2
  exit 5
fi
if [ "$commit" != "$head_sha" ]; then
  echo "ignorado: $commit foi superado por $head_sha"
  exit 0
fi

new="$(mktemp code-reviewer.new.XXXXXX)"
trap 'rm -f "$new"' EXIT
head -c "$((MAX_BYTES + 1))" > "$new"
if [ "$(stat -c %s "$new")" -gt "$MAX_BYTES" ]; then
  echo "erro: binário acima de $MAX_BYTES bytes" >&2
  exit 3
fi
if [ "$(sha256sum "$new" | cut -d' ' -f1)" != "$checksum" ]; then
  echo "erro: checksum do binário não confere" >&2
  exit 4
fi
chmod 755 "$new"

if [ -f code-reviewer ]; then
  cp -p code-reviewer code-reviewer.prev
fi
mv -f "$new" code-reviewer

restart() { sudo -n systemctl restart code-reviewer; }
# Com Type=simple, o restart retorna sucesso mesmo se o binário novo cair no boot.
healthy() { curl -fsS -o /dev/null --retry 15 --retry-delay 2 --retry-all-errors --max-time 5 "$HEALTH_URL"; }

if restart && healthy; then
  echo "deploy concluído: $commit"
  exit 0
fi

echo "erro: o serviço não subiu com o binário novo; voltando ao anterior" >&2
if [ -f code-reviewer.prev ]; then
  mv -f code-reviewer.prev code-reviewer
  restart || echo "erro: o reinício com o binário anterior também falhou" >&2
fi
exit 6
