GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    apic.go
    decode.go
    parserc.go
    readerc.go
    resolve.go
    scannerc.go
    yaml.go
    yamlh.go
    yamlprivateh.go
)

GO_XTEST_SRCS(decode_test.go)

END()

RECURSE(
    gotest
)
