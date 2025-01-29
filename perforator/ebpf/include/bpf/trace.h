#pragma once

#include "funcs.h"

#define BPF_PRINTK(fmt, ...)                                               \
    ({                                                                     \
     char __fmt[] = fmt;                                                   \
     bpf_trace_printk(__fmt, sizeof(__fmt), ##__VA_ARGS__);                \
     })

#ifdef BPF_DEBUG

#define BPF_TRACE BPF_PRINTK

#else // BPF_DEBUG

#define BPF_TRACE(...)

#endif // BPF_DEBUG
