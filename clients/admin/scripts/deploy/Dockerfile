ARG NODE_IMAGE=node:24.18.0-bookworm-slim@sha256:6f7b03f7c2c8e2e784dcf9295400527b9b1270fd37b7e9a7285cf83b6951452d
ARG NGINX_IMAGE=nginx:1.30.4-alpine@sha256:97d490c12ba55b4946b01546d1c3ed324e8d41ab1c9fcb2a616aa470620e5b46

FROM ${NODE_IMAGE} AS builder

# --max-old-space-size
ENV PNPM_HOME="/pnpm"
ENV PATH="$PNPM_HOME:$PATH"
ENV NODE_OPTIONS=--max-old-space-size=8192
ENV TZ=Asia/Shanghai
ENV CI=true
ENV TURBO_TELEMETRY_DISABLED=1
ARG VITE_GLOB_API_URL=/api/v1
ARG VITE_BASE=/admin/
ENV VITE_GLOB_API_URL=${VITE_GLOB_API_URL}
ENV VITE_BASE=${VITE_BASE}

RUN corepack enable && corepack prepare pnpm@10.32.1 --activate

WORKDIR /app

# Admin 依赖 clients/shared，Docker context 必须是 clients/。
COPY package.json pnpm-lock.yaml pnpm-workspace.yaml /app/
COPY scripts /app/scripts
COPY shared /app/shared
COPY admin /app/admin

RUN --mount=type=cache,id=pnpm,target=/pnpm/store pnpm install --frozen-lockfile --filter @stuhelper/shared...
RUN pnpm --filter @stuhelper/shared build
RUN --mount=type=cache,id=pnpm-admin,target=/pnpm/store pnpm --dir admin install --frozen-lockfile
RUN pnpm --dir admin --filter @vben/vite-config stub
RUN pnpm --dir admin build:ele

FROM ${NGINX_IMAGE} AS production

# 配置 nginx
RUN echo "types { application/javascript js mjs; }" > /etc/nginx/conf.d/mjs.conf \
    && rm -rf /etc/nginx/conf.d/default.conf

# 复制构建产物
COPY --from=builder /app/admin/apps/web-ele/dist /usr/share/nginx/html

# 复制 nginx 配置
COPY --from=builder /app/admin/scripts/deploy/nginx.conf /etc/nginx/nginx.conf

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://127.0.0.1:8080/ || exit 1

USER nginx

CMD ["nginx", "-g", "daemon off;"]
