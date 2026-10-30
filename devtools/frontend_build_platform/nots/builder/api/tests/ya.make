PY3TEST()

TEST_SRCS(
    __init__.py
    test_create_node_modules.py
    test_globs.py
    test_local_directory_extensions.py
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
    build/external_resources/pnpm
    build/platform/nodejs/24.22.1
    devtools/frontend_build_platform/nots/builder/api
    devtools/frontend_build_platform/nots/builder/cli
)

DATA(
    arcadia/devtools/frontend_build_platform/nots/constants/src/tier0-settings.yaml
)

ENV(NOTS_TEST_PNPM_ROOT=$PNPM_RESOURCE_GLOBAL)

ENV(NOTS_TEST_NODEJS_ROOT=$NODEJS_24_22_1_RESOURCE_GLOBAL)

END()
