LIBRARY()


SRCS(
  searcher.cpp
  secret.cpp
  snooper.cpp
  validator.cpp
)

PEERDIR(
  security/ant-secret/snooper/internal/secret
  security/ant-secret/snooper/internal
  security/ant-secret/internal/validation
)

END()

RECURSE(
  logger
  perf
  ut
  fuzz
)
