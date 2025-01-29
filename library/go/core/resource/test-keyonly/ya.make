GO_TEST(test)

RESOURCE(
    - foo=bar
    - bar=baz
)

GO_TEST_SRCS(resource_test.go)

END()
