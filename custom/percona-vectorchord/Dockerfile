# syntax=docker/dockerfile:1
FROM alpine:3.22 AS extension
RUN apk add --no-cache unzip
ADD --checksum=sha256:73fb60bf8b0e2c1bb4df700fa5400d3f2819482e0b7dbdc7e2e88212efff1819 https://github.com/supervc-stack/VectorChord/releases/download/0.5.3/postgresql-18-vchord_0.5.3_x86_64-linux-gnu.zip /tmp/vchord.zip
RUN unzip /tmp/vchord.zip -d /extension

FROM percona/percona-distribution-postgresql-with-postgis:18.6.1-3@sha256:7ac2782afc11321cfd3fb5f5f393b1cf9cada727e32c404cd6f5285604e504e3
COPY --from=extension /extension/pkglibdir/ /usr/pgsql-18/lib/
COPY --from=extension /extension/sharedir/extension/ /usr/pgsql-18/share/extension/
