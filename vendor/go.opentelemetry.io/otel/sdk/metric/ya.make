GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.36.0)

SRCS(
    aggregation.go
    cache.go
    config.go
    doc.go
    env.go
    exemplar.go
    exporter.go
    instrument.go
    instrumentkind_string.go
    manual_reader.go
    meter.go
    periodic_reader.go
    pipeline.go
    provider.go
    reader.go
    version.go
    view.go
)

GO_TEST_SRCS(
    aggregation_test.go
    benchmark_test.go
    cache_test.go
    config_test.go
    exemplar_test.go
    instrument_test.go
    manual_reader_test.go
    meter_test.go
    periodic_reader_test.go
    pipeline_registry_test.go
    pipeline_test.go
    provider_test.go
    reader_test.go
    version_test.go
    view_test.go
)

GO_XTEST_SRCS(example_test.go)

END()

RECURSE(
    exemplar
    gotest
    internal
    metricdata
)
