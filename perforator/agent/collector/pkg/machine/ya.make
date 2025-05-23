GO_LIBRARY()

SRCS(
    bpf.go
    links.go
)

END()

RECURSE(
    uprobe
)
