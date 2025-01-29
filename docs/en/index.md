# Perforator

Perforator is a system for analyzing performance of your code by efficient monitoring of your production apps using eBPF.

## Useful links
Technology - https://ebpf.io

## Architecture
### Overview
Perforator consists of five components: 
- agent, running on every node and responsible for collecting profiles
- storage, responsible for profiles and binaries
- proxy, responsible for aggregating data about CPU utilization
- web, responsible for routing your queries to S3/proxy/hosting your UI
- offline_processing, responsible for preprocessing binaries for speeding up subsequent aggregation queries
