GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.3.0)

SRCS(
    date.go
    time.go
    timerfc1123.go
    unixtime.go
    utility.go
)

GO_TEST_SRCS(
    date_test.go
    time_test.go
    timerfc1123_test.go
    unixtime_test.go
)

END()

RECURSE(
    gotest
)
