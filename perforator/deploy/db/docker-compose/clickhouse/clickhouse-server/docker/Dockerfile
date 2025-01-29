FROM clickhouse/clickhouse-server

COPY clickhouse/clickhouse-server/docker/certificates /etc/clickhouse-server/certs

RUN chown clickhouse:clickhouse -R /etc/clickhouse-server/certs \
    && chmod 600 /etc/clickhouse-server/certs/* \
    && chmod 755 /etc/clickhouse-server/certs