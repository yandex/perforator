PY3_LIBRARY()

STYLE_PYTHON()

PY_SRCS(
    commands/build_next.py
    commands/build_package.py
    commands/build_ts_proto.py
    commands/build_tsc.py
    commands/build_vite.py
    commands/build_webpack.py
    commands/create_node_modules.py
    commands/prepare_deps.py
    __init__.py
    main.py
    models.py
    cli_args.py
)

PEERDIR(
    build/plugins/lib/nots
    devtools/frontend_build_platform/nots/builder/api
    library/python/archive
)

END()

RECURSE_FOR_TESTS(
    tests
)
