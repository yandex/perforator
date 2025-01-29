IF (TS_USE_PREBUILT_NOTS_TOOL)
    INCLUDE(prebuilt.ya.make.inc)
ENDIF()

IF (NOT PREBUILT)
    MESSAGE(Using branch nots/recipes/extract_output_tars)
    INCLUDE(local.ya.make.inc)
ENDIF()
