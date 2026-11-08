PY3TEST()

TEST_SRCS(
    test_nots_tar.py
)

DEPENDS(
    devtools/frontend_build_platform/nots/builder/tar
)

END()
