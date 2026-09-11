import os
import shlex

import pytest

from devtools.frontend_build_platform.nots.builder.cli.cli_args import get_args_parser, parse_args


def split_to_argv(command: str) -> list[str]:
    return shlex.split(command)


BASE_ARGS = [
    "--arcadia-root",
    "/source",
    "--arcadia-build-root",
    "/build",
    "--moddir",
    "project",
    "--nodejs-bin",
    "/node",
    "--pm-script",
    "/pnpm",
    "--pm-type",
    "pnpm",
]


@pytest.mark.parametrize("command", ["build-tsc", "build-next", "build-vite", "build-webpack", "build-rspack"])
def test_removed_legacy_commands_are_not_registered(command):
    with pytest.raises(SystemExit) as error:
        get_args_parser().parse_args(BASE_ARGS + [command])
    assert error.value.code == 2


@pytest.mark.parametrize("auto", [False, True])
def test_proto_parser_does_not_depend_on_legacy_builders(auto):
    command = BASE_ARGS + [
        "build-ts-proto",
        "--output-file",
        "/build/project/output.tar",
        "--tsconfigs",
        "tsconfig.json",
        "tsconfig.esm.json",
        "--protoc-bin",
        "/protoc",
        "--proto-paths",
        "/source",
        "/build",
        "--proto-srcs",
        "message.proto",
    ]
    if auto:
        command += ["--auto-package-name", "@proto/*", "--auto-deps-path", "library/typescript/ts-proto-deps"]
    args = get_args_parser().parse_args(command)
    assert args.tsconfigs == ["tsconfig.json", "tsconfig.esm.json"]
    assert args.func.__name__ == "build_ts_proto_func"
    assert args.auto_package_name == ("@proto/*" if auto else None)


def __convert_args_to_dict(command_args: str, nots_builder_verbose_env: str = '') -> dict[str, str]:
    os.environ['NOTS_BUILDER_VERBOSE'] = nots_builder_verbose_env
    parser = get_args_parser()
    args = parse_args(parser, split_to_argv(command_args))

    result = vars(args)
    del result['func']  # skip – this is a function, hard to check

    return result


def test_create_node_modules_args():
    # arrange
    # note the --moddir argument, that is overridden by create_node_modules's argument with the same name
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --local-cli no
        --moddir devtools/dummy_arcadia/typescript/simple/tests
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --verbose no
        create-node-modules
        --moddir devtools/dummy_arcadia/typescript/simple
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        # Flags
        local_cli=False,
        nm_bundle=False,
        nm_bundle_prod=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle=False,
        # Command-specific
        command='create-node-modules',
        moddir='devtools/dummy_arcadia/typescript/simple',  # overridden
    )


def test_create_node_modules_bundle_args():
    # arrange
    # note the --moddir argument, that is overridden by create_node_modules's argument with the same name
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --local-cli no
        --moddir devtools/dummy_arcadia/typescript/simple/tests
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --nm-bundle yes
        --verbose no
        create-node-modules
        --moddir devtools/dummy_arcadia/typescript/simple
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        # Flags
        local_cli=False,
        nm_bundle=True,
        nm_bundle_prod=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/workspace_node_modules.tar',
        # Command-specific
        command='create-node-modules',
        moddir='devtools/dummy_arcadia/typescript/simple',  # overridden
    )


@pytest.mark.parametrize(
    ("build_command_arg", "expected_build_command"),
    [
        ("", None),
        (
            '--build-command "tsc -c tsconfig.json && node --run build:esm"',
            'tsc -c tsconfig.json && node --run build:esm',
        ),
    ],
)
def test_build_library_args(build_command_arg, expected_build_command):
    # arrange
    command_args = f"""
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --moddir devtools/dummy_arcadia/typescript/library
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --verbose no
        build-library
        --output-file /Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/library/output.tar
        --outputs dist build
        --build-script build
        {build_command_arg}
        --exclude-globs ya.make (.idea|.vscode|node_modules)/**/*
        --vcs-info /path/to/vcs_info.json
        --env NODE_ENV=production
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/library',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        env=['NODE_ENV=production'],
        # Flags
        local_cli=False,
        nm_bundle=False,
        nm_bundle_prod=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/library',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/library/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/library',
        node_modules_bundle=False,
        # Command-specific
        command='build-library',
        output_file='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/library/output.tar',
        vcs_info='/path/to/vcs_info.json',
        outputs=['dist', 'build'],
        build_script='build',
        build_command=expected_build_command,
        exclude_globs=["ya.make", "(.idea|.vscode|node_modules)/**/*"],
    )


def test_build_package_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --moddir devtools/dummy_arcadia/typescript/simple
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --verbose no
        build-package
        --output-file /Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/simple',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        env=[],
        # Flags
        local_cli=False,
        nm_bundle=False,
        nm_bundle_prod=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle=False,
        # Command-specific
        command='build-package',
        output_file='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar',
        vcs_info='',
        exclude_globs=[],
        outputs=[],
    )


def test_build_package_nm_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --moddir devtools/dummy_arcadia/typescript/simple
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --nm-bundle yes
        --verbose no
        build-package
        --output-file /Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/simple',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        env=[],
        # Flags
        local_cli=False,
        nm_bundle=True,
        nm_bundle_prod=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/workspace_node_modules.tar',
        # Command-specific
        command='build-package',
        output_file='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar',
        vcs_info='',
        exclude_globs=[],
        outputs=[],
    )


def test_build_verbose_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --moddir devtools/dummy_arcadia/typescript/simple
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        build-package
        --output-file /Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar
    """

    # act + assert
    assert __convert_args_to_dict(command_args, nots_builder_verbose_env='yes') == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/simple',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        ld_library_path=None,
        bun_bin=None,
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        squashfs_tools_path=None,
        use_legacy_pnpm_virtual_store=False,
        inject_peers=False,
        hermetic_node_modules=False,
        env=[],
        # Flags
        local_cli=False,
        nm_bundle=False,
        nm_bundle_prod=False,
        verbose=True,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        node_modules_layer='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/node_modules.layer',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle=False,
        # Command-specific
        command='build-package',
        output_file='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/output.tar',
        vcs_info='',
        exclude_globs=[],
        outputs=[],
    )


@pytest.mark.parametrize("value,expected", [("yes", True), ("no", False)])
def test_nm_bundle_prod_arg(value, expected):
    args = __convert_args_to_dict(
        f"--arcadia-root /source --arcadia-build-root /build --moddir project "
        f"--nodejs-bin /node --pm-script /pnpm.cjs --pm-type pnpm "
        f"--nm-bundle yes --nm-bundle-prod {value} --inject-peers yes create-node-modules --moddir project"
    )
    assert args["nm_bundle_prod"] is expected
