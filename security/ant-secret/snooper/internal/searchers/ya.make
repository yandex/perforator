LIBRARY()

PEERDIR(
  library/cpp/json
  library/cpp/string_utils/quote
  library/cpp/digest/crc32c

  contrib/libs/re2

  security/ant-secret/internal/base62
  security/ant-secret/internal/blackbox
  security/ant-secret/internal/helpdesk
  security/ant-secret/internal/jwt
  security/ant-secret/internal/re2utils
  security/ant-secret/internal/string_utils
  security/ant-secret/internal/tvm
  security/ant-secret/internal/yc
  security/ant-secret/internal/yhomo
  security/ant-secret/snooper/internal/secret
)

SRCS(
  whole.cpp
  prefixed.cpp
  utils.cpp

  boxberry.cpp
  dev_apikey.cpp
  geo_sharing.cpp
  gitlab.cpp
  jwt.cpp
  market_pull.cpp
  mds_sign.cpp
  ms_cookie.cpp
  s3_presign_v2.cpp
  s3_presign_v4.cpp
  s3_secret_key.cpp
  tvm_secret.cpp
  tvm_service.cpp
  tvm_user.cpp
  url_password.cpp
  yf_token.cpp
  yandex_oauth.cpp
  yandex_session.cpp
  yc_api_key.cpp
  yc_cookie.cpp
  yc_oauth_secret.cpp
  yc_refresh_token.cpp
  yc_static_cred.cpp
  yc_token.cpp
  yhomo_password.cpp
  yrobo_password_v2.cpp
  yrobo_password.cpp
)

END()
