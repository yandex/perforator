GO_LIBRARY()

LICENSE(MIT)

VERSION(v4.15.2)

SRCS(
    driver.go
    errors.go
    migration.go
    parse.go
)

END()

RECURSE(
    aws_s3
    file
    go_bindata
    godoc_vfs
    google_cloud_storage
    httpfs
    iofs
    stub
    testing
)
