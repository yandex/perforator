FROM postgres:latest

COPY postgres/docker/certificates /etc/postgres/certs

RUN chown root:postgres -R /etc/postgres/certs \
    && chmod 640 /etc/postgres/certs/*