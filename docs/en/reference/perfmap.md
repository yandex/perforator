# Perfmap-based symbolization

This page explains how Perforator uses perfmap.

## Motivation

Many applications employ runtime code generation. For example, this is true for all code written in JIT-compiled languages such as Java or JavaScript. `perf` tool established perfmap - a simple protocol which allows an application to report all dynamic symbols to a profiling tool. Perforator is capable of loading and using this information.

## Configuration {#configuration}

Perfmap usage is controlled by environment variable `__PERFORATOR_ENABLE_PERFMAP`. Its value consists of comma-separated key-value pairs specifying individual settings, e.g. `java=true,percentage=25`. Any value, including empty string, is a signal that perfmap should be used in addition to any other methods when profiling this process. Any unsupported settings are ignored.

### `java` {#configuration-java}

This setting enables [JVM-specific support](./language-support/java.md#jvm). The only supported value is `true`.

### `percentage`

Percentage option can be used to gradually rollout perfmap-based symbolization. Note that it is applied uniformly at random for each process, so overall share of affected processes may differ. Supported values are 0..100 inclusively, where 0 effectively disables feature. If not set, default value is 100.

## Loading perfmaps

Perforator agent follows convention that perfmap is written to `/tmp/perf-N.map`, where N is the process identifier (PID). If process is running in container, agent searches inside the container's root directory. Additionally, instead of the real PID agent uses namespaced pid, i.e. pid inside the container. This way, profiling containerized workloads just works.

Perforator supports both streaming (i.e. process appends new symbols to the end of the perfmap) and periodic (i.e. process periodically rewrites the perfmap with all active symbols) updates.

By default Perforator assumes that creating perfmap will be arranged by some external entity (i.e. application developer specifying certain flags). However, [special support](./language-support/java.md#jvm) is provided for Java.
