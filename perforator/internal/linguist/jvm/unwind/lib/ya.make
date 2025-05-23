LIBRARY()

INCLUDE(gen.inc)
IF (DEFINED JDK_NOT_CONFIGURED)
    SRCS(
        fallback.cpp
    )
ELSE()

    SRCS(
        cheatsheet.cpp
        offsets.cpp
    )

ENDIF()

# TODO: get rid of this
NO_COMPILER_WARNINGS()


END()
