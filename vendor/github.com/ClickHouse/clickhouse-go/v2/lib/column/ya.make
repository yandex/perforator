GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v2.33.1)

SRCS(
    array.go
    array_gen.go
    bigint.go
    bool.go
    column.go
    column_gen.go
    column_gen_option.go
    date.go
    date32.go
    datetime.go
    datetime64.go
    decimal.go
    dynamic.go
    dynamic_gen.go
    enum.go
    enum16.go
    enum8.go
    fixed_string.go
    geo_multi_polygon.go
    geo_point.go
    geo_polygon.go
    geo_ring.go
    interval.go
    ipv4.go
    ipv6.go
    json.go
    json_reflect.go
    lowcardinality.go
    map.go
    nested.go
    nothing.go
    nullable.go
    object_json.go
    sharedvariant.go
    simple_aggregate_function.go
    slice_helper.go
    string.go
    time_helper.go
    tuple.go
    uuid.go
    variant.go
)

END()

RECURSE(
    codegen
    orderedmap
)
