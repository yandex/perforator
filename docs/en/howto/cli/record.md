# Collecting a Local Profile

Perforator can collect ad-hoc profiles on a local machine.

## Prerequisites

To work, you need:
1. A recent Linux kernel. Minimum 5.4, preferably 5.15. You can check the kernel version using `uname -r`.
2. Full root access. Perforator requires `CAP_SYS_ADMIN` because it runs an eBPF program capable of reading any state from the kernel or userspace. (run with `sudo`)

## Collect profile of a process and save to a pprof file

```console
perforator record --format pprof -p <pid> --duration 1m --output profile.pprof
```

View `profile.pprof` file.

## Start a subprocess and collect its flamegraph

```console
perforator record --duration 1m -o ./flame.svg -- ls
```

View `flame.svg` file.

## Collect profile of a process and serve a flamegraph on localhost:9000

```console
perforator record -p <pid> --duration 1m --serve ":9000"
```

View the flamegraph at `http://localhost:9000` in your browser.


## Collect profile of a whole system and serve a flamegraph on localhost:9000

```console
perforator record -a --duration 1m --serve ":9000"
```

View the flamegraph at `http://localhost:9000` in your browser.

## Collect profile of a whole system and save flamegraph SVG to file.

```console
perforator record -a --duration 1m --output flame.svg
```

View `flame.svg` file.
