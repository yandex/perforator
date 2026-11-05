#pragma once

#ifdef __x86_64__

#include "arch/x86/unwind_ctx.h"

#elif __aarch64__

#include "arch/arm/unwind_ctx.h"

#else

#error This arch is not supported by Perforator yet

#endif

ALWAYS_INLINE bool read_return_address(void* location, u64* ra) {
    BPF_TRACE("read_return_address: read RA from %p\n", location);
    int err = bpf_probe_read_user(ra, sizeof(*ra), location);
    if (err != 0) {
        BPF_TRACE("read_return_address: bpf_probe_read_user failed: %d\n", err);
        return false;
    }

    // Return addresses point past the call. Move inside it for symbol and unwind-rule lookup.
    // Note that some consumers don't like this change, see Leaf field in profilerext.Address
    // and its references.
    if (*ra != 0) {
        *ra -= 1;
    }

    return true;
}

