#!/usr/bin/env bash
# Exercita code-reviewer-deploy.sh com git real e npm/go/sudo/curl falsos.
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/code-reviewer-deploy.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t

fail() { echo "FALHOU: $1" >&2; exit 1; }

log="$tmp/calls.log"
mkdir "$tmp/bin"
for cmd in npm go sudo curl; do
  printf '#!/bin/sh\necho "%s args=$* token=${CI_GIT_TOKEN:-vazio} private=${GOPRIVATE:-vazio}" >> "%s"\n' "$cmd" "$log" > "$tmp/bin/$cmd"
  chmod +x "$tmp/bin/$cmd"
done
printf 'token-secreto\n' > "$tmp/git-token"

git init -q --bare -b main "$tmp/remote.git"
git clone -q "$tmp/remote.git" "$tmp/dev" 2>/dev/null
git -C "$tmp/dev" switch -qc main 2>/dev/null || true
mkdir "$tmp/dev/web"
printf 'v1\n' > "$tmp/dev/web/index.html"
git -C "$tmp/dev" add .
git -C "$tmp/dev" commit -qm base
git -C "$tmp/dev" push -q origin main
git clone -q "$tmp/remote.git" "$tmp/app"
printf '{}\n' > "$tmp/app/.env"

# Histórico divergente, como depois de um force-push no main.
git -C "$tmp/dev" commit -q --amend -m reescrito
antigo="$(git -C "$tmp/dev" rev-parse HEAD)"
printf 'v2\n' > "$tmp/dev/web/index.html"
git -C "$tmp/dev" add .
git -C "$tmp/dev" commit -qm novo
git -C "$tmp/dev" push -qf origin main
sha="$(git -C "$tmp/dev" rev-parse HEAD)"

git -C "$tmp/dev" switch -qc fora-do-main
git -C "$tmp/dev" commit -q --allow-empty -m lateral
git -C "$tmp/dev" push -q origin fora-do-main
outro="$(git -C "$tmp/dev" rev-parse HEAD)"

run() {
  SSH_ORIGINAL_COMMAND="$1" APP_DIR="$tmp/app" TOKEN_FILE="${TOKEN:-$tmp/git-token}" \
    GO_BIN_DIR="$tmp/bin" PATH="$tmp/bin:$PATH" bash "$script" >/dev/null 2>&1
}

nl=$'\n'
for bad in '' 'main' 'abc123' "$sha; rm -rf /" '$(id)' "$sha$nl" "$nl$sha" "${sha%?}á"; do
  if run "$bad"; then fail "aceitou entrada inválida: $bad"; fi
done
[ ! -e "$log" ] || fail "entrada inválida executou comandos"

if run "$outro"; then fail "aceitou commit fora do main"; fi
[ ! -e "$log" ] || fail "commit fora do main executou comandos"

# CI atrasado ou re-run de um commit superado não regride a produção.
run "$antigo" || fail "commit superado deveria sair sem erro"
[ ! -e "$log" ] || fail "commit superado executou comandos"
[ "$(git -C "$tmp/app" rev-parse HEAD)" != "$antigo" ] || fail "commit superado alterou o checkout"

before="$(git -C "$tmp/app" rev-parse HEAD)"
if TOKEN="$tmp/inexistente" run "$sha"; then fail "aceitou token ausente"; fi
[ "$(git -C "$tmp/app" rev-parse HEAD)" = "$before" ] || fail "token ausente alterou o checkout"
[ ! -e "$log" ] || fail "token ausente executou comandos"

printf 'alterado-pelo-build\n' > "$tmp/app/web/index.html"
run "$sha" || fail "deploy válido falhou"

[ "$(git -C "$tmp/app" rev-parse HEAD)" = "$sha" ] || fail "HEAD não foi para o commit pedido"
[ "$(cat "$tmp/app/web/index.html")" = "v2" ] || fail "alteração local não foi descartada"
[ -f "$tmp/app/.env" ] || fail ".env não rastreado foi apagado"
grep -q '^npm args=ci --ignore-scripts ' "$log" || fail "npm ci sem --ignore-scripts"
if grep '^npm ' "$log" | grep -qv 'token=vazio'; then fail "npm recebeu o token no ambiente"; fi
if grep -E '^(sudo|curl) ' "$log" | grep -q 'token-secreto'; then fail "sudo ou curl recebeu o token"; fi
grep -q '^go args=build .*token=token-secreto private=git.pauloholanda.com.br/\*' "$log" || fail "go build sem token ou GOPRIVATE"
sed -n '/^sudo /,$p' "$log" | head -1 | grep -q '^sudo args=-n systemctl restart code-reviewer' || fail "reinício ausente"
tail -n1 "$log" | grep -q '^curl .*/healthz' || fail "health check não foi o último passo"

# Serviço que não sobe depois do reinício precisa falhar o deploy.
printf '#!/bin/sh\nexit 7\n' > "$tmp/bin/curl"
if run "$sha"; then fail "deploy passou com health check falhando"; fi

echo "deploy_test: ok"
