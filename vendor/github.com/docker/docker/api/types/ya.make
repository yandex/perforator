GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v28.2.2+incompatible)

SRCS(
    client.go
    error_response.go
    error_response_ext.go
    plugin.go
    plugin_device.go
    plugin_env.go
    plugin_interface_type.go
    plugin_mount.go
    plugin_responses.go
    types.go
    types_deprecated.go
)

END()

RECURSE(
    auxprogress
    blkiodev
    build
    checkpoint
    common
    container
    events
    filters
    image
    mount
    network
    plugins
    registry
    storage
    strslice
    swarm
    system
    time
    versions
    volume
)
