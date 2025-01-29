GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v25.0.5+incompatible)

SRCS(
    client.go
    configs.go
    error_response.go
    error_response_ext.go
    graph_driver_data.go
    id_response.go
    plugin.go
    plugin_device.go
    plugin_env.go
    plugin_interface_type.go
    plugin_mount.go
    plugin_responses.go
    port.go
    stats.go
    types.go
    types_deprecated.go
)

END()

RECURSE(
    blkiodev
    checkpoint
    container
    events
    filters
    image
    mount
    network
    plugins
    registry
    strslice
    swarm
    system
    time
    versions
    volume
)
