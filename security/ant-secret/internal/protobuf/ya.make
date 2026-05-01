LIBRARY()

SRCS(
  protobuf.cpp
)

PEERDIR(
  library/cpp/protobuf/runtime

  security/ant-secret/internal/string_utils
)

END()

RECURSE_FOR_TESTS(
  ut
)

RECURSE(
  fuzzer
)
