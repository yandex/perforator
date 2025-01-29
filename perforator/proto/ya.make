RECURSE(
    perforator
    pprofprofile
    profile
    storage
)

IF(NOT OPENSOURCE)
    RECURSE(
        yt
   )
ENDIF()
