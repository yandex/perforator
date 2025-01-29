import os.path
import sys
import tarfile
import uuid
from datetime import datetime
from pprint import pformat

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


def __add_uuid_for_output(output_file: str):
    uuid_file_name = f'{output_file}.uuid'

    with open(uuid_file_name, 'w') as f:
        output_filename = os.path.basename(output_file)
        uuid_str = uuid.uuid1().hex
        timestamp = datetime.utcnow().isoformat()

        f.write(f"{output_filename}: {uuid_str} - {timestamp}")


def __add_tracing_to_output(dir_name: str, output_file: str):
    traces_dir = '.traces'
    traces_dir_path = os.path.join(dir_name, traces_dir)
    timeit_options.dump_trace(os.path.join(traces_dir_path, 'builder.trace.json'))

    with tarfile.open(output_file, "a") as tf:
        tf.add(traces_dir_path, arcname=traces_dir)


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

    # There is no output.tar for TS_PACKAGE module without dependencies
    if os.path.isfile(output_file):
        if args.trace:
            __add_tracing_to_output(args.bindir, output_file)

        if output_file != args.node_modules_bundle:
            __add_uuid_for_output(output_file)


if __name__ == "__main__":
    main()
