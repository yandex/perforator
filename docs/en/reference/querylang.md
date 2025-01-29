# Querylang

This page describes the query language used by Perforator.

## Selector

In Perforator, selectors are used to specify a set of profiles. They filter the total set of profiles to aggregate info from by profile or stack labels.

## Examples

- `{service="perforator.storage-production", timestamp>="now-30m", timestamp<"now"}` - a set of profiles collected from the deployment `perforator.storage-production` over the last 30 minutes.
- `{build_ids="abacaba|aba", node_id="cl1e9f5mob6348aja6cc-ywel"}` - a set of profiles collected from the node `cl1e9f5mob6348aja6cc-ywel` for binaries with `build_id` `abacaba` or `aba`.
- `{pod_id="perforator-storage-production-73", timestamp >= "now-30m"}` - a set of profiles collected from the pod `perforator-storage-production-73` no earlier than `now-30m`.
- `{service="perforator.storage-production", event_type="wall.seconds"}` - a set of profiles collected from the deployment `perforator.storage-production` with `wall.seconds` event type.

