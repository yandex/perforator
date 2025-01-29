# Simple databases installation via docker compose
This document helps to deploy local databases for testing full fledged Perforator cluster. It deploys and setups local PostgreSQL, Clickhouse and MinIO (S3) databases via docker compose.

## Prerequisites

- docker

## Setup containers

From the repository root navigate to the docker compose directory.

```console
cd perforator/deploy/db/docker-compose/
```

Start containers

```console
docker compose up -d
```

After starting, you can verify which containers are running.

```console
docker ps
```

After deploying DBs, you should run migrations manually via [migrate tool](migrate-schema.md).