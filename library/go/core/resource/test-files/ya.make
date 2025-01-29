GO_TEST(test)

RESOURCE_FILES(
    testdata/a.txt
    testdata/b.bin
)

RESOURCE_FILES(
    PREFIX my_data/
    testdata/collision.txt
)

TEST_CWD(library/go/core/resource/test-files)

DATA(arcadia/library/go/core/resource/test-files)

GO_TEST_SRCS(resource_test.go)

END()
