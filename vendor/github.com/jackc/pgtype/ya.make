GO_LIBRARY()

LICENSE(MIT)

VERSION(v1.12.0)

SRCS(
    aclitem.go
    aclitem_array.go
    array.go
    array_type.go
    bit.go
    bool.go
    bool_array.go
    box.go
    bpchar.go
    bpchar_array.go
    bytea.go
    bytea_array.go
    cid.go
    cidr.go
    cidr_array.go
    circle.go
    composite_fields.go
    composite_type.go
    convert.go
    database_sql.go
    date.go
    date_array.go
    daterange.go
    enum_array.go
    enum_type.go
    float4.go
    float4_array.go
    float8.go
    float8_array.go
    generic_binary.go
    generic_text.go
    hstore.go
    hstore_array.go
    inet.go
    inet_array.go
    int2.go
    int2_array.go
    int4.go
    int4_array.go
    int4_multirange.go
    int4range.go
    int8.go
    int8_array.go
    int8_multirange.go
    int8range.go
    interval.go
    json.go
    json_array.go
    jsonb.go
    jsonb_array.go
    line.go
    lseg.go
    macaddr.go
    macaddr_array.go
    multirange.go
    name.go
    num_multirange.go
    numeric.go
    numeric_array.go
    numrange.go
    oid.go
    oid_value.go
    path.go
    pgtype.go
    pguint32.go
    point.go
    polygon.go
    qchar.go
    range.go
    record.go
    record_array.go
    text.go
    text_array.go
    tid.go
    time.go
    timestamp.go
    timestamp_array.go
    timestamptz.go
    timestamptz_array.go
    tsrange.go
    tsrange_array.go
    tstzrange.go
    tstzrange_array.go
    unknown.go
    uuid.go
    uuid_array.go
    varbit.go
    varchar.go
    varchar_array.go
    xid.go
)

GO_TEST_SRCS(
    multirange_test.go
    range_test.go
)

GO_XTEST_SRCS(
    # aclitem_array_test.go
    # aclitem_test.go
    array_test.go
    # array_type_test.go
    # bit_test.go
    # bool_array_test.go
    # bool_test.go
    # box_test.go
    # bpchar_array_test.go
    # bpchar_test.go
    # bytea_array_test.go
    # bytea_test.go
    # cid_test.go
    # cidr_array_test.go
    # circle_test.go
    # composite_bench_test.go
    # composite_fields_test.go
    # composite_type_test.go
    custom_composite_test.go
    # date_array_test.go
    # date_test.go
    # daterange_test.go
    # enum_array_test.go
    # enum_type_test.go
    # float4_array_test.go
    # float4_test.go
    # float8_array_test.go
    # float8_test.go
    # hstore_array_test.go
    # hstore_test.go
    # inet_array_test.go
    # inet_test.go
    # int2_array_test.go
    # int2_test.go
    # int4_array_test.go
    int4_multirange_test.go
    # int4_test.go
    # int4range_test.go
    # int8_array_test.go
    int8_multirange_test.go
    # int8_test.go
    # int8range_test.go
    # interval_test.go
    json_array_test.go
    # json_test.go
    jsonb_array_test.go
    # jsonb_test.go
    # line_test.go
    # lseg_test.go
    # macaddr_array_test.go
    # macaddr_test.go
    # name_test.go
    num_multirange_test.go
    # numeric_array_test.go
    # numeric_test.go
    # numrange_test.go
    # oid_value_test.go
    # path_test.go
    # pgtype_test.go
    # point_test.go
    # polygon_test.go
    # qchar_test.go
    record_array_test.go
    # record_test.go
    # text_array_test.go
    # text_test.go
    # tid_test.go
    # time_test.go
    # timestamp_array_test.go
    # timestamp_test.go
    # timestamptz_array_test.go
    # timestamptz_test.go
    # tsrange_test.go
    # tstzrange_test.go
    # uuid_array_test.go
    # uuid_test.go
    # varbit_test.go
    # varchar_array_test.go
    # xid_test.go
)

END()

RECURSE(
    ext
    # gotest
    pgxtype
    testutil
    zeronull
)
