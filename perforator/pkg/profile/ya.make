RECURSE(
    flamegraph
    labels
    parse
    python
    quality
    samplefilter
)

IF(NOT OPENSOURCE)
    RECURSE(
        ytconv
    )
ENDIF()
