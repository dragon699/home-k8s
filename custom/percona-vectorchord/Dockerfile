# syntax=docker/dockerfile:1
FROM alpine:3.22 AS extension
RUN apk add --no-cache unzip
ADD --checksum=sha256:73fb60bf8b0e2c1bb4df700fa5400d3f2819482e0b7dbdc7e2e88212efff1819 https://github.com/supervc-stack/VectorChord/releases/download/0.5.3/postgresql-18-vchord_0.5.3_x86_64-linux-gnu.zip /tmp/vchord.zip
RUN unzip /tmp/vchord.zip -d /extension

# Preserve Percona's PostgreSQL 18.1-3 runtime, Patroni integration, and user.
FROM percona/percona-distribution-postgresql:18.1-3@sha256:4ff9341d6b0cef309ddfa27fb1224bee3a39378a86f1791409abfc984afeb4b6
COPY --from=extension /extension/pkglibdir/ /usr/pgsql-18/lib/
COPY --from=extension /extension/sharedir/extension/ /usr/pgsql-18/share/extension/
