#!/bin/bash
set -e

cd /opt/code-reviewer

# Garantir acesso ao módulo privado
current_goprivate=$(go env GOPRIVATE)
if [[ "$current_goprivate" != *"git.pauloholanda.com.br"* ]]; then
    go env -w 'GOPRIVATE=git.pauloholanda.com.br/*'
fi

if ! git config --global --get url."ssh://git@ssh.git.pauloholanda.com.br:2222/".insteadOf > /dev/null 2>&1; then
    git config --global url."ssh://git@ssh.git.pauloholanda.com.br:2222/".insteadOf "https://git.pauloholanda.com.br/"
fi

git pull

go mod tidy

cd web
npm ci
npm run build

cd /opt/code-reviewer
CGO_ENABLED=1 go build -ldflags="-s -w" -o code-reviewer ./cmd/server

sudo systemctl restart code-reviewer
