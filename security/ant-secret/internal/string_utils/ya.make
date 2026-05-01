LIBRARY()

SRCS(
  common.cpp
  entropy.cpp
  hash.cpp
  reducer.cpp
)

PEERDIR(
  contrib/libs/openssl
)

END()

RECURSE_FOR_TESTS(
  ut
)
