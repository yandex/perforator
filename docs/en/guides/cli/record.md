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

## Wait until the profiler is ready

At start, the profiler loads its eBPF program into the kernel, and no samples are collected until it is over. This is the bulk of the startup CPU time. To start a workload once profiling is running, wait for the `Successfully initialized profiler` message in the profiler log (stderr):

```console
perforator record --format pprof -p <pid> --output profile.pprof 2> record.log &
until grep -q 'Successfully initialized profiler' record.log; do sleep 0.2; done
```

The message is logged once, at INFO, so do not raise `--log-level` above `info`. With `--log-format json` the log goes to stdout instead of stderr. The profiler still does some work per process after the message (it analyzes their binaries), but it is small compared to the initialization.

## Start a subprocess and collect its flamegraph

```console
perforator record --duration 1m -o ./flame.html -- ls
```

View `flame.html` file.

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

## Collect profile of a whole system and save flamegraph HTML to file.

```console
perforator record -a --duration 1m --output flame.html
```

View `flame.html` file.
