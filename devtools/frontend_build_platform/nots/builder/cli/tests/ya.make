PY3TEST()

TEST_SRCS(
    __init__.py
    test_cli_args.py
    test_models.py
    test_output_meta.py
    test_recipes.py
)

PEERDIR(
    devtools/frontend_build_platform/nots/builder/cli
)

END()
