LIBRARY()

SRCS(
  jwt.cpp
)

PEERDIR(
  library/cpp/string_utils/base64

  security/ant-secret/internal/string_utils
)

END()

RECURSE_FOR_TESTS(
  ut
)
