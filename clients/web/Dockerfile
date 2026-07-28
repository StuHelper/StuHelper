ARG NODE_IMAGE=node:24.18.0-alpine@sha256:a0b9bf06e4e6193cf7a0f58816cc935ff8c2a908f81e6f1a95432d679c54fbfd
ARG NGINX_IMAGE=nginx:1.30.4-alpine@sha256:97d490c12ba55b4946b01546d1c3ed324e8d41ab1c9fcb2a616aa470620e5b46

# Build stage
FROM ${NODE_IMAGE} AS builder
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.32.1 --activate
ARG VITE_API_URL=/api
ARG VITE_SSO_URL=https://sso.stuhelper.com
ARG VITE_WEB_URL=https://stuhelper.com
ARG VITE_ADMIN_URL=https://stuhelper.com/admin/
ARG VITE_API_TIMEOUT_MS=15000
ARG VITE_QQ_BOT_ENTRY=""
ARG VITE_QQ_BIND_COMMAND="绑定"
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_SSO_URL=${VITE_SSO_URL}
ENV VITE_WEB_URL=${VITE_WEB_URL}
ENV VITE_ADMIN_URL=${VITE_ADMIN_URL}
ENV VITE_API_TIMEOUT_MS=${VITE_API_TIMEOUT_MS}
ENV VITE_QQ_BOT_ENTRY=${VITE_QQ_BOT_ENTRY}
ENV VITE_QQ_BIND_COMMAND=${VITE_QQ_BIND_COMMAND}
COPY clients/package.json clients/pnpm-lock.yaml clients/pnpm-workspace.yaml ./
COPY clients/web/package.json ./web/
COPY clients/shared/package.json ./shared/
COPY clients/uniappx/package.json ./uniappx/
RUN pnpm install --frozen-lockfile
COPY clients/scripts ./scripts
COPY clients/shared ./shared
COPY clients/web ./web
COPY clients/uniappx ./uniappx
COPY server/internal/pkg/capability/catalog.go /server/internal/pkg/capability/catalog.go
RUN pnpm run build:web

# Serve stage
FROM ${NGINX_IMAGE}
COPY --from=builder /app/web/dist /usr/share/nginx/html
COPY --from=builder /app/web/nginx.conf /etc/nginx/conf.d/default.conf

RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid && \
    sed -i 's#^pid .*#pid /tmp/nginx.pid;#' /etc/nginx/nginx.conf

USER nginx
EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:80/ || exit 1
