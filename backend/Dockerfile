// Multi-stage Dockerfile for Radiant Norma backend.
//
// Validação 21 (F21.7): docker-compose.yml referencia este arquivo
// para os perfis `api` e `worker`. Sem isso, build falha.
//
// Build:
//
//	docker build -t radiant-norma -f backend/Dockerfile .
//
// Run (Postgres primeiro, ver docker-compose.yml):
//
//	docker run --rm -p 8080:8080 \
//	  -e DATABASE_URL="postgres://radiant:radiant@postgres:5432/radiant?sslmode=disable" \
//	  -e RADIANT_NORMA_ADMIN_TOKEN="..." \
//	  radiant-norma /app/api

# ============ Stage 1: build ============
FROM golang:1.25-alpine AS build

# Build deps para CGO (alguns drivers Go podem precisar; usamos pure-Go, então
# mas instalamos por garantia).
RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copia go.mod primeiro para cache de layers.
COPY go.mod go.sum ./
RUN go mod download

# Argumento VERSION para bake da versão em build-time (Validação 27 F27.2).
# Override no docker build: --build-arg VERSION=2.0.1+commit123
ARG VERSION=2.0.0
ENV VERSION=${VERSION}

# Copia o resto e build os 4 binários.
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=${VERSION}'" \
    -o /out/api ./cmd/api
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=${VERSION}'" \
    -o /out/worker ./cmd/worker
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=${VERSION}'" \
    -o /out/radar ./cmd/radar
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X 'github.com/fortvna/radiant-norma/backend/internal/version.Version=${VERSION}'" \
    -o /out/verify ./cmd/_verify

# ============ Stage 2: runtime ============
FROM alpine:3.19

# ca-certificates para TLS (HTTPS a BACEN/etc).
# tini para signal forwarding (PID 1 limpo).
RUN apk add --no-cache ca-certificates tini tzdata
ENV TZ=UTC

# Non-root user.
RUN addgroup -S radiant && adduser -S radiant -G radiant

WORKDIR /app
COPY --from=build /out/api /app/api
COPY --from=build /out/worker /app/worker
COPY --from=build /out/radar /app/radar
COPY --from=build /out/verify /app/verify

# Volumes para DB local (SQLite) e logs.
RUN mkdir -p /data /var/log/radiant && chown radiant:radiant /data /var/log/radiant
VOLUME ["/data", "/var/log/radiant"]
USER radiant

# Default = API; override com `command: ["/app/worker"]` ou ["/app/radar"].
ENTRYPOINT ["/sbin/tini", "--"]
CMD ["/app/api"]

# API listens em :8080 (override via RADIANT_ADDR).
EXPOSE 8080

# Health check (configura se você subir com --profile prod).
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD wget --quiet --tries=1 --spider http://localhost:8080/healthz || exit 1
