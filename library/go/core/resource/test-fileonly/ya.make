GO_TEST(test)

RESOURCE(
    testdata/a.txt /a.txt
    testdata/b.bin /b.bin
    testdata/collision.txt testdata/collision.txt
)

TEST_CWD(library/go/core/resource/test-fileonly)

DATA(arcadia/library/go/core/resource/test-fileonly)

GO_TEST_SRCS(resource_test.go)

END()
