import textwrap

from devtools.frontend_build_platform.nots.builder.cli.cli_args import get_args_parser, parse_args


def split_to_argv(command: str) -> list[str]:
    return textwrap.dedent(command).strip().replace('\n', ' ').split(' ')


def __convert_args_to_dict(command_args: str) -> dict[str, str]:
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
        --trace no
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
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        # Flags
        local_cli=False,
        bundle=True,
        trace=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/workspace_node_modules.tar',
        # Command-specific
        command='create-node-modules',
        moddir='devtools/dummy_arcadia/typescript/simple',  # overridden
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
        --trace no
        --verbose no
        build-package
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/simple',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        # Flags
        local_cli=False,
        bundle=True,
        trace=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/workspace_node_modules.tar',
        # Command-specific
        command='build-package',
    )


def test_build_tsc_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/5gxr/000067
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/simple
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace yes
        --verbose yes
        build-tsc
        --output-file /Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/dummy_arcadia_typescript_simple.output.tar
        --tsconfigs tsconfig.json
        --vcs-info
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/5gxr/000067',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/simple',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=[],
        # Flags
        local_cli=True,
        bundle=True,
        trace=True,
        verbose=True,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/simple',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/workspace_node_modules.tar',
        # Command-specific
        command='build-tsc',
        output_file='/Users/khoden/.ya/build/build_root/5gxr/000067/devtools/dummy_arcadia/typescript/simple/dummy_arcadia_typescript_simple.output.tar',
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js=None,
        after_build_args=None,
        after_build_outdir=None,
    )


def test_build_next_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/j1sk/000245
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/nextjs13
        --nodejs-bin /Users/khoden/.ya/tools/v4/3777807975/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace no
        --verbose no
        build-next
        --output-file /Users/khoden/.ya/build/build_root/j1sk/000245/devtools/dummy_arcadia/typescript/nextjs13/dummy_arcadia_nextjs13.output.tar
        --tsconfigs tsconfig.json
        --vcs-info
        --ts-next-command build
        --bundler-config-path /Users/khoden/arcadia/devtools/dummy_arcadia/typescript/nextjs13/next.config.js
        --output-dirs .next
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/j1sk/000245',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/nextjs13',
        nodejs_bin='/Users/khoden/.ya/tools/v4/3777807975/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=[],
        # Flags
        local_cli=True,
        bundle=True,
        trace=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/j1sk/000245/devtools/dummy_arcadia/typescript/nextjs13',
        bundler_config_path='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/nextjs13/next.config.js',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/nextjs13',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/j1sk/000245/devtools/dummy_arcadia/typescript/nextjs13/workspace_node_modules.tar',
        # Command-specific
        bundler_config='next.config.js',
        command='build-next',
        output_file='/Users/khoden/.ya/build/build_root/j1sk/000245/devtools/dummy_arcadia/typescript/nextjs13/dummy_arcadia_nextjs13.output.tar',
        output_dirs=['.next'],
        ts_next_command='build',
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js=None,
        after_build_args=None,
        after_build_outdir=None,
    )


def test_build_vite_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/41qi/0000e5
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/vite_project
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace no
        --verbose no
        build-vite
        --output-file /Users/khoden/.ya/build/build_root/41qi/0000e5/devtools/dummy_arcadia/typescript/vite_project/dummy_arcadia_typescript_vite_project.output.tar
        --tsconfigs tsconfig.json
        --vcs-info
        --bundler-config-path /Users/khoden/arcadia/devtools/dummy_arcadia/typescript/vite_project/vite.config.ts
        --output-dirs dist
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/41qi/0000e5',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/vite_project',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=[],
        # Flags
        local_cli=True,
        bundle=True,
        trace=False,
        verbose=False,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/41qi/0000e5/devtools/dummy_arcadia/typescript/vite_project',
        bundler_config_path='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/vite_project/vite.config.ts',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/vite_project',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/41qi/0000e5/devtools/dummy_arcadia/typescript/vite_project/workspace_node_modules.tar',
        # Command-specific
        bundler_config='vite.config.ts',
        command='build-vite',
        output_file='/Users/khoden/.ya/build/build_root/41qi/0000e5/devtools/dummy_arcadia/typescript/vite_project/dummy_arcadia_typescript_vite_project.output.tar',
        output_dirs=['dist'],
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js=None,
        after_build_args=None,
        after_build_outdir=None,
    )


# noinspection SpellCheckingInspection
def test_build_webpack_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/emev/00008e
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/with_simple_bundling
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace no
        --verbose yes
        build-webpack
        --bundler-config-path /Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js
        --output-file /Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/dummy_arcadia_typescript_with_simple_bundling.output.tar
        --output-dirs dev-bundle prod-bundle
        --tsconfigs tsconfig.json
        --vcs-info
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/emev/00008e',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/with_simple_bundling',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=[],
        # Flags
        local_cli=True,
        bundle=True,
        trace=False,
        verbose=True,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling',
        bundler_config_path='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/workspace_node_modules.tar',
        # Command-specific
        bundler_config='webpack.config.js',
        command='build-webpack',
        output_file='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/dummy_arcadia_typescript_with_simple_bundling.output.tar',
        output_dirs=['dev-bundle', 'prod-bundle'],
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js=None,
        after_build_args=None,
        after_build_outdir=None,
    )


# noinspection SpellCheckingInspection
def test_build_webpack_with_env_args():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/emev/00008e
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/with_simple_bundling
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace no
        --verbose yes
        build-webpack
        --bundler-config-path /Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js
        --output-file /Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/dummy_arcadia_typescript_with_simple_bundling.output.tar
        --output-dirs dev-bundle prod-bundle
        --tsconfigs tsconfig.json
        --vcs-info
        --env VAR1=value
        --env VAR2=value
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/emev/00008e',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/with_simple_bundling',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=["VAR1=value", "VAR2=value"],
        # Flags
        local_cli=True,
        bundle=True,
        trace=False,
        verbose=True,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling',
        bundler_config_path='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/workspace_node_modules.tar',
        # Command-specific
        bundler_config='webpack.config.js',
        command='build-webpack',
        output_file='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/dummy_arcadia_typescript_with_simple_bundling.output.tar',
        output_dirs=['dev-bundle', 'prod-bundle'],
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js=None,
        after_build_args=None,
        after_build_outdir=None,
    )


# noinspection SpellCheckingInspection
def test_build_webpack_with_after_build():
    # arrange
    command_args = """
        --arcadia-root /Users/khoden/arcadia
        --arcadia-build-root /Users/khoden/.ya/build/build_root/emev/00008e
        --local-cli yes
        --moddir devtools/dummy_arcadia/typescript/with_simple_bundling
        --nodejs-bin /Users/khoden/.ya/tools/v4/5356355025/node
        --pm-script /Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs
        --pm-type pnpm
        --trace no
        --verbose yes
        build-webpack
        --bundler-config-path /Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js
        --output-file /Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/output.tar
        --output-dirs dev-bundle prod-bundle
        --tsconfigs tsconfig.json
        --vcs-info
        --after-build-js path/to/script.js
        --after-build-args some-args
        --after-build-outdir dist
    """

    # act + assert
    assert __convert_args_to_dict(command_args) == dict(
        # Base
        arcadia_build_root='/Users/khoden/.ya/build/build_root/emev/00008e',
        arcadia_root='/Users/khoden/arcadia',
        moddir='devtools/dummy_arcadia/typescript/with_simple_bundling',
        nodejs_bin='/Users/khoden/.ya/tools/v4/5356355025/node',
        pm_script='/Users/khoden/.ya/tools/v4/4992859933/node_modules/pnpm/dist/pnpm.cjs',
        pm_type='pnpm',
        yatool_prebuilder_path=None,
        env=[],
        # Flags
        local_cli=True,
        bundle=True,
        trace=False,
        verbose=True,
        # Calculated
        bindir='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling',
        bundler_config_path='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling/webpack.config.js',
        curdir='/Users/khoden/arcadia/devtools/dummy_arcadia/typescript/with_simple_bundling',
        node_modules_bundle='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/workspace_node_modules.tar',
        # Command-specific
        bundler_config='webpack.config.js',
        command='build-webpack',
        output_file='/Users/khoden/.ya/build/build_root/emev/00008e/devtools/dummy_arcadia/typescript/with_simple_bundling/output.tar',
        output_dirs=['dev-bundle', 'prod-bundle'],
        tsconfigs=['tsconfig.json'],
        vcs_info=None,
        after_build_js='path/to/script.js',
        after_build_args='some-args',
        after_build_outdir='dist',
    )
