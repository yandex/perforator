RECURSE(
    flamegraph
    labels
    parse
    python
    samplefilter
)

IF(NOT OPENSOURCE)
    RECURSE(
        ytconv
    )
ENDIF()
