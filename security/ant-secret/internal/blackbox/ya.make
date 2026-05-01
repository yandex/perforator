LIBRARY()

SRCS(
  oauth.cpp
)

PEERDIR(
  library/cpp/string_utils/base64
  library/cpp/digest/old_crc

  security/ant-secret/internal/string_utils
)

END()
