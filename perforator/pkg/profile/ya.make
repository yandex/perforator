RECURSE(
    bundle
    flamegraph
    interpreterstack
    labels
    lua
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
