#!/usr/bin/env bash
# Único comando da chave SSH do CI (forced command em authorized_keys).
# Instalado fora do checkout, em /usr/local/bin, para o deploy não poder reescrevê-lo.
set -euo pipefail
export LC_ALL=C
umask 022

APP_DIR="${APP_DIR:-/opt/code-reviewer}"
TOKEN_FILE="${TOKEN_FILE:-/etc/code-reviewer/git-token}"
HEALTH_URL="${HEALTH_URL:-http://localhost:8080/healthz}"
export PATH="${GO_BIN_DIR:-/usr/local/go/bin}:$PATH"

# O CI envia só o SHA testado; qualquer outra coisa é recusada antes de agir.
sha="${SSH_ORIGINAL_COMMAND:-}"
if [[ ! "$sha" =~ ^[0-9a-f]{40}$ ]]; then
  echo "erro: esperado um SHA de 40 caracteres" >&2
  exit 2
fi

cd "$APP_DIR"
exec 9>.git/deploy.lock
flock -w 600 9

git fetch origin main
head_sha="$(git rev-parse FETCH_HEAD)"
if [ "$sha" != "$head_sha" ]; then
  # Rollback é por git revert: um CI atrasado ou re-run nunca regride a produção.
  if git cat-file -e "$sha^{commit}" 2>/dev/null && git merge-base --is-ancestor "$sha" "$head_sha"; then
    echo "ignorado: $sha foi superado por $head_sha"
    exit 0
  fi
  echo "erro: $sha não pertence ao main" >&2
  exit 3
fi

# Falha antes de alterar o checkout. Fica fora do ambiente do npm, mas o código
# do build roda com este usuário e pode ler o arquivo.
token="$(cat "$TOKEN_FILE")"
if [ -z "$token" ]; then
  echo "erro: $TOKEN_FILE está vazio" >&2
  exit 4
fi

# Sem pull: o servidor espelha o main, e o build altera o dist/index.html versionado.
git reset --hard "$sha"

(cd web && npm ci --ignore-scripts && npm run build)

# Helper com reset: o token não vai para ps nem para um credential.helper=store.
CI_GIT_TOKEN="$token" \
GOPRIVATE='git.pauloholanda.com.br/*' \
GIT_CONFIG_COUNT=2 \
GIT_CONFIG_KEY_0='credential.https://git.pauloholanda.com.br.helper' \
GIT_CONFIG_VALUE_0='' \
GIT_CONFIG_KEY_1='credential.https://git.pauloholanda.com.br.helper' \
GIT_CONFIG_VALUE_1='!f() { test "$1" = get && printf "username=oauth2\npassword=%s\n" "$CI_GIT_TOKEN"; }; f' \
CGO_ENABLED=1 go build -ldflags="-s -w" -o code-reviewer ./cmd/server

sudo -n systemctl restart code-reviewer
# Com Type=simple, o restart retorna sucesso mesmo se o binário novo cair no boot.
curl -fsS -o /dev/null --retry 15 --retry-delay 2 --retry-all-errors --max-time 5 "$HEALTH_URL"
echo "deploy concluído: $sha"
