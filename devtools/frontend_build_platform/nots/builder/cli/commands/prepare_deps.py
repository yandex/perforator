from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import prepare_deps, PrepareDepsOptions


def prepare_deps_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser('prepare-deps', help="prepare pnpm-lock.yaml, pnpm-workspace.yaml and tarballs")

    subparser.add_argument('--resource-root', required=False, help="Root location of build node resources")
    subparser.add_argument(
        '--tarballs-store', required=True, help="Relative path to the tarballs store from the CURDIR"
    )
    subparser.add_argument(
        '--ts-proto-auto-deps-path', required=False, help="Arcadia relative path to ts-proto deps module"
    )

    subparser.set_defaults(func=prepare_deps_func)

    return subparser


@timeit
def prepare_deps_func(args: PrepareDepsOptions):
    prepare_deps(args)
