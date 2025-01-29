GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v4.4.1)

SRCS(
    allocator.go
    doc.go
    fields.go
    jsontagutil.go
    list.go
    listreflect.go
    listunstructured.go
    map.go
    mapreflect.go
    mapunstructured.go
    reflectcache.go
    scalar.go
    structreflect.go
    value.go
    valuereflect.go
    valueunstructured.go
)

GO_TEST_SRCS(
    less_test.go
    reflectcache_test.go
    valuereflect_test.go
)

GO_XTEST_SRCS(equals_test.go)

END()

RECURSE(
    gotest
)
