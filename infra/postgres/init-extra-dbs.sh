#!/bin/bash
# 为 Zitadel 和 OpenFGA 创建独立数据库
# 共享 PostgreSQL 模式：stuhelper(主库) + zitadel + openfga
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    CREATE DATABASE zitadel;
    GRANT ALL PRIVILEGES ON DATABASE zitadel TO "$POSTGRES_USER";
    CREATE DATABASE openfga;
    GRANT ALL PRIVILEGES ON DATABASE openfga TO "$POSTGRES_USER";
EOSQL
