#!/bin/bash
# 为应用、备份、复制和 OpenFGA 创建独立数据库/用户。
# CASDOOR_DB_PASSWORD 非空时额外创建本地开发用 Casdoor 数据库；生产 SSO 独立部署。
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    CREATE ROLE stuhelper_app LOGIN PASSWORD '${STUHELPER_APP_DB_PASSWORD}' CONNECTION LIMIT 30;
    CREATE ROLE stuhelper_backup LOGIN PASSWORD '${STUHELPER_BACKUP_DB_PASSWORD}' CONNECTION LIMIT 5;
    CREATE ROLE stuhelper_replication WITH LOGIN REPLICATION PASSWORD '${STUHELPER_REPLICATION_DB_PASSWORD}' CONNECTION LIMIT 5;
    CREATE ROLE openfga LOGIN PASSWORD '${OPENFGA_DB_PASSWORD}' CONNECTION LIMIT 20;

    GRANT pg_read_all_data TO stuhelper_backup;
    GRANT pg_read_all_settings TO stuhelper_backup;
    GRANT pg_read_all_stats TO stuhelper_backup;

    CREATE DATABASE openfga OWNER openfga;
EOSQL

if [ -n "${CASDOOR_DB_PASSWORD:-}" ]; then
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    CREATE ROLE casdoor LOGIN PASSWORD '${CASDOOR_DB_PASSWORD}' CONNECTION LIMIT 20;
    CREATE DATABASE casdoor OWNER casdoor;
EOSQL
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-EOSQL
    REVOKE CREATE ON SCHEMA public FROM PUBLIC;
    GRANT CONNECT ON DATABASE "$POSTGRES_DB" TO stuhelper_app;
    GRANT USAGE ON SCHEMA public TO stuhelper_app;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO stuhelper_app;
    GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO stuhelper_app;
    ALTER DEFAULT PRIVILEGES GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO stuhelper_app;
    ALTER DEFAULT PRIVILEGES GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO stuhelper_app;

    GRANT CONNECT ON DATABASE "$POSTGRES_DB" TO stuhelper_backup;
EOSQL
