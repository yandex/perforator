IF (NOT OPENSOURCE)
    RECURSE(
        alerts
        docs
        opensource
        sandbox
        scripts
        tasklets
        v0
    )
ENDIF()

IF (NOT CI)
    RECURSE(ui)
ENDIF()

RECURSE(
    agent
    bundle    
    cmd
    ebpf
    internal
    lib
    pkg
    proto
    symbolizer
    tools
    util
)
