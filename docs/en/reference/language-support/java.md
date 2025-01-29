# JVM-based language support

This page documents support for Java and other JVM-based languages.

## Requirements

Following requirements must be met:

* HotSpot JVM version 17 or newer is used.

* JVM is running with `-XX:+PreserveFramePointer` flag.

* Perfmap-based symbolization is enabled for the JVM process, and [`java` option](../perfmap.md#configuration-java) is enabled as well.

## Automatic perfmap generation for JVM {#jvm}

When enabled, Perforator can automatically instruct a JVM process to generate perfmap. Internally, agent will use [JDK Attach API](https://docs.oracle.com/en/java/javase/21/docs/api/jdk.attach/module-summary.html) to periodically execute equivalent to the following command
```bash
jcmd ${PID} Compiler.perfmap
```

{% note warning %}

Attach API is an OpenJDK extension. It may be unavailable in other implementations of the JVM.

{% endnote %}
