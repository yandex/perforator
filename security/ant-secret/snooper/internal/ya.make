LIBRARY()


SRCS(
  masker.cpp
  proto_searcher.cpp
  searcher.cpp
  searchers_store.cpp
)

PEERDIR(
  contrib/libs/re2

  security/ant-secret/internal/protobuf
  security/ant-secret/internal/re2utils
  security/ant-secret/internal/validation

  security/ant-secret/snooper/internal/searchers
  security/ant-secret/snooper/internal/secret
)

END()

RECURSE(
  searchers
  secret
)
