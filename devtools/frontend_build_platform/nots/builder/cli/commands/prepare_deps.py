import argparse
from argparse import ArgumentParser

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    prepare_deps,
    prepare_node_modules_layer,
    PrepareDepsOptions,
)

from ...api.utils import normalize_output_prefix


def prepare_deps_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser('prepare-deps', help="prepare pnpm-lock.yaml, pnpm-workspace.yaml and tarballs")

    subparser.add_argument('--resource-root', required=False, help="Root location of build node resources")
    subparser.add_argument(
        '--tarballs-platform',
        nargs=3,
        metavar=('OS', 'CPU', 'LIBC'),
        help="Install action platform for optional tarball selection",
    )
    subparser.add_argument(
        '--tarballs-store', required=True, help="Relative path to the tarballs store from the CURDIR"
    )
    subparser.add_argument(
        '--ts-proto-auto-deps-path', required=False, help="Arcadia relative path to ts-proto deps module"
    )
    subparser.add_argument('--ts-proto-auto-package-name', required=False, help="Generated TS_PROTO package name")
    subparser.add_argument(
        '--ts-proto-workspace-peers',
        dest='proto_peers',
        default=[],
        nargs='*',
        help="Arcadia-relative built TS_PROTO peers to include in generated workspace metadata",
    )

    subparser.add_argument('--output-prefix', type=normalize_output_prefix, default=argparse.SUPPRESS)
    subparser.set_defaults(func=prepare_deps_func)

    return subparser


@timeit
def prepare_deps_func(args: PrepareDepsOptions):
    prepare_deps(args)
    if args.hermetic_node_modules:
        prepare_node_modules_layer(args)

    return []
