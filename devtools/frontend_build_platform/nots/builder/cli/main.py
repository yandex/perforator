import os.path
import sys
import tarfile
import uuid
from datetime import datetime, UTC
from pprint import pformat

import library.python.fs

from build.plugins.lib.nots.package_manager import (
    constants as pm_constants,
)
from devtools.frontend_build_platform.libraries.logging import init_logging, timeit_options
from devtools.frontend_build_platform.nots.builder.api import BuildError
from devtools.frontend_build_platform.nots.builder.cli.cli_args import AllOptions, get_args_parser, parse_args


def on_crash(exctype, value, traceback):
    if issubclass(exctype, BuildError):
        print(str(value), file=sys.stderr)
        sys.exit(value.code)
    else:
        sys.__excepthook__(exctype, value, traceback)


sys.excepthook = on_crash


def __add_uuid_for_output(bindir: str, output_file: str):
    uuid_file_name = f'{bindir}/{pm_constants.OUTPUT_TAR_UUID_FILENAME}'

    with open(uuid_file_name, 'w') as f:
        output_filename = os.path.basename(output_file)
        uuid_str = uuid.uuid1().hex
        timestamp = datetime.now(UTC).isoformat()

        f.write(f"{output_filename}: {uuid_str} - {timestamp}")


def __add_tracing_to_output(dir_name: str, output_file: str):
    traces_dir = '.traces'
    traces_dir_path = os.path.join(dir_name, traces_dir)
    timeit_options.dump_trace(os.path.join(traces_dir_path, 'builder.trace.json'))

    with tarfile.open(output_file, "a") as tf:
        tf.add(traces_dir_path, arcname=traces_dir)


def __produce_old_output_tar(output_file: str):
    # TODO FBP-1978 (remove the function)
    old_output_tar_file = os.path.join(os.path.dirname(output_file), 'output.tar')

    library.python.fs.hardlink_or_copy(output_file, old_output_tar_file)


# @timeit тут нельзя, т.к. измерение включается внутри
def main():
    args_parser = get_args_parser()
    args: AllOptions = parse_args(args_parser)

    if args.verbose:
        sys.stderr.write(
            f"Raw command string:\n\n{' '.join(sys.argv)}\n\nParsed arguments:\n\n{pformat(vars(args))}\n\n"
        )

    if args.trace:
        timeit_options.enable(silent=True, use_dumper=True, use_stderr=True)

    init_logging(args.verbose)

    args.func(args)

    output_file = getattr(args, 'output_file', args.node_modules_bundle)

    # There is no <module_name>.output.tar for TS_PACKAGE module
    if os.path.isfile(output_file):
        if args.trace:
            __add_tracing_to_output(args.bindir, output_file)

        if output_file != args.node_modules_bundle:
            # TODO FBP-1978 (remove call)
            __produce_old_output_tar(output_file)

            __add_uuid_for_output(args.bindir, output_file)


if __name__ == "__main__":
    main()
