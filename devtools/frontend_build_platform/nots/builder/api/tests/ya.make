PY3TEST()

TEST_SRCS(
    __init__.py
    test_create_node_modules.py
    test_globs.py
    test_node_modules_build_lifecycle.py
    test_node_modules_bundler.py
    test_node_modules_layer.py
    test_node_modules_utils.py
    test_output_prefix.py
    test_package_manager.py
    test_prepare_deps.py
    test_ts_proto_generator.py
    test_utils_copy_files_with_exclusions.py
    test_workspace.py
    test_workspace_common_config.py
)

PEERDIR(
    devtools/frontend_build_platform/nots/builder/api
    devtools/frontend_build_platform/nots/builder/cli
)

END()
