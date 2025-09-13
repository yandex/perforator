# Databases

Perforator relies on several databases to store its persistent state.

## PostgreSQL

Perforator stores various metadata and configuration in a PostgreSQL database.

### Tables

Following tables are used:

* `tasks` - contains recent tasks (user-specified parameters as well as coordination state and task outcome).

* `binaries` - contains metadata for binaries, uploaded to [object storage](#object-storage).

* `banned_users` - contains users banned from interacting with API (i.e. they are unable to create new tasks).

* `binary_processing_queue` - queue-like table used in GSYM-based symbolization.

[//]: # (TODO: link to GSYM)

* `gsym` - contains metadata for the GSYM-based symbolization.

* `schema_migrations` - coordination state for the [database migration process](../howto/migrate-schema).

* `microscopes` - contains data for the [microscopes](./microscope).


## ClickHouse

Perforator stores profiles metadata in a ClickHouse table named `profiles`. Perforator uses it to find profiles matching filter.

[//]: # (Document projections?)

## Object storage {#object-storage}

Additionally, Perforator requires the following S3-compatible object storage buckets:

* `profiles` - stores raw profiles referenced by metadata stored in the ClickHouse database

* `binaries` - contains the actual binary files referenced by metadata in the PostgreSQL database

* `taskResults` - stores the outcomes of user-requested tasks

* `binariesGSYM` - contains processed binaries in GSYM format for faster symbolization

{% note warning %}

You'll need to set up at least two separate buckets for your Perforator deployment to store binaries and binariesGSYM files separately, due to their collision when placed in the same bucket

{% endnote %}

## Compatibility

Perforator is tested to work with the following databases:

* PostgreSQL 15.

* ClickHouse 24.3 and 24.8.
