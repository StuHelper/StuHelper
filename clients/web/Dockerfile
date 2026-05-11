# Build stage
FROM node:24-alpine AS builder
WORKDIR /app
RUN corepack enable && corepack prepare pnpm@10.32.1 --activate
ARG VITE_API_URL=https://api.stuhelper.com
ARG VITE_SSO_URL=https://sso.stuhelper.com
ARG VITE_ADMIN_URL=https://stuhelper.com/admin/
ARG VITE_API_TIMEOUT_MS=15000
ENV VITE_API_URL=${VITE_API_URL}
ENV VITE_SSO_URL=${VITE_SSO_URL}
ENV VITE_ADMIN_URL=${VITE_ADMIN_URL}
ENV VITE_API_TIMEOUT_MS=${VITE_API_TIMEOUT_MS}
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml ./
COPY web/package.json ./web/
COPY shared/package.json ./shared/
COPY uniappx/package.json ./uniappx/
RUN pnpm install --frozen-lockfile
COPY . .
RUN pnpm run build:web

# Serve stage
FROM nginx:1.27-alpine
COPY --from=builder /app/web/dist /usr/share/nginx/html
COPY web/nginx.conf /etc/nginx/conf.d/default.conf

RUN chown -R nginx:nginx /usr/share/nginx/html && \
    chown -R nginx:nginx /var/cache/nginx && \
    chown -R nginx:nginx /var/log/nginx && \
    touch /var/run/nginx.pid && \
    chown -R nginx:nginx /var/run/nginx.pid

USER nginx
EXPOSE 80

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:80/ || exit 1
