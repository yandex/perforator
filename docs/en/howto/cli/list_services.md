# List Services

Perforator CLI allows you to list profiled services from the Perforator server.

## View the complete list of services sorted by name

```console
perforator list services
```

## View the list of services filtered by regex and sorted by number of profiles

```console
perforator list services -r "perforator" --order-by "profiles"
```

## View the list of services that had profiles in the last hour

```console
perforator list services --max-stale-age "1h"
```

