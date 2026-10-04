This image extends the existing Percona PostgreSQL 18.1-3 image with VectorChord 0.5.3 for x86_64. The PostgreSQL major version, operating system, default user, and Percona/Patroni entrypoint are preserved. The base digest and upstream extension archive SHA256 are pinned in Dockerfile.

Build from the repository root:

    docker build -t skullfighter6/percona-postgresql-vectorchord:18.1-3-vchord0.5.3 custom/percona-vectorchord

The smoke-test.sql file creates every Immich database extension and exercises a VectorChord index. Run it in an isolated PostgreSQL instance with shared_preload_libraries=pg_stat_monitor,pgaudit,vchord.so; it expects the nearest-neighbor result to be id1. PostgreSQL18 io_method=worker permits testing inside a container without io_uring permissions.

Before deploying an updated image, repeat that isolated test, publish an immutable image tag/digest, and update helm/apps/postgresql-shared/values.yaml. Keep the existing preload libraries when adding vchord.so. Do not change PostgreSQL major versions by changing the image alone.
