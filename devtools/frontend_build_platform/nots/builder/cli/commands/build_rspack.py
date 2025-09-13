from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    RspackBuilder,
    RspackBuilderOptions,
)


def build_rspack_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser("build-rspack", help="build with the Rspack.js")

    subparser.set_defaults(func=build_rspack_func)

    return subparser


@timeit
def build_rspack_func(args: RspackBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    builder = RspackBuilder(options=args, ts_config_path=args.tsconfigs[0])
    builder.build()

    # Step 3 - create '<module_name>.output.tar'
    builder.bundle()
