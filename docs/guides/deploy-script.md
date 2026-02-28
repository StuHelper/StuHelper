# 自动部署脚本（无 Docker）

本文档说明如何使用 `deploy/deploy.sh`：
- 从 `https://gitea.stuhelper.com/StuHelper/StuHelper` 拉取指定分支
- 本地构建前端与后端
- 推送到远端服务器（默认 `81.70.178.230`）

> 适用场景：前后端分离部署、后端单二进制部署、数据库独立部署。

## 1. 前置条件

本机需要以下命令：
- `git`
- `go`
- `ssh`
- `rsync`
- `tar`
- `pnpm`（若你在配置里用 pnpm；也可改成 npm）

远端服务器需要：
- 可 SSH 登录
- 目标目录有写权限
- 已配置后端进程管理（如 systemd）与 Web 服务（如 nginx）

## 2. 统一配置管理

脚本统一读取 `deploy/deploy.env`（可通过 `DEPLOY_ENV_FILE` 覆盖）。

初始化：

1. 复制模板：`deploy/deploy.env.example` -> `deploy/deploy.env`
2. 填写你的分支、远端用户、目录和重启命令

关键配置项：
- `REPO_URL`：仓库地址
- `BRANCH`：目标分支
- `REMOTE_HOST`：默认 `81.70.178.230`
- `REMOTE_USER`：远端 SSH 用户
- `REMOTE_BACKEND_BIN_DIR`：后端二进制落盘目录
- `WEB_MODULES`：前端模块映射（`module:relative_path`）
- `SITES_BASE_DIR`：站点根目录（默认 `/opt/1panel/www/sites`）
- `REMOTE_WEB_DIR`：旧模式下单模块静态目录（仅 `WEB_MODULES` 为空时使用）
- `RESTART_BACKEND_CMD`：后端重启命令（可空）
- `RELOAD_WEB_CMD`：Web 重载命令（可空）

私有仓库认证：
- `GIT_USERNAME`
- `GIT_PASSWORD`（推荐填 token）

如果仓库需要登录，填写后脚本会通过 `GIT_ASKPASS` 非交互拉取。

可选：
- `BACKEND_ENV_FILE`：本地后端环境文件路径，脚本会上传到远端 `REMOTE_BACKEND_ENV_PATH`

或者开启自动生成：
- `BACKEND_ENV_GENERATE=true`
- 用 `BACKEND_*` 统一填写后端配置（包括你提到的必填项）

已覆盖的关键后端项包括：
- `POSTGRES_PASSWORD`
- `REDIS_PASSWORD`
- `DATABASE_URL`
- `APP_ENV`
- `HMAC_SECRET`
- Casdoor 全套字段（endpoint/client_id/client_secret/org/app/redirect/cert）
- `TOKEN_COOKIE_SECURE`
- `CORS_ORIGINS`
- `TRUSTED_PROXIES`
- `METRICS_PASSWORD`
- `DB_SSL_MODE`

## 前端多模块部署（推荐）

可通过 `WEB_MODULES` 一次部署多个前端模块，并映射到对应子域名：

- `clients/web/<module>` -> `<module>.stuhelper.com` -> `/opt/1panel/www/sites/<module>.stuhelper.com/index`
- 特例：`module=homepage` -> `stuhelper.com`

示例：

```dotenv
WEB_MODULES="course:clients/web,homepage:clients/web/homepage"
SITES_BASE_DIR="/opt/1panel/www/sites"
```

兼容模式（旧版）：

- 当 `WEB_MODULES` 留空时，脚本仅构建一个前端目录，并发布到 `REMOTE_WEB_DIR`。

脚本会做基础校验：
- `HMAC_SECRET` 长度 >= 32
- `APP_ENV=production` 时，强制 `TOKEN_COOKIE_SECURE=true` 且 `DB_SSL_MODE!=disable`

## 3. 运行部署

在仓库根目录执行：

```bash
bash deploy/deploy.sh
```

如果你想用其他配置文件：

```bash
DEPLOY_ENV_FILE=/path/to/your.env bash deploy/deploy.sh
```

## 4. 脚本做了什么

1. 读取统一配置
2. 克隆指定分支代码到临时目录
3. 本地构建后端（`server/build/stuhelper`）
4. 本地构建前端（`clients/web/dist`）
5. （可选）自动生成并上传后端 `.env`，并上传 Casdoor 证书文件
6. 使用 `rsync` 推送：
   - 后端二进制 -> `REMOTE_BACKEND_BIN_DIR`
   - 前端静态资源 -> `REMOTE_WEB_DIR`
7. 可选执行远端命令：
   - `RESTART_BACKEND_CMD`
   - `RELOAD_WEB_CMD`

## 5. 建议

- 先确保后端在远端由 `systemd` 托管（便于 restart）
- 数据库/Redis 建议独立部署，不通过本脚本管理
- 首次部署建议先把 `RESTART_BACKEND_CMD` 置空，仅验证上传路径与文件
