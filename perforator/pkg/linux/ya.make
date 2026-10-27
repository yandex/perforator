GO_LIBRARY()

SRCS(
    types.go
)

IF (OS_LINUX)
    SRCS(
        inode_generation.go
    )
ENDIF()

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
