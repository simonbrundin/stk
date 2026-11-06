# syntax=docker/dockerfile:1
#
# Rust API - Multi-stage build with distroless runtime
#
# Stages:
#   - builder:  Kompilerar statisk musl-binär (rustup target add musl)
#   - runtime:  distroless (minimal attack surface), non-root
#
# Notera: ingen OpenAPI-fil genereras i byggsteget – specen är code-first
# och genereras vid runtime av utoipa (se src/openapi.rs).
#
# Best practices:
# - Multi-stage build (builder/runtime separation)
# - BuildKit cache mounts for faster rebuilds
# - Fully static binary => körs i distroless/static
# - Non-root user
# - HEALTHCHECK via samma binär (distroless saknar curl/wget)
#

# =============================================================================
# Stage 1: Builder
# =============================================================================
FROM --platform=$BUILDPLATFORM rust:1.99-slim-bookworm AS builder

ARG TARGETARCH
ENV CARGO_TERM_COLOR=always

WORKDIR /build

# Cache dependencies: kopiera manifesten först (placeholder-src så att cargo
# kan parsa manifesten utan den riktiga koden)
COPY Cargo.toml Cargo.lock ./
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    mkdir -p src && \
    printf 'fn main() {}\n' > src/main.rs && \
    : > src/lib.rs && \
    cargo fetch --locked && \
    rm -rf src

COPY src ./src

# Bygg statisk binär för arkitekturen som byggs
RUN --mount=type=cache,target=/usr/local/cargo/registry \
    --mount=type=cache,target=/build/target \
    set -eux; \
    case "${TARGETARCH:-amd64}" in \
      amd64) target=x86_64-unknown-linux-musl ;; \
      arm64) target=aarch64-unknown-linux-musl ;; \
      *) echo "unsupported TARGETARCH: ${TARGETARCH}" >&2; exit 1 ;; \
    esac; \
    rustup target add "$target"; \
    cargo build --release --locked --target "$target"; \
    mkdir -p /out; \
    cp "/build/target/$target/release/repo-template-api" /out/repo-template-api

# =============================================================================
# Stage 2: Runtime - distroless (minimal attack surface)
# =============================================================================
FROM gcr.io/distroless/static-debian12:nonroot AS runtime

LABEL org.opencontainers.image.title="Rust API" \
      org.opencontainers.image.source="https://github.com/simon/repo-template" \
      org.opencontainers.image.description="REST-API byggd med Axum, Tokio och utoipa"

COPY --from=builder /out/repo-template-api /api

ENV PORT=8080 \
    APP_VERSION=dev \
    RUST_LOG=info

# Distroless nonroot använder UID 65532
USER nonroot

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/api", "healthcheck"]

ENTRYPOINT ["/api"]
