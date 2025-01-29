GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    bpffs.go
    cgroup.go
    checkers.go
    cpu.go
    feature.go
    glob.go
    programs.go
    rlimit.go
    seed.go
)

END()

RECURSE(
    fdtrace
)
