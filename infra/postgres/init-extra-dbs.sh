#!/bin/bash
# 为应用、备份、复制和 OpenFGA 创建独立数据库/用户。
# CASDOOR_DB_PASSWORD 非空时额外创建本地开发用 Casdoor 数据库；生产 SSO 独立部署。
set -euo pipefail

: "${POSTGRES_USER:?POSTGRES_USER is required}"
: "${POSTGRES_DB:?POSTGRES_DB is required}"
: "${STUHELPER_APP_DB_PASSWORD:?STUHELPER_APP_DB_PASSWORD is required}"
: "${STUHELPER_BACKUP_DB_PASSWORD:?STUHELPER_BACKUP_DB_PASSWORD is required}"
: "${STUHELPER_REPLICATION_DB_PASSWORD:?STUHELPER_REPLICATION_DB_PASSWORD is required}"
: "${POSTGRES_EXPORTER_DB_PASSWORD:?POSTGRES_EXPORTER_DB_PASSWORD is required}"
: "${OPENFGA_DB_PASSWORD:?OPENFGA_DB_PASSWORD is required}"

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-'EOSQL'
    \getenv stuhelper_app_password STUHELPER_APP_DB_PASSWORD
    \getenv stuhelper_backup_password STUHELPER_BACKUP_DB_PASSWORD
    \getenv stuhelper_replication_password STUHELPER_REPLICATION_DB_PASSWORD
    \getenv postgres_exporter_password POSTGRES_EXPORTER_DB_PASSWORD
    \getenv openfga_password OPENFGA_DB_PASSWORD

    CREATE ROLE stuhelper_app LOGIN PASSWORD :'stuhelper_app_password' CONNECTION LIMIT 30;
    CREATE ROLE stuhelper_backup LOGIN PASSWORD :'stuhelper_backup_password' CONNECTION LIMIT 5;
    CREATE ROLE stuhelper_replication WITH LOGIN REPLICATION PASSWORD :'stuhelper_replication_password' CONNECTION LIMIT 5;
    CREATE ROLE stuhelper_metrics LOGIN PASSWORD :'postgres_exporter_password' CONNECTION LIMIT 5;
    CREATE ROLE openfga LOGIN PASSWORD :'openfga_password' CONNECTION LIMIT 20;

    GRANT pg_read_all_data TO stuhelper_backup;
    GRANT pg_read_all_settings TO stuhelper_backup;
    GRANT pg_read_all_stats TO stuhelper_backup;
    GRANT pg_monitor TO stuhelper_metrics;
    GRANT CONNECT ON DATABASE postgres TO stuhelper_metrics;

    CREATE DATABASE openfga OWNER openfga;
EOSQL

if [ -n "${CASDOOR_DB_PASSWORD:-}" ]; then
psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-'EOSQL'
    \getenv casdoor_password CASDOOR_DB_PASSWORD
    CREATE ROLE casdoor LOGIN PASSWORD :'casdoor_password' CONNECTION LIMIT 20;
    CREATE DATABASE casdoor OWNER casdoor;
EOSQL
fi

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname "$POSTGRES_DB" <<-'EOSQL'
    REVOKE CREATE ON SCHEMA public FROM PUBLIC;
    GRANT USAGE ON SCHEMA public TO stuhelper_app;
    GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO stuhelper_app;
    GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA public TO stuhelper_app;
    ALTER DEFAULT PRIVILEGES GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO stuhelper_app;
    ALTER DEFAULT PRIVILEGES GRANT USAGE, SELECT, UPDATE ON SEQUENCES TO stuhelper_app;
EOSQL

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-'EOSQL'
    \getenv app_database POSTGRES_DB
    SELECT format('GRANT CONNECT ON DATABASE %I TO stuhelper_app', :'app_database') \gexec
    SELECT format('GRANT CONNECT ON DATABASE %I TO stuhelper_backup', :'app_database') \gexec
EOSQL
