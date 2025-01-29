GO_LIBRARY()

SRCS(
    inode_generation.go
    types.go
)

END()

RECURSE(
    cpuinfo
    cpulist
    kallsyms
    memfd
    mountinfo
    perfevent
    pidfd
    procfs
    uname
    vdso
)
