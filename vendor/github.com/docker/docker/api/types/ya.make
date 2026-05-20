GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v27.5.1+incompatible)

SRCS(
    client.go
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
    types.go
    types_deprecated.go
)

END()

RECURSE(
    auxprogress
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
