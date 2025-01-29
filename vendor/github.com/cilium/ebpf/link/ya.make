GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    anchor.go
    cgroup.go
    doc.go
    iter.go
    kprobe.go
    kprobe_multi.go
    link.go
    netfilter.go
    netkit.go
    netns.go
    perf_event.go
    program.go
    query.go
    raw_tracepoint.go
    socket_filter.go
    syscalls.go
    tcx.go
    tracepoint.go
    tracing.go
    uprobe.go
    uprobe_multi.go
    xdp.go
)

END()

RECURSE(
    # gotest
)
