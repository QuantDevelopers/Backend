# ==============================================================================
# QuantBackend Dockerfile (Go + Echo + Wire + GORM)
# ==============================================================================
FROM golang:1.27-alpine AS builder

WORKDIR /app

ARG GOPROXY=https://goproxy.io,https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}

# Cache Go module downloads
COPY go.mod go.sum ./
RUN go mod download

# Copy full source and build static binary
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/quant-server ./cmd/app

# ==============================================================================
# Minimal Runtime Image (Alpine with ArvanCloud Mirror)
# ==============================================================================
FROM alpine:3.21 AS runtime

ARG ALPINE_MIRROR=http://mirror.arvancloud.ir/alpine

WORKDIR /app

# Replace Alpine APK repositories with ArvanCloud mirror (override ALPINE_MIRROR="" to use upstream)
RUN if [ -n "$ALPINE_MIRROR" ]; then \
      ALPINE_VER="v$(cut -d. -f1,2 /etc/alpine-release)" && \
      sed -i 's/^/#/' /etc/apk/repositories && \
      printf '%s/%s/main\n%s/%s/community\n' "$ALPINE_MIRROR" "$ALPINE_VER" "$ALPINE_MIRROR" "$ALPINE_VER" >> /etc/apk/repositories; \
    fi && \
    apk update && \
    apk add --no-cache ca-certificates curl tzdata && \
    adduser -D -u 10001 appuser && \
    mkdir -p /app/data/raw && \
    chown -R appuser:appuser /app

COPY --from=builder /app/quant-server /app/quant-server

USER appuser

EXPOSE 8000

HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD curl -f http://localhost:8000/health || exit 1

CMD ["/app/quant-server"]
