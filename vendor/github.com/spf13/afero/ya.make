GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v1.11.0)

SRCS(
    afero.go
    basepath.go
    cacheOnReadFs.go
    copyOnWriteFs.go
    httpFs.go
    iofs.go
    ioutil.go
    lstater.go
    match.go
    memmap.go
    os.go
    path.go
    readonlyfs.go
    regexpfs.go
    symlink.go
    unionFile.go
    util.go
)

GO_TEST_SRCS(
    afero_test.go
    basepath_test.go
    composite_test.go
    copyOnWriteFs_test.go
    iofs_test.go
    ioutil_test.go
    lstater_test.go
    match_test.go
    memmap_test.go
    path_test.go
    ro_regexp_test.go
    symlink_test.go
    util_test.go
)

IF (OS_LINUX)
    SRCS(
        const_win_unix.go
    )
ENDIF()

IF (OS_DARWIN)
    SRCS(
        const_bsds.go
    )
ENDIF()

IF (OS_WINDOWS)
    SRCS(
        const_win_unix.go
    )
ENDIF()

END()

RECURSE(
    # gcsfs
    gotest
    internal
    mem
    # sftpfs
    tarfs
    zipfs
)
