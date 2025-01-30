# System requirements
This page describes the system requirements for running Perforator.

## Limitations
Perforator profiling core is based on eBPF which is a relatively new technology and evolving rapidly. So there are some requirements on the system you can run Perforator on.

### Architecture
The only supported architecture is x86-64. We plan to support aarch64 later though.

### Linux kernel version
We support Linux 5.4 or newer. Before Linux 5.4 there was a hard limit on the number of eBPF instructions analyzed by verifier (no more than 4096 instructions), so the profiler is required to run in a very constrained environment. In 5.4 this limit was raised to 1M instructions. Moreover, latest version of the Linux kernel may not work. Complex eBPF programs that read kernel structures by design rely on evolving definitions of the kernel types, so they sometimes should be adapted to the new kernel versions. We test Perforator on all LTS kernels after 5.4, so if you are using LTS kernel you should be fine. Otherwise there is a minor probability that your kernel becomes incompatible with compiled Perforator version. We try to fix such issues as fast as possible.

In addition, the kernel should be compiled with [BPF Type Format](https://docs.kernel.org/bpf/btf.html) (BTF). You can check presence of file `/sys/kernel/btf/vmlinux` to determine if your kernel has BTF. Most modern Linux distributions enable BTF by default. See [this page](https://github.com/libbpf/libbpf#bpf-co-re-compile-once--run-everywhere) for more info.

### Root privileges
Perforator agent should have `CAP_SYS_ADMIN` capability (in most cases this means the agent must run under root user). eBPF programs have access to raw kernel memory, so there is no way to run the agent in more restricted environments.

### LBR profiles
Perforator supports collecting LBR profiles and collects LBR profiles by default. There are two additional requirements on the LBR profiles though:
- The minimal supported kernel version is raised to 5.7. In Linux 5.7 the required eBPF helper which allows us to read last branch records was introduced.
- LBR is supported on Intel processors only.
Such requirements apply only if you are trying to collect and use LBR profiles (for example, for sPGO or BOLT).
