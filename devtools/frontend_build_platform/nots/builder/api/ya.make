PY3_LIBRARY()

STYLE_PYTHON()

PY_SRCS(
    __init__.py
    builders/__init__.py
    builders/base_builder.py
    builders/next_builder.py
    builders/tsc_builder.py
    builders/vite_builder.py
    builders/webpack_builder.py
    generators/default_ts_proto_opt.py
    generators/ts_proto_generator.py
    create_node_modules.py
    models.py
    prepare_deps.py
    utils.py
)

PEERDIR(
    build/plugins/lib/nots/package_manager
    build/plugins/lib/nots/typescript
    contrib/python/click
    devtools/frontend_build_platform/libraries/logging
    devtools/ya/yalibrary/fetcher/uri_parser
    library/python/archive
    library/python/color
    library/python/fs
    library/python/resource
)

END()
