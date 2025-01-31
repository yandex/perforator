<img width="64" src="docs/_assets/logo.svg" /><br/>

[![License](https://img.shields.io/badge/License-MIT-blue.svg)](https://github.com/yandex/perforator/blob/main/LICENSE)
[![eBPF code license](https://img.shields.io/badge/eBPF_code_License-GPLv2-blue.svg)](https://github.com/yandex/perforator/tree/main/perforator/agent/collector/progs/unwinder/LICENSE)
[![Telegram](https://img.shields.io/badge/Telegram-ru-2ba2d9.svg)](https://t.me/perforator_ru)
[![Telegram](https://img.shields.io/badge/Telegram-en-2ba2d9.svg)](https://t.me/perforator_en)

# Perforator

[Documentation](https://perforator.tech/docs/)

Perforator is a production-ready, open-source Continuous Profiling app that can collect CPU profiles from your production without affecting its performance, made by Yandex and inspired by [Google-Wide Profiling](https://research.google/pubs/google-wide-profiling-a-continuous-profiling-infrastructure-for-data-centers/). Perforator is deployed on tens of thousands of servers in Yandex and already has helped many developers to fix performance issues in their services.

## Main features
- Efficient and high-quality collection of kernel + userspace stacks via eBPF technology.
- Scalable storage for storing profiles and binaries.
- Support of unwinding without frame pointers and debug symbols on host.
- Convenient query language and UI to inspect CPU usage of applications via flamegraphs.
- Support for C++, C, Go, and Rust, with experimental support for Java and Python.
- Generation of sPGO profiles for building applications with Profile Guided Optimization (PGO) via [AutoFDO](https://github.com/google/autofdo).

## Minimal system requirements

Perforator runs on x86 64-bit Linux platforms consuming 512Mb of RAM (more on very large hosts with many CPUs) and <1% of host CPUs.

## Quick start

You can profile your laptop using local [perforator record CLI command](https://perforator.tech/docs/en/tutorials/native-profiling).

You can also deploy Perforator on playground/production Kubernetes cluster using our [Helm chart](https://perforator.tech/docs/en/guides/helm-chart).

## How to build

- Instructions on how to build from source are located [here](https://perforator.tech/docs/en/guides/build).

- If you want to use prebuilt binaries, you can find them [here](https://github.com/yandex/perforator/releases).

## How to Contribute

We are welcome to contributions! The [contributor's guide](CONTRIBUTING.md) provides more details on how to get started as a contributor.

## License

This project is licensed under the MIT License (MIT). [MIT License](https://github.com/yandex/perforator/tree/main/LICENSE)

The eBPF source code is licensed under the GPL 2.0 license. [GPL 2.0](https://github.com/yandex/perforator/tree/main/perforator/agent/collector/progs/unwinder/LICENSE)
