# Perforator

## What is Perforator?

Perforator is a modern profiling tool designed for large data centers. Perforator can be easily deployed onto your Kubernetes cluster to collect performance profiles with negligible overhead. Perforator can also be launched as a standalone replacement for Linux perf without the need to recompile your programs.

The profiler is designed to be as non-invasive as possible using beautiful technology called [eBPF](https://ebpf.io). That allows Perforator to profile different languages and runtimes without modification on the build side. Also Perforator supports many advanced features like [sPGO](./guides/autofdo.md) or discriminated profiles for A/B tests.

Perforator is developed by Yandex and used inside Yandex as the main cluster-wide profiling service.

## Quick start
You can start with [tutorial on local usage](./tutorials/native-profiling.md) or delve into [architecture overview](./explanation/architecture/overview.md). Alternatively see a [guide to deploy Perforator on a Kubernetes cluster](guides/helm-chart.md).

## Useful links
- [GitHub repository](https://github.com/yandex/perforator)
- [Documentation](https://perforator.tech/docs)
- [Post on Habr in Russian](https://habr.com/ru/companies/yandex/articles/875070/)
- [Telegram Community chat (RU)](https://t.me/perforator_ru)
- [Telegram Community chat (EN)](https://t.me/perforator_en)
