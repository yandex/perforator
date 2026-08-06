PY3_LIBRARY()

STYLE_PYTHON()

PY_SRCS(
    commands/build_library.py
    commands/build_next.py
    commands/build_package.py
    commands/build_ts_proto.py
    commands/build_tsc.py
    commands/build_vite.py
    commands/build_webpack.py
    commands/build_rspack.py
    commands/create_node_modules.py
    commands/extract_node_modules.py
    commands/extract_output_tars.py
    commands/install_node_modules.py
    commands/prepare_deps.py
    commands/recipe_utils.py
    __init__.py
    main.py
    models.py
    recipes.py
    cli_args.py
)

PEERDIR(
    build/plugins/lib/nots
    devtools/frontend_build_platform/nots/builder/api
    library/python/archive
    library/python/testing/recipe
    library/python/testing/yatest_common
)

END()

RECURSE_FOR_TESTS(
    tests
)
