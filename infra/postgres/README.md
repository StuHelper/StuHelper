# PostgreSQL runtime files

`infra/generated/postgres/` contains the PostgreSQL server CA private key,
server private key, and certificates. Only the PostgreSQL service may mount
that directory.

Client services mount `infra/generated/postgres-client-ca/`, which is prepared
by `infra/ops/prepare-datastore-client-cas.sh` and contains only `ca.crt`.
