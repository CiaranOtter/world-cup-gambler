# ── Stage 1: build ──────────────────────────────────────────
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy module files first so dependency downloads are cached
COPY go.mod ./
RUN go mod download

# Copy source and build a fully static binary
COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o worldcup_gambler .

# ── Stage 2: runtime ────────────────────────────────────────
# scratch = zero OS overhead; only the binary + static files
FROM scratch

WORKDIR /app

# TLS root certificates (needed to call https://worldcup26.ir)
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Binary
COPY --from=builder /build/worldcup_gambler .

# Static web assets
COPY static/ ./static/

# User data directory — mount a volume here to persist profiles
# across container restarts (see docker-compose.yml)
VOLUME ["/app/data"]

EXPOSE 8080

ENTRYPOINT ["/app/worldcup_gambler"]