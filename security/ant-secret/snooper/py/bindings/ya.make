PY23_NATIVE_LIBRARY(ant_secret_snooper_bindings)

SRCS(
  snooper.cpp
  xpython.cpp
)

PEERDIR(
  contrib/libs/pycxx

  security/ant-secret/snooper/cpp
)

INCLUDE(${ARCADIA_ROOT}/security/ant-secret/snooper/py/pycxx.inc)

IF (USE_ARCADIA_PYTHON)
  PY_REGISTER(ant_secret_snooper_bindings)

  SRCS(main.cpp)
ENDIF()

CXXFLAGS(
  ${PYCXX_FLAGS}
)

END()
