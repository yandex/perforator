IF (TS_USE_PREBUILT_NOTS_TOOL)
    INCLUDE(prebuilt.inc)
ENDIF()

IF (NOT PREBUILT)
    MESSAGE(Using branch nots/builder)
    INCLUDE(local.inc)
ENDIF()

RECURSE(
    api
    cli
)
