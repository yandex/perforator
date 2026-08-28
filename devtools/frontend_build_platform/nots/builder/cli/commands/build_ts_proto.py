import os
from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import TsLibraryBuilderOptions
from devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator import (
    make_ts_proto_build_command,
)
from devtools.frontend_build_platform.nots.builder.api.utils import extract_output_tar
from .build_library import build_library_func
from .build_tsc import add_tsc_parser_args


class TsProtoBuilderOptions(TsLibraryBuilderOptions):
    protoc_bin: str
    proto_paths: list[str]
    proto_srcs: list[str]
    ts_proto_opt: list[str]
    tsconfigs: list[str]
    auto_package_name: str | None
    auto_deps_path: str | None


def build_ts_proto_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser(
        "build-ts-proto", help="Build .js and .d.ts from .proto with protoc + ts-proto plugin and tcs"
    )

    add_tsc_parser_args(subparser)

    subparser.add_argument('--protoc-bin', required=True, help="Path to protoc binary")
    subparser.add_argument('--proto-paths', required=True, nargs='+', help="List for --proto-path (-I) argument")
    subparser.add_argument('--proto-srcs', required=True, nargs='+', help="List of .proto sources")
    subparser.add_argument('--ts-proto-opt', default=[], action='append', help="List for --ts_proto_opt")
    subparser.add_argument('--auto-package-name', required=False, help="Name for TS_PROTO_AUTO package")
    subparser.add_argument(
        '--auto-deps-path', required=False, help="Arcadia relative path to TS_PROTO_AUTO deps module"
    )

    subparser.set_defaults(func=build_ts_proto_func)

    return subparser


@timeit
def build_ts_proto_func(args: TsProtoBuilderOptions):
    is_auto_package = args.auto_package_name is not None and args.auto_deps_path is not None
    if is_auto_package:
        extract_output_tar(os.path.join(args.arcadia_build_root, args.auto_deps_path))
    args.env.extend(
        [
            'PROTOC={}'.format(args.protoc_bin),
            'ARCADIA_ROOT={}'.format(args.arcadia_root),
            'ARCADIA_BUILD_ROOT={}'.format(args.arcadia_build_root),
        ]
    )
    args.build_command = make_ts_proto_build_command(
        args.arcadia_root,
        args.arcadia_build_root,
        args.curdir,
        args.proto_paths,
        args.proto_srcs,
        args.ts_proto_opt,
        args.tsconfigs,
        is_auto_package,
        args.auto_deps_path,
    )
    args.build_script = 'nots:build'
    args.outputs = ['build']
    args.exclude_globs = ['.npmrc']
    return build_library_func(args)
