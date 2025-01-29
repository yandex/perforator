# How to build

There are two primary ways to build Perforator:
1. Build inside Docker container. This method is more convenient if you simply want to obtain working binaries, because it has no host requirements besides Docker.
1. Build directly on host. This method is more convenient if you want to contribute to Perforator (i.e. if you need to quickly patch source code and iterate)

## Components

Perforator consists of several binaries, located in `perforator/cmd` directory.

Currently following components exist:

- `agent`
- `cli`
- `gc`
- `migrate`
- `offline_processing`
- `proxy`
- `storage`
- `web`

## Building inside a container {#container}

{% note info %}

While this guide assumes you are using Docker, any other BuildKit-compatible tool should work as well.

{% endnote %}

Example usage

```bash
# See above for possible ${COMPONENT} values
# Add flags such as `--tag` or `--push` as necessary
docker build -f Dockerfile.build --target ${COMPONENT} ../../..
```

This command will build the desired component and create an image.

To obtain a standalone `cli` binary, you can do this:
```bash
# Replace tmp with cli image name
id=$(docker create tmp -- /)
docker cp ${id}:/perforator/cli .
# Now ./cli is Perforator cli
./cli version
# Cleanup
docker rm ${id}
```

## Building directly on host {#host}

We use [yatool](https://github.com/yandex/yatool) to build Perforator. The only dependency is Python 3.12+.

To build any component, simply run from the repository root:
```bash
./ya make -r <relative path to binary>
```
It will build binary in the `<relative path to binary>` directory in the release mode. If you want to build fully static executable without dynamic linker, add flag `--musl` to the build command. Such executable can be safely transferred between different Linux machines.

There's also a convenient way to build all binaries in one command:
```bash
./ya make -r perforator/bundle
```
Binaries will be available in `perforator/bundle` directory.

To create a docker image for a component named `foo`:
1. Prepare a directory which will be a build context
1. Put `foo` binary in the directory root
1. Additionally, when building Perforator proxy, put `create_llvm_prof` binary in the directory root
1. Run `docker build -f perforator/deploy/docker/Dockerfile.prebuilt --target foo path/to/context` with other flags as necessary.

{% note info %}

Note that Ya has local caches to speed up subsequent builds. Default cache locations are `~/.ya/build` and `~/.ya/tools`. If you do not wish to leave these artifacts in your host system, please use [Docker](#container) with provided Dockerfiles.

{% endnote %}

All microservices required to run in full fledged environment are located in `perforator/cmd` directory. If you wish to record flamegraph locally you can build `perforator/cmd/cli` tool.

Note that initial building process can take a long time and consumes ~10GiB of disk space. Perforator uses LLVM extensively to analyze binaries and we love static builds, so unfortunately we need to compile a lot of LLVM internals.
