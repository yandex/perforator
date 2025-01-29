GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.46.7)

SRCS(
    decode.go
    defaults.go
    dep_service_ids.go
    doc.go
    endpoints.go
    legacy_regions.go
    v3model.go
)

GO_TEST_SRCS(
    decode_test.go
    endpoints_test.go
    main_test.go
    v3model_legacy_region_test.go
    v3model_shared_test.go
    v3model_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    # gotest
)
