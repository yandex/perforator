from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    NextBuilder,
    NextBuilderOptions,
)


def build_next_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser("build-next", help="build with the Next.js")

    subparser.add_argument(
        '--ts-next-command',
        required=False,
        default="build",
        help="Use a specific next build command",
    )

    subparser.set_defaults(func=build_next_func)

    return subparser


@timeit
def build_next_func(args: NextBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    builder = NextBuilder(options=args, ts_config_path=args.tsconfigs[0])
    builder.build()

    # Step 3 - create '<module_name>.output.tar'
    builder.bundle()
