LIBRARY()

SRCS(
    error.cpp
    merge.cpp
    profile.cpp
    string.cpp
)

PEERDIR(
    perforator/lib/profile
    perforator/proto/pprofprofile
    perforator/proto/profile
)

END()
