# Migrating database schema

This guide shows how to upgrade database schema when upgrading Perforator.

{% note warning %}

To minimize disruption, apply migrations before updating binaries.

{% endnote %}

## Migrating PostgreSQL database

Run the following command from the source root:

```console
./ya run perforator/cmd/migrate postgres up --hosts HOST --user USER --pass PASSWORD
```

Where:

* `HOST` is the hostname of the PostgreSQL primary server.
* `USER` and `PASSWORD` are credentials to connect to the PostgreSQL server.

## Migrating ClickHouse database

Run the following command from the source root:

```console
./ya run perforator/cmd/migrate clickhouse up --hosts HOSTS --user USER --pass PASSWORD
```

Where:

* `HOSTS` is a comma-separated list of ClickHouse hosts.
* `USER` and `PASSWORD` are credentials to connect to the ClickHouse server.

