# Perforator Helm Chart
This guide provides instructions on how to deploy Perforator on a Kubernetes cluster via the Helm package manager.

## Prerequisites

- Kubernetes cluster
- Helm 3+
- PostgreSQL database
- ClickHouse database
- S3 storage

{% note warning %}

Make sure you have necessary buckets in your [S3 storage](../../reference/database.md#object-storage)

{% endnote %}

{% note info %}

For testing purposes, you can use this [tutorial](../../tutorials/kubernetes/helm-chart.md) to set up a Perforator deployment with all databases inside your Kubernetes cluster

{% endnote %}

## Adding Helm Repository

```
helm repo add perforator https://helm.perforator.tech
helm repo update
```

## Installing Helm Chart

Create file `my-values.yaml` and add necessary values:

my-values.yaml example
```yaml
databases:
  postgresql:
    migrations:
      enabled: true # Creates necessary migrations in your PostgreSQL database.
    endpoints:
      - host: "<host>"
        port: <port>
    db: "<db>"
    user: "<user>"
    password: "<password>"
  clickhouse:
    migrations:
      enabled: true # Creates necessary migrations in your Clickhouse database.
    replicas:
      - "<host>:<port>"
    db: "<db>"
    user: "<user>"
    password: "<password>"
  s3:
    buckets:
      # If buckets were created with recommended names
      profiles: "perforator-profile"
      binaries: "perforator-binary"
      taskResults: "perforator-task-results"
      binariesGSYM: "perforator-binary-gsym"
    endpoint: "<host>:<port>"
    accessKey: "<accessKey>"
    secretKey: "<secretKey>"
web:
  host: "https://<host.to.serve.UI.on>"
```

{% note info %}

Alternatively, you can use existing kubernetes secrets.

```yaml
databases:
    secretName: "<kubernetes secret>"
    secretKey: "<key>"
  clickhouse:
    secretName: "<kubernetes secret>"
    secretKey: "<key>"
  s3:
    secretName: "<kubernetes secret>"
```

{% endnote %}

Use created values to install chart:

```console
helm install perforator-release -n perforator perforator/perforator -f my-values.yaml
```

## Connecting to Perforator UI

To access the Perforator UI, configure port forwarding to the local machine:

```console
kubectl port-forward svc/perforator-release-web-service -n perforator 8080:80
```
Then open `http://localhost:8080` in your browser

## Uninstalling Helm Chart

```console
helm uninstall perforator-release -n perforator
```

## Upgrading Helm Chart

```console
helm upgrade perforator-release -n perforator perforator/perforator -f my-values.yaml
```
