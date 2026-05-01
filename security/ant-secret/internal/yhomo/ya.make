LIBRARY()

SRCS(
  yhomo.cpp
)

PEERDIR(
  library/cpp/digest/crc32c
  library/cpp/string_utils/base64

  security/ant-secret/internal/string_utils
  security/ant-secret/pwdgen/pkg/pwd
)

END()
