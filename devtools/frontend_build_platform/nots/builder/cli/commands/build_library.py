from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    TsLibraryBuilder,
    TsLibraryBuilderOptions,
)


def add_library_builder_args(subparser: ArgumentParser) -> ArgumentParser:
    """Add command-specific arguments for build-library"""
    subparser.add_argument('--outputs', required=True, nargs='+', help="List of output directories for the build")

    subparser.add_argument('--build-script', required=True, help="Name of the npm script from package.json to execute")

    subparser.add_argument(
        '--exclude-globs', required=False, nargs='*', default=[], help="Glob patterns to exclude when copying files"
    )

    return subparser


def build_library_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser('build-library', help="build TypeScript library using TsLibraryBuilder")

    add_library_builder_args(subparser)
    subparser.set_defaults(func=build_library_func)

    return subparser


@timeit
def build_library_func(args: TsLibraryBuilderOptions):
    # Step 1 - install node_modules
    # create_node_modules(args)

    # Step 2 - run build script
    builder = TsLibraryBuilder(options=args)
    try:
        builder.build()

        # Step 3 - create 'output.tar'
        builder.bundle()

        return builder.options.outputs
    finally:
        builder.cleanup()
