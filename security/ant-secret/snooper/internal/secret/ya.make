LIBRARY()


SRCS(
  secret.cpp
  secret_types.cpp
)

GENERATE_ENUM_SERIALIZATION(secret_types.h)

END()
