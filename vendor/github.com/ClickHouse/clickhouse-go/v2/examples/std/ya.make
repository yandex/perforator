GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.46.0)

SRCS(
    async_http.go
    async_native.go
    auth.go
    batch.go
    bfloat16.go
    bind.go
    client_info.go
    compression.go
    connect.go
    connect_http.go
    connect_settings.go
    context.go
    dynamic.go
    dynamic_scan_types.go
    ephemeral_http.go
    ephemeral_native.go
    exec.go
    external_data.go
    geo.go
    json_paths.go
    json_strings.go
    map.go
    multi_host.go
    open_db.go
    open_telemetry.go
    progress.go
    qbit.go
    qbit_subcolumns.go
    query_parameters.go
    query_row.go
    query_rows.go
    ssl.go
    utils.go
    variant.go
)

END()
