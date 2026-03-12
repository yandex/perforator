from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import create_node_modules, PackageBuilder, PackageBuilderOptions


def build_package_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser(
        'build-package', help="build package (actually just create node_modules directory)"
    )

    subparser.set_defaults(func=build_package_func)

    return subparser


@timeit
def build_package_func(args: PackageBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    builder = PackageBuilder(options=args)
    builder.build()

    # Step 3 - create 'output.tar'
    builder.bundle()

    return []
