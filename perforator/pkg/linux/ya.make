GO_LIBRARY()

SRCS(
    inode_generation.go
    types.go
)

END()

RECURSE(
    cgroupfs
    clock
    cpuinfo
    cpulist
    kallsyms
    memfd
    mountinfo
    perfevent
    pidfd
    procfs
    procmem
    uname
    vdso
)
