# List Profiles by Selector

Perforator CLI allows you to list profiles by selector.

## View the list of profiles by host for the last 15 minutes

```console
perforator list profiles --node-id "worker-us-east1-b-1" -s "now-15m"
```

## View the list of profiles by service for the last 30 minutes

```console
perforator list profiles --service "kafka-broker" -s "now-30m""
```

## View the list of profiles by abstract selector

```console
perforator list profiles --selector '{node_id="example.org|worker-us-east1-b-1", timestamp>="now-30m"}'
```
