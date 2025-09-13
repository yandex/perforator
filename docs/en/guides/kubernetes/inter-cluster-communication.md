# Guide for Inter-Cluster Perforator Setup

This guide walks you through setting up Perforator across two separate Kubernetes clusters. You'll create a primary deployment that collects and stores profiles from both its own cluster and remote clusters, while the secondary deployment simply runs an agent that gathers profiles and forwards them to the primary cluster.

## Prerequisites

- Two Kubernetes Clusters
- Helm 3+
- PostgreSQL Database
- ClickHouse Database
- S3 Storage

## Deployment on the Primary Cluster

On your primary cluster, deploy the full application with all components, see [this guide](helm-chart.md) for reference.

1. Add the storage ingress configuration to your values.yaml file:

```yaml
ingress:
  storage:
    enabled: true
    annotations:
      # nginx.ingress.kubernetes.io/backend-protocol: "GRPC" # add this annotation if you are using nginx
    className: # "nginx"
    hosts:
      - host: storage.example.com
        paths:
          - path: /
            pathType: Prefix
```

2. Optional: Configure TLS for the ingress:
 
```yaml
ingress:
  storage:
    enabled: true
    annotations:
      # nginx.ingress.kubernetes.io/backend-protocol: "GRPC" # add this annotation if you are using nginx
    className: # "nginx"
    hosts:
      - host: storage.example.com
        paths:
          - path: /
            pathType: Prefix
    tls:
      - secretName: storage-tls-secret
        hosts:
          - storage.example.com
```
 
3. Install the perforator deployment:

```bash 
helm repo add perforator https://helm.perforator.tech && \
helm repo update && \
helm install perforator-release perforator/perforator -f values.yaml -n perforator --create-namespace
``` 

## Deployment on the Secondary Cluster

1. Change your kubectl context:

```bash
kubectl config use-context <secondary-cluster>
```

2. Create a values-agent.yaml file:
 
```yaml
agent:
  config:
    storageHostnameOverride: "storage.example.com:80"
storage:
  enabled: false
proxy:
  enabled: false
web:
  enabled: false
gc:
  enabled: false
offlineprocessing:
  enabled: false
```
3. Optional: Enable TLS for the agent:

```yaml
agent:
  config:
    storageHostnameOverride: "storage.example.com:443"
  tls:
    enabled: true
```

4. Install the perforator deployment:

```bash 
helm install perforator-release perforator/perforator -f values-agent.yaml -n perforator --create-namespace
```
## Connecting to Perforator UI

Change your kubectl context back to primary cluster:

```bash
kubectl config use-context <primary-cluster>
```

To access the Perforator UI, configure port forwarding to the local machine:

```bash
kubectl port-forward svc/perforator-release-web-service -n perforator 8080:80
```
Then open `http://localhost:8080` in your browser
Verify that the secondary cluster's samples are sent to the primary cluster.