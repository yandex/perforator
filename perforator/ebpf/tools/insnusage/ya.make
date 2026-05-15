PROGRAM(insnusage)

# perforator/ebpf/ is GPL-2.0 (see perforator/ebpf/LICENSE); libbpf
# (LGPL-2.1 or BSD) and libelf (GPL-2 or LGPL-3, elfutils) are
# GPL-2-compatible. Exempt from the default permissive-license check.
LICENSE_RESTRICTION_EXCEPTIONS(
    contrib/libs/libbpf
    contrib/restricted/libelf
)

SRCS(
    main.cpp
)

ADDINCL(
    contrib/libs/libbpf/include
    contrib/libs/libbpf/src
)

# libbpf's <btf.h> pulls in <linux-tools/types.h> which redefines
# __bitwise / __bitwise__ relative to <linux-headers/linux/types.h> (which
# is on the include path via libbpf.h → linux/bpf.h). The two are
# semantically equivalent (both ultimately empty) but lexically different,
# tripping -Werror -Wmacro-redefined. Suppress the warning for this target.
CFLAGS(-Wno-macro-redefined)

PEERDIR(
    contrib/libs/libbpf
    contrib/libs/llvm18/include
    contrib/libs/llvm18/lib/DebugInfo/BTF
    contrib/libs/llvm18/lib/DebugInfo/DWARF
    contrib/libs/llvm18/lib/Object
    library/cpp/getopt
    perforator/lib/llvmex
    perforator/lib/profile
)

END()
