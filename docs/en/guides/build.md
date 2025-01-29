# How to build

We use [yatool](https://github.com/yandex/yatool) to build Perforator. The only dependency is Python 3.12+.

To build any component, simply run from the repository root:
```bash
./ya make -r <relative path to binary>
```
It will build binary in the `<relative path to binary>` directory in the release mode. If you want to build fully static executable without dynamic linker, add flag `--musl` to the build command. Such executable can be safely transferred between different Linux machines.

{% note info %}

Note that Ya has local cache to speed up subsequent builds. If you do not wish to leave these artifacts in your host system, please use [Docker](TODO) with provided Dockerfiles.

{% endnote %}

All microservices required to run in full fledged environment are located in `perforator/cmd` directory. If you wish to record flamegraph locally you can build `perforator/cmd/cli` tool.

Note that initial building process can take a long time and consumes ~10GiB of disk space. Perforator uses LLVM extensively to analyze binaries and we love static builds, so unfortunately we need to compile a lot of LLVM internals.
