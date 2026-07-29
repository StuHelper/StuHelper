# Redis runtime files

`infra/generated/redis/` contains the Redis CA private key, server private key,
certificates, and ACL password verifiers. The ACL file stores SHA-256 password
hashes rather than plaintext credentials. Only the Redis service may mount that
directory.

Client services mount `infra/generated/redis-client-ca/`, which is prepared by
`infra/ops/prepare-datastore-client-cas.sh` and contains only `ca.crt`.

The application user and `stuhelper_metrics` exporter user have independent
passwords and explicit command allowlists. The exporter cannot read or write
application keys, and its single-target `/scrape` endpoint is disabled.
