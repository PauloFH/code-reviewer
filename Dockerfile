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
FROM golang:1.25-alpine AS go-builder

# CGO is required by go-sqlite3
RUN apk add --no-cache gcc musl-dev

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

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
