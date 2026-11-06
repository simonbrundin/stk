# syntax=docker/dockerfile:1
#
# Nuxt SPA + Static File Server
#
# Architecture:
#   Browser → Nuxt SPA → Go API → PostgreSQL
#
# Stages:
#   - base: Shared dependencies
#   - dev: Development server (bun)
#   - builder: Builds static SPA files
#   - runtime: nginx serves static files
#
# Usage:
#   Development: docker build --target dev -t nuxt-dev . && docker run -it -p 3000:3000 nuxt-dev
#   Production:  docker build -t nuxt-prod . && docker run -p 3000:80 nuxt-prod
#

# =============================================================================
# Base: Shared dependencies (cached layer)
# =============================================================================
FROM oven/bun:1-alpine AS base

WORKDIR /app

# Install dependencies once (shared by dev and builder)
COPY package.json bun.lock ./
RUN --mount=type=cache,target=/home/bun/.cache/install \
    bun install --frozen-lockfile

# =============================================================================
# Dev Stage: Full development environment
# =============================================================================
FROM base AS dev

# Copy source
COPY . .

EXPOSE 3000

CMD ["bun", "run", "dev", "--", "--host", "0.0.0.0"]

# =============================================================================
# Builder Stage: Build static SPA files
# =============================================================================
FROM base AS builder

# Copy source and build
COPY . .
RUN bun run generate

# =============================================================================
# Runtime Stage: Serve static files with nginx
# =============================================================================
FROM nginx:alpine AS production

# Copy generated static files
COPY --from=builder /app/.output/public /usr/share/nginx/html

# Copy nginx config for SPA fallback
COPY --chown=nginx:nginx <<'EOF' /etc/nginx/conf.d/default.conf
server {
    listen 80;
    server_name _;
    root /usr/share/nginx/html;
    index index.html;

    # Keep port in redirects
    port_in_redirect on;
    server_name_in_redirect off;

    # Gzip compression
    gzip on;
    gzip_types text/plain text/css application/json application/javascript text/xml application/xml;

    # Proxy API requests to Go API
    location /api/ {
        proxy_pass http://go-api-service:8080/;
        proxy_http_version 1.1;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    # SPA routes - don't add trailing slash, serve index.html
    location ~ ^/[a-zA-Z0-9_-]+$ {
        try_files $uri /index.html;
    }

    location / {
        try_files $uri $uri/ /index.html?$args;
    }

    # Cache static assets
    location /_nuxt/ {
        expires 1y;
        add_header Cache-Control "public, immutable";
    }

    # No cache for index.html
    location = /index.html {
        add_header Cache-Control "no-cache, no-store, must-revalidate";
    }
}
EOF

EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["wget", "-q", "--spider", "http://localhost/index.html"] || exit 1

CMD ["nginx", "-g", "daemon off;"]
