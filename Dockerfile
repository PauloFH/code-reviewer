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
FROM golang:1.26-alpine AS go-builder

# CGO is required by go-sqlite3; git is required to fetch the private SDK module
RUN apk add --no-cache gcc musl-dev git

# The SDK lives on the private Gitea: skip the public proxy/checksum DB for it
ENV GOPRIVATE=git.pauloholanda.com.br/*

WORKDIR /app

COPY go.mod go.sum ./
# The token comes from a BuildKit secret (docker build --secret id=git_token,...),
# is mounted only for this command and injected via GIT_CONFIG_* env vars of this
# single process: never written to a file, ENV, ARG or image layer.
RUN --mount=type=secret,id=git_token \
    if [ ! -s /run/secrets/git_token ]; then \
      echo "ERROR: secret 'git_token' ausente ou vazio. Use: docker build --secret id=git_token,env=CI_GIT_TOKEN ." >&2; \
      exit 1; \
    fi; \
    GIT_CONFIG_COUNT=1 \
    GIT_CONFIG_KEY_0="url.https://oauth2:$(cat /run/secrets/git_token)@git.pauloholanda.com.br/.insteadOf" \
    GIT_CONFIG_VALUE_0="https://git.pauloholanda.com.br/" \
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
