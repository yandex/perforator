GO_LIBRARY()

TAG(ya:run_go_benchmark)

DATA(
    sbr://6915012877=structures
)

SRCS(
    bpf.go
)

END()

RECURSE(
    gotest
)
