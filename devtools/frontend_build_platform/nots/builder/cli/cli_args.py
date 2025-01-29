import argparse
import os
from argparse import ArgumentParser

import sys

from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
)
from devtools.frontend_build_platform.libraries.logging import timeit
from .commands.build_next import build_next_parser, NextBuilderOptions
from .commands.build_package import build_package_parser, PackageBuilderOptions
from .commands.build_ts_proto import build_ts_proto_parser, TsProtoBuilderOptions
from .commands.build_tsc import build_tsc_parser, TscBuilderOptions
from .commands.build_vite import build_vite_parser, ViteBuilderOptions
from .commands.build_webpack import build_webpack_parser, WebpackBuilderOptions
from .commands.create_node_modules import create_node_modules_parser, CreateNodeModulesOptions
from .commands.prepare_deps import prepare_deps_parser, PrepareDepsOptions
from .models import YesNoAction


@timeit
def register_base_args(parser: ArgumentParser) -> None:
    # Arcadia paths. See https://docs.yandex-team.ru/ya-make/manual/common/vars
    parser.add_argument('--arcadia-root', required=True, help="Absolute path to the root of Arcadia (mount point)")
    parser.add_argument('--arcadia-build-root', required=True, help="Absolute path for the temporary build directory")
    parser.add_argument('--moddir', required=True, help="Relative path to the target from the root of Arcadia")

    # Essential
    parser.add_argument('--nodejs-bin', required=True, help="Path to the 'node' executable file")
    parser.add_argument('--pm-script', required=True, help="Path to package manager script to run `install` command")
    parser.add_argument('--pm-type', required=True, help="Type of package manager (pnpm or npm)")
    parser.add_argument(
        '--yatool-prebuilder-path', required=False, help="Path to `@yatool/prebuilder` script, if it needed"
    )

    # Flags
    parser.add_argument(
        '--local-cli', action=YesNoAction, default=False, help="Is run locally (from `nots`) or on the distbuild"
    )
    parser.add_argument('--bundle', action=YesNoAction, default=True, help="Bundle the result into a tar archive")

    parser.add_argument(
        '--trace',
        action=YesNoAction,
        default=False,
        help="Add to the output.tar *.trace file (Trace Events Format, Chrome DevTools compatible)",
    )
    parser.add_argument('--verbose', action=YesNoAction, default=False, help="Use logging")


@timeit
def __with_bundlers_options(parser: ArgumentParser) -> ArgumentParser:
    """Common arguments for bundlers"""

    parser.add_argument('--output-dirs', required=True, nargs='+', help="Defined output directories for the bundler")
    parser.add_argument(
        '--bundler-config-path',
        required=True,
        help="Path to the bundler config (vite.config.ts, webpack.config.js, etc...)",
    )

    return parser


@timeit
def __with_builders_options(parser: ArgumentParser):
    """Common arguments for all builders"""

    parser.add_argument(
        '--output-file', required=True, help="Absolute path to output.tar, expected to be generated during build"
    )

    parser.add_argument(
        '--vcs-info',
        required=False,
        nargs='?',
        default='',
        help="Path to the VCS_INFO_FILE, see https://docs.yandex-team.ru/ya-make/manual/package/macros#vcs_info_file",
    )

    parser.add_argument(
        '--tsconfigs',
        required=True,
        nargs='+',
        help="List of the tsconfigs (multiple tsconfigs are supported only in `build-tsc` command)",
    )

    parser.add_argument(
        "--env",
        default=[],
        required=False,
        action="append",
        help="Environment variable in VAR format, can be set many times",
    )

    return parser


@timeit
def register_builders(subparsers):
    prepare_deps_parser(subparsers)

    # Only build node_modules
    build_package_parser(subparsers)
    create_node_modules_parser(subparsers)

    # TS transpilers
    __with_builders_options(build_tsc_parser(subparsers))
    __with_builders_options(build_ts_proto_parser(subparsers))

    # Bundlers
    __with_builders_options(__with_bundlers_options(build_next_parser(subparsers)))
    __with_builders_options(__with_bundlers_options(build_vite_parser(subparsers)))
    __with_builders_options(__with_bundlers_options(build_webpack_parser(subparsers)))


@timeit
def get_args_parser():
    parser = argparse.ArgumentParser(prog='nots_builder')

    register_base_args(parser)

    subparsers = parser.add_subparsers(title="commands", dest='command')

    register_builders(subparsers)

    return parser


AllOptions = (
    CreateNodeModulesOptions
    | NextBuilderOptions
    | PackageBuilderOptions
    | TsProtoBuilderOptions
    | TscBuilderOptions
    | ViteBuilderOptions
    | WebpackBuilderOptions
    | PrepareDepsOptions
)


@timeit
def parse_args(parser, custom_args: list[str] = None) -> AllOptions:
    args: AllOptions = parser.parse_args(custom_args or sys.argv[1:])

    # Calculated arguments
    curdir = os.path.join(args.arcadia_root, args.moddir)
    setattr(args, 'curdir', curdir)

    bindir = os.path.join(args.arcadia_build_root, args.moddir)
    setattr(args, 'bindir', bindir)

    node_modules_bundle = os.path.join(bindir, pm_constants.NODE_MODULES_WORKSPACE_BUNDLE_FILENAME)
    setattr(args, 'node_modules_bundle', node_modules_bundle)

    if hasattr(args, 'bundler_config_path'):
        bundler_config = args.bundler_config_path.removeprefix(args.curdir).strip('/')
        setattr(args, 'bundler_config', bundler_config)

    return args
