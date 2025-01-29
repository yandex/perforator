GO_TEST(test)

RESOURCE(
    testdata/a.txt /a.txt
    testdata/b.bin /b.bin
    testdata/collision.txt testdata/collision.txt
    - foo=bar
)

TEST_CWD(library/go/core/resource/test)

DATA(arcadia/library/go/core/resource/test)

GO_TEST_SRCS(resource_test.go)

END()
