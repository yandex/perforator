from argparse import ArgumentParser
from dataclasses import dataclass

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    TscBuilder,
    TscBuilderOptions,
    TsProtoAutoTscBuilder,
    TsProtoGenerator,
    TsProtoGeneratorOptions,
)
from .build_tsc import add_tsc_parser_args, get_output_dirs


@dataclass
class TsProtoBuilderOptions(TscBuilderOptions, TsProtoGeneratorOptions):
    pass


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
    generator = TsProtoGenerator(options=args)

    # Step 0 - generate package.json and tsconfigs
    generator.generate_auto_package()

    # Step 1 - install node_modules
    create_node_modules(args, original_lf_path=generator.get_auto_deps_lf_path())

    # Step 2 - run generate script
    generator.generate()

    # Step 3 - run build script
    if generator.is_auto_package:
        ts_config_names = ["tsconfig.cjs.json", "tsconfig.esm.json"]
        ts_configs = [TscBuilder.load_ts_config(tc, args.bindir) for tc in ts_config_names]
        for ts_config in ts_configs:
            TsProtoAutoTscBuilder(options=args, ts_config=ts_config).build()
        generator.generate_cjs_pj()
    else:
        ts_configs = [TscBuilder.load_ts_config(tc, args.curdir) for tc in args.tsconfigs]
        for ts_config in ts_configs:
            TscBuilder(options=args, ts_config=ts_config).build()

    # Step 4 - create '<module_name>.output.tar'
    TscBuilder.bundle_dirs(get_output_dirs(ts_configs), args.bindir, args.output_file)
