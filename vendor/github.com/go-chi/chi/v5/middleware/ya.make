GO_LIBRARY()

LICENSE(MIT)

VERSION(v5.1.0)

GO_SKIP_TESTS(TestWrapWriterHTTP2)

SRCS(
    basic_auth.go
    clean_path.go
    compress.go
    content_charset.go
    content_encoding.go
    content_type.go
    get_head.go
    heartbeat.go
    logger.go
    maybe.go
    middleware.go
    nocache.go
    page_route.go
    path_rewrite.go
    profiler.go
    realip.go
    recoverer.go
    request_id.go
    request_size.go
    route_headers.go
    strip.go
    sunset.go
    supress_notfound.go
    terminal.go
    throttle.go
    timeout.go
    url_format.go
    value.go
    wrap_writer.go
)

GO_TEST_SRCS(
    compress_test.go
    content_charset_test.go
    content_encoding_test.go
    content_type_test.go
    get_head_test.go
    logger_test.go
    middleware_test.go
    realip_test.go
    recoverer_test.go
    request_id_test.go
    strip_test.go
    sunset_test.go
    throttle_test.go
    url_format_test.go
    wrap_writer_test.go
)

END()

RECURSE(
    gotest
)
