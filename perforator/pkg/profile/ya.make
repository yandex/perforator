RECURSE(
    flamegraph
    interpreterstack
    labels
    merge
    parse
    php
    lua
    python
    quality
    samplefilter
)

IF(NOT OPENSOURCE)
    RECURSE(
        ytconv
    )
ENDIF()
