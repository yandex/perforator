GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20260721132016-d427ff9ee9ad)

ALL_GO_SRCS()

END()

RECURSE(
    jsonflags
    jsonopts
    jsontest
    jsonwire
)
