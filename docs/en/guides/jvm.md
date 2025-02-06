# How to configure profiling for JVM applications

Perforator is able to profile JVM application. This guide shows how to configure application to get meaningful profiles.

{% note warning %}

For now, JVM support is experimental and has known limitations. It will be improved in the future releases.

{% endnote %}

## Prerequisites

* Application runs on HotSpot JVM.
* JVM is 17 or newer.

## Configure JVM

Add the following flag to the JVM process

```
-XX:+PreserveFramePointer
```

Additionally, add the following environment variable to the JVM process

```
__PERFORATOR_ENABLE_PERFMAP=java=true,percentage=50
```

See [reference](../reference/perfmap.md#configuration) for configuration options.
