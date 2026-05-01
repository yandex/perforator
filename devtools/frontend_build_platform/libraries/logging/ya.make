PY3_LIBRARY()

PY_SRCS(
    __init__.py
    get_logger.py
    init_logging.py
    timeit.py
)

PEERDIR(
    contrib/python/click
)

END()

RECURSE(
    demo
)

RECURSE_FOR_TESTS(
    tests
)
