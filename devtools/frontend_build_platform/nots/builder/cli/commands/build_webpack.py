from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    WebpackBuilder,
    WebpackBuilderOptions,
)


def build_webpack_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser("build-webpack", help="build with the Webpack.js")

    subparser.set_defaults(func=build_webpack_func)

    return subparser


@timeit
def build_webpack_func(args: WebpackBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    builder = WebpackBuilder(options=args, ts_config_path=args.tsconfigs[0])
    builder.build()

    # Step 3 - create '<module_name>.output.tar'
    builder.bundle()
