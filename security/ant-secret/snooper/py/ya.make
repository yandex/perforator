PY23_LIBRARY()

PY_SRCS(
  NAMESPACE security.ant_secret.snooper
  __init__.py
)

PEERDIR(
  security/ant-secret/snooper/py/bindings
)

END()

RECURSE(
  example
  pypi
  bindings
  bindings_py
)

RECURSE_FOR_TESTS(
  ut
)
