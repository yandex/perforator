# Fetch Profile

Perforator CLI allows you to fetch profiles from the Perforator server.

## Collect a flamegraph for a service over 30 minutes from 10 samples and start viewer on localhost:9000

```console
perforator fetch --format=flamegraph -s "now-30m" --service "redis-master" -m 10 --serve ":9000""
```

## Collect a flamegraph from a pod for the last 30 minutes and start viewer on localhost:9000

```console
perforator fetch -s "now-30m" --pod-id "mongodb-statefulset-2" -m 10 --serve ":9000"
```

## Collect a flamegraph for an executable over 30 minutes from 10 samples

To identify the executable, use the BuildID. You can find the BuildID using the following command:

```console
readelf -n <path_to_binary>
```

```console
perforator fetch --format=flamegraph -s "now-30m" --build-id "abacaba" -m 10 --serve ":9000"
```

## Collect a pprof profile for an arbitrary selector

```console
perforator fetch --format=pprof -s "now-30m" --selector
'{node_id="example.org|worker-us-east1-b-1", timestamp>="now-30m"}' -m 10 -o profile.pprof
```

## Collect a flamegraph filtered by TLS string variable value

Before collecting a profile, you need to mark TLS variables in your code using one of `Y_PERFORATOR_THREAD_LOCAL` macros.

```console 
perforator fetch --format=flamegraph -s "now-30m" --selector {node_id="worker-us-east1-b-1", "tls.TString_KEY"="VALUE"}
```

