GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

SRCS(
    btf.go
    btf_types.go
    btf_types_string.go
    core.go
    doc.go
    ext_info.go
    feature.go
    format.go
    handle.go
    kernel.go
    marshal.go
    strings.go
    traversal.go
    types.go
    workarounds.go
)

END()

RECURSE(
    # gotest
)
