from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import PackageBuilder, PackageBuilderOptions


def add_package_builder_args(subparser: ArgumentParser) -> ArgumentParser:
    """Add command-specific arguments for build-library"""
    subparser.add_argument(
        '--outputs', required=False, nargs='+', default=[], help="List of output directories for the build"
    )

    subparser.add_argument(
        '--exclude-globs', required=False, nargs='*', default=[], help="Glob patterns to exclude when copying files"
    )

    return subparser


def build_package_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser(
        'build-package', help="build package (actually just create node_modules directory)"
    )

    add_package_builder_args(subparser)
    subparser.set_defaults(func=build_package_func)

    return subparser


@timeit
def build_package_func(args: PackageBuilderOptions):
    builder = PackageBuilder(options=args)
    builder.build()
    builder.bundle()

    return []
