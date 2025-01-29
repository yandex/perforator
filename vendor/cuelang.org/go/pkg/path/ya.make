GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.4.3)

SRCS(
    match.go
    os.go
    path.go
    path_nix.go
    path_p9.go
    path_win.go
    pkg.go
)

GO_TEST_SRCS(
    match_test.go
    path_test.go
)

GO_XTEST_SRCS(
    example_nix_test.go
    example_test.go
    # pathtxtar_test.go
)

IF (OS_WINDOWS)
    GO_TEST_SRCS(path_windows_test.go)
ENDIF()

END()

RECURSE(
    gotest
)
