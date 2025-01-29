from argparse import ArgumentParser
from dataclasses import dataclass

from build.plugins.lib.nots.package_manager import (
    PackageJson,
    utils as pm_utils,
)
from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import BaseOptions, create_node_modules


@dataclass
class PackageBuilderOptions(BaseOptions):
    pass


def build_package_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser(
        'build-package', help="build package (actually just create node_modules directory)"
    )

    subparser.set_defaults(func=build_package_func)

    return subparser


@timeit
def build_package_func(args: PackageBuilderOptions):
    pj = PackageJson.load(pm_utils.build_pj_path(args.curdir))
    if pj.has_dependencies():
        create_node_modules(args)
