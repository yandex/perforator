from argparse import ArgumentParser
from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    TscBuilder,
    TscBuilderOptions,
    TsProtoGenerator,
    TsProtoGeneratorOptions,
)
from .build_tsc import get_output_dirs


@dataclass
class TsProtoBuilderOptions(TscBuilderOptions, TsProtoGeneratorOptions):
    pass


def build_ts_proto_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser(
        "build-ts-proto", help="Build .js and .d.ts from .proto with protoc + ts-proto plugin and tcs"
    )

    subparser.add_argument('--protoc-bin', required=True, help="Path to protoc binary")
    subparser.add_argument('--proto-paths', required=True, nargs='+', help="List for --proto-path (-I) argument")
    subparser.add_argument('--proto-srcs', required=True, nargs='+', help="List of .proto sources")
    subparser.add_argument('--ts-proto-opt', default=[], action='append', help="List for --ts_proto_opt")

    subparser.set_defaults(func=build_ts_proto_func)

    return subparser


@timeit
def build_ts_proto_func(args: TsProtoBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run generate script
    TsProtoGenerator(options=args).generate()

    # Step 3 - run build script
    ts_configs = [TscBuilder.load_ts_config(tc, args.curdir) for tc in args.tsconfigs]

    for ts_config in ts_configs:
        TscBuilder(options=args, ts_config=ts_config).build()

    # Step 4 - create '<module_name>.output.tar'
    TscBuilder.bundle_dirs(get_output_dirs(ts_configs), args.bindir, args.output_file)
