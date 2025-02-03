# Force profile saving using Microscope

Microscope is a way to save all profiles from a specific selector bypassing sampling (for example `{host="worker-us-east1-b-1"}`).

## Save profiles from a host {#save-profiles-from-a-host}

Create a microscope to save profiles from the entire host for 1 hour starting from the current moment:

```console
perforator microscope create --node-id "worker-us-east1-b-1" --duration "1h" --start-time "now"
```

After this command, minute-by-minute profiles will start being saved from the host `worker-us-east1-b-1`. You can view the list of profiles from the node for the last 15 minutes using this command:

```console
perforator list profiles --node-id "worker-us-east1-b-1" -s "now-15m"
```

## Save profiles from a pod

Create a microscope to save profiles from a pod for 15 minutes, starting in 30 minutes:

```console
perforator microscope create --pod-id "perforator-storage-production-73" --duration "15m" --start-time "now+30m"
```

View the profiles from the pod for the last 15 minutes:

```console
perforator list profiles --pod-id "perforator-storage-production-73" -s "now-15m"
```

## List created microscopes

View your created microscopes:

```console
perforator microscope list
```
