GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.27.0)

SRCS(
    app_engine.go
    bms.go
    detector.go
    faas.go
    gce.go
    gke.go
)

GO_TEST_SRCS(
    app_engine_test.go
    bms_test.go
    detector_test.go
    faas_test.go
    gce_test.go
    gke_test.go
    utils_test.go
)

END()

RECURSE(
    gotest
)
