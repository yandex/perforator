RECURSE(
    flamegraph
    interpreterstack
    labels
    merge
    parse
    php
    python
    quality
    samplefilter
)

IF(NOT OPENSOURCE)
    RECURSE(
        ytconv
    )
ENDIF()
