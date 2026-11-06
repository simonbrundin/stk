# syntax=docker/dockerfile:1
#
# Go API - Multi-stage build with distroless runtime
#
# Note: CLI verktyg kompileras lokalt, inte i Docker.
# Se README.md för hur man bygger CLI lokalt.
#
# Best practices applied:
# - Multi-stage build (builder/runtime separation)
# - BuildKit cache mounts for faster rebuilds
# - Distroless runtime (minimal attack surface)
# - Non-root user
# - HEALTHCHECK
# - Cross-platform build support
#

# =============================================================================
# Stage 1: Generate OpenAPI docs with swag
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS swag

ENV GOTOOLCHAIN=auto

RUN apk add --no-cache go git && \
    go install github.com/swaggo/swag/cmd/swag@v1.16.6 && \
    mv $(go env GOPATH)/bin/swag /usr/local/bin/

WORKDIR /build
COPY . .
RUN swag init -g cmd/api/main.go -o cmd/api/docs --parseFuncBody --quiet

# =============================================================================
# Stage 2: Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM golang:1.27-alpine AS builder

ENV GOTOOLCHAIN=auto \
    CGO_ENABLED=0

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

WORKDIR /build

# Copy go mod files first (layer caching)
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod \
    go mod download

# Build args for cross-compilation
ARG TARGETOS
ARG TARGETARCH

# Copy source and build
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    go build -ldflags="-s -w" -trimpath -o /api ./cmd/api

# =============================================================================
# Stage 2: Runtime - distroless (minimal attack surface)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

LABEL org.opencontainers.image.title="Go API" \
      org.opencontainers.image.source="https://github.com/simon/repo-template" \
      org.opencontainers.image.description="Minimal Go API server"

# Copy binary and OpenAPI docs
COPY --from=builder /api /api
COPY --from=swag /build/cmd/api/docs /app/docs

ENV DOCS_PATH=/app/docs/swagger.json

# Non-root (distroless nonroot user is UID 65532)
USER nonroot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["wget", "-q", "--spider", "http://localhost:8080/health"] || exit 1

ENTRYPOINT ["/api"]
