#!/usr/bin/env bash
# Exercita code-reviewer-deploy.sh com git real e sudo/curl falsos.
set -euo pipefail

script="$(cd "$(dirname "$0")" && pwd)/code-reviewer-deploy.sh"
tmp="$(mktemp -d)"
trap 'rm -rf "$tmp"' EXIT
export GIT_CONFIG_GLOBAL=/dev/null GIT_CONFIG_SYSTEM=/dev/null
export GIT_AUTHOR_NAME=t GIT_AUTHOR_EMAIL=t@t GIT_COMMITTER_NAME=t GIT_COMMITTER_EMAIL=t@t

fail() { echo "FALHOU: $1" >&2; exit 1; }

log="$tmp/calls.log"
mkdir "$tmp/bin" "$tmp/app"
printf '#!/bin/sh\necho "sudo args=$*" >> "%s"\n' "$log" > "$tmp/bin/sudo"
printf '#!/bin/sh\necho "curl args=$*" >> "%s"\nexit "$(cat "%s/health" 2>/dev/null || echo 0)"\n' "$log" "$tmp" > "$tmp/bin/curl"
chmod +x "$tmp/bin/sudo" "$tmp/bin/curl"

git init -q --bare -b main "$tmp/remote.git"
git clone -q "$tmp/remote.git" "$tmp/dev" 2>/dev/null
git -C "$tmp/dev" switch -qc main 2>/dev/null || true
git -C "$tmp/dev" commit -q --allow-empty -m antigo
antigo="$(git -C "$tmp/dev" rev-parse HEAD)"
git -C "$tmp/dev" commit -q --allow-empty -m novo
git -C "$tmp/dev" push -q origin main
sha="$(git -C "$tmp/dev" rev-parse HEAD)"

app="$tmp/app"
printf 'binario-v1\n' > "$app/code-reviewer"
chmod 755 "$app/code-reviewer"
printf 'SEGREDO=1\n' > "$app/.env"
printf 'binario-v2\n' > "$tmp/v2"
sum="$(sha256sum "$tmp/v2" | cut -d' ' -f1)"

run() {
  SSH_ORIGINAL_COMMAND="$1" APP_DIR="$app" REPO_URL="$tmp/remote.git" PATH="$tmp/bin:$PATH" \
    bash "$script" < "${2:-$tmp/v2}" >/dev/null 2>&1
}
binary() { cat "$app/code-reviewer"; }
leftovers() { find "$app" -name 'code-reviewer.new*' | grep -c . || true; }

nl=$'\n'
for bad in '' main "$sha" "$sha abc" "$sha $sum extra" "$sha; rm -rf / $sum" '$(id)' "$sha $sum$nl" "$nl$sha $sum" "${sha%?}á $sum"; do
  if run "$bad"; then fail "aceitou entrada inválida: $bad"; fi
done
[ ! -e "$log" ] || fail "entrada inválida executou comandos"
[ "$(binary)" = binario-v1 ] || fail "entrada inválida trocou o binário"

# CI atrasado ou re-run de um commit superado não regride a produção.
run "$antigo $sum" || fail "commit superado deveria sair sem erro"
[ ! -e "$log" ] || fail "commit superado executou comandos"
[ "$(binary)" = binario-v1 ] || fail "commit superado trocou o binário"

# Transferência corrompida ou truncada não chega a substituir o binário.
printf 'truncado' > "$tmp/ruim"
if run "$sha $sum" "$tmp/ruim"; then fail "aceitou checksum diferente"; fi
[ ! -e "$log" ] || fail "checksum diferente executou comandos"
[ "$(binary)" = binario-v1 ] || fail "checksum diferente trocou o binário"
[ "$(leftovers)" = 0 ] || fail "ficou arquivo temporário depois de checksum diferente"

# Binário acima do limite é recusado sem encher o disco.
truncate -s 1M "$tmp/grande"
if MAX_BYTES=1024 run "$sha $(sha256sum "$tmp/grande" | cut -d' ' -f1)" "$tmp/grande"; then fail "aceitou binário acima do limite"; fi
[ "$(binary)" = binario-v1 ] || fail "binário acima do limite trocou o binário"

# Deploy válido.
run "$sha $sum" || fail "deploy válido falhou"
[ "$(binary)" = binario-v2 ] || fail "binário não foi trocado"
[ -x "$app/code-reviewer" ] || fail "binário novo sem permissão de execução"
[ "$(stat -c %a "$app/code-reviewer")" = 755 ] || fail "binário novo sem modo 755, cr-app não executaria"
[ "$(cat "$app/.env")" = 'SEGREDO=1' ] || fail ".env foi alterado"
grep -q '^sudo args=-n systemctl restart code-reviewer$' "$log" || fail "reinício ausente"
tail -n1 "$log" | grep -q '^curl .*/healthz' || fail "health check não foi o último passo"
[ "$(leftovers)" = 0 ] || fail "ficou arquivo temporário depois do deploy"

# Binário novo que não sobe: volta ao anterior e reinicia de novo.
: > "$log"
printf 'binario-v3\n' > "$tmp/v3"
printf '7' > "$tmp/health"
git -C "$tmp/dev" commit -q --allow-empty -m v3
git -C "$tmp/dev" push -q origin main
sha3="$(git -C "$tmp/dev" rev-parse HEAD)"
if run "$sha3 $(sha256sum "$tmp/v3" | cut -d' ' -f1)" "$tmp/v3"; then fail "deploy passou com health check falhando"; fi
[ "$(binary)" = binario-v2 ] || fail "não voltou ao binário anterior"
[ "$(grep -c '^sudo args=-n systemctl restart code-reviewer$' "$log")" = 2 ] || fail "não reiniciou depois de voltar"

# Conexão SSH caindo durante o health check de um binário ruim: o rollback ainda precisa acontecer.
: > "$log"
printf 'binario-v4\n' > "$tmp/v4"
git -C "$tmp/dev" commit -q --allow-empty -m v4
git -C "$tmp/dev" push -q origin main
sha4="$(git -C "$tmp/dev" rev-parse HEAD)"
SSH_ORIGINAL_COMMAND="$sha4 $(sha256sum "$tmp/v4" | cut -d' ' -f1)" APP_DIR="$app" REPO_URL="$tmp/remote.git" PATH="$tmp/bin:$PATH" \
  bash "$script" < "$tmp/v4" 2>&1 | true || true
[ "$(binary)" = binario-v2 ] || fail "conexão perdida impediu a volta ao binário anterior"
[ "$(grep -c '^sudo args=-n systemctl restart code-reviewer$' "$log")" = 2 ] || fail "conexão perdida impediu o reinício depois de voltar"

echo "deploy_test: ok"
