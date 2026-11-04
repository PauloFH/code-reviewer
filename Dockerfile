# ─────────────────────────────────────────────
# Stage 1: build the Vue 3 frontend
# ─────────────────────────────────────────────
FROM node:22-alpine AS frontend-builder

WORKDIR /web
COPY web/package*.json ./
RUN npm ci --prefer-offline

COPY web/ ./
# vite.config.ts sets outDir to ../internal/transport/http/frontend/dist
RUN npm run build

# ─────────────────────────────────────────────
# Stage 2: build the Go binary
# ─────────────────────────────────────────────
# Tag fixa: a imagem define GOTOOLCHAIN=local, então a linha `toolchain` do go.mod
# não baixa Go novo; uma tag flutuante em cache poderia compilar com stdlib vulnerável.
FROM golang:1.26.10-alpine AS go-builder

# CGO is required by go-sqlite3; git is required to fetch the private SDK module
RUN apk add --no-cache gcc musl-dev git

# The SDK lives on the private Gitea: skip the public proxy/checksum DB for it
ENV GOPRIVATE=git.pauloholanda.com.br/*

WORKDIR /app

COPY go.mod go.sum ./
# The token comes from a BuildKit secret (docker build --secret id=git_token,...),
# mounted only for this command. A credential helper scoped to the Gitea host
# (reset first, so no `store` helper can persist it) reads the file when git asks:
# the token is never in a command line/ps, file, ENV, ARG or image layer.
RUN --mount=type=secret,id=git_token \
    if [ ! -s /run/secrets/git_token ]; then \
      echo "ERROR: secret 'git_token' ausente ou vazio. Use: docker build --secret id=git_token,env=CI_GIT_TOKEN . (ou docker compose build com CI_GIT_TOKEN exportado)" >&2; \
      exit 1; \
    fi; \
    GIT_CONFIG_COUNT=2 \
    GIT_CONFIG_KEY_0='credential.https://git.pauloholanda.com.br.helper' \
    GIT_CONFIG_VALUE_0='' \
    GIT_CONFIG_KEY_1='credential.https://git.pauloholanda.com.br.helper' \
    GIT_CONFIG_VALUE_1='!f() { test "$1" = get && printf "username=oauth2\npassword=%s\n" "$(cat /run/secrets/git_token)"; }; f' \
    go mod download

COPY . .
# Copy the compiled frontend into the embed path
COPY --from=frontend-builder /web/../internal/transport/http/frontend/dist \
     ./internal/transport/http/frontend/dist

RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o code-reviewer ./cmd/server

# ─────────────────────────────────────────────
# Stage 3: minimal runtime image
# ─────────────────────────────────────────────
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=go-builder /app/code-reviewer .

RUN mkdir -p /app/data

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=5s \
  CMD wget -qO- http://localhost:8080/healthz || exit 1

CMD ["./code-reviewer"]
