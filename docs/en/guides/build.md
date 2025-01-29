# How to build

We use [Ya](https://github.com/yandex/yatool) to build Perforator.
The only dependency is Python 3.12+.

To build any component, simply run from the repository root:
```bash
    ./ya make <relative path to binary>
```
It will build static binary in the `<relative path to binary>` directory.

{% note info %}

Note that Ya has local cache to speed up subsequent builds. If you do not wish to leave these artifacts in your host system, please use [Docker](TODO) with provided Dockerfiles.

{% endnote %}

All microservices required to run in full fledged environment are located in `perforator/cmd` directory.

If you wish to record flamegraph locally you can build `perforator/cmd/cli` tool.
