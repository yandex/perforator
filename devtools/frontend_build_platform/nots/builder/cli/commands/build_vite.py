from argparse import ArgumentParser
import sys

from devtools.frontend_build_platform.libraries.logging import timeit
from devtools.frontend_build_platform.nots.builder.api import (
    create_node_modules,
    ViteBuilder,
    ViteBuilderOptions,
)


def build_vite_parser(subparsers) -> ArgumentParser:
    subparser = subparsers.add_parser("build-vite", help="build with the Vite.js")

    subparser.set_defaults(func=build_vite_func)

    return subparser


@timeit
def build_vite_func(args: ViteBuilderOptions):
    # Step 1 - install node_modules
    create_node_modules(args)

    # Step 2 - run build script
    for i, bundler_config_path in enumerate(args.bundler_configs):
        output_dir = args.output_dirs[i]
        ts_config_path = args.tsconfigs[0]  # The first only!

        if args.verbose:
            print(f"Build vite with config: {bundler_config_path} and output dir: {output_dir}", file=sys.stderr)

        builder = ViteBuilder(
            options=args, bundler_config_path=bundler_config_path, output_dir=output_dir, ts_config_path=ts_config_path
        )
        builder.build()

    # Step 2.5 - run after build script
    out_dirs = args.output_dirs.copy()

    if args.with_after_build:
        bundler_config_path = args.bundler_configs[0]
        output_dir = args.output_dirs[0]
        ts_config_path = args.tsconfigs[0]

        builder = ViteBuilder(
            options=args, bundler_config_path=bundler_config_path, output_dir=output_dir, ts_config_path=ts_config_path
        )
        builder.run_javascript_after_build()
        if args.after_build_outdir:
            out_dirs.append(args.after_build_outdir)

    # Step 3 - create 'output.tar'
    builder.bundle()

    return out_dirs
