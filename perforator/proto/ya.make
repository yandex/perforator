RECURSE(
    custom_profiling_operation
    jvmsupp
    lib
    perforator
    pprofprofile
    profile
    storage
    symbolizer
)

IF(NOT OPENSOURCE)
    RECURSE(
        yt
   )
ENDIF()
