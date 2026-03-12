import os.path
import json
import sys
import uuid
from datetime import datetime, UTC
from pprint import pformat

from build.plugins.lib.nots.package_manager import constants as pm_constants, utils as pm_utils
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


def __add_uuid_for_output(bindir: str, output_file: str, outputs: list[str] | None):
    uuid_file_name = f'{bindir}/{pm_constants.OUTPUT_TAR_UUID_FILENAME}'

    with open(uuid_file_name, 'w') as f:
        output_filename = os.path.basename(output_file)
        uuid_str = uuid.uuid1().hex
        timestamp = datetime.now(UTC).isoformat()

        f.write(f"{output_filename}: {uuid_str} - {timestamp}")
        f.write("\noutputs: ")
        json.dump(list(set(outputs)), f)


def _postprocess_output(args: AllOptions, outputs: list[str]) -> None:
    output_file = getattr(args, 'output_file', args.node_modules_bundle)
    outputs.extend(getattr(args, 'outputs', []))
    outputs.extend(getattr(args, 'output_dirs', []))
    after_build_outdir = getattr(args, 'after_build_outdir', [])
    if after_build_outdir:
        outputs.append(after_build_outdir)

    if output_file and os.path.isfile(output_file):
        if output_file != args.node_modules_bundle:
            __add_uuid_for_output(args.bindir, output_file, outputs)


# @timeit тут нельзя, т.к. измерение включается внутри
def main():
    args_parser = get_args_parser()
    args: AllOptions = parse_args(args_parser)

    if args.verbose:
        sys.stderr.write(
            f"Raw command string:\n\n{' '.join(sys.argv)}\n\nParsed arguments:\n\n{pformat(vars(args))}\n\n"
        )

    if args.local_cli:
        timeit_options.enable(silent=True, use_dumper=True, use_stderr=True)

    init_logging(args.verbose)

    output_dirs = args.func(args)

    _postprocess_output(args, output_dirs)

    if args.local_cli:
        dir_name = pm_utils.build_traces_store_path(args.arcadia_build_root, args.moddir)
        trace_file = os.path.join(dir_name, f'{args.command}.builder.trace.json')
        timeit_options.dump_trace(trace_file, otherData=dict(moddir=args.moddir))
        if args.verbose:
            sys.stderr.write(f"Trace file: {trace_file}\n")


if __name__ == "__main__":
    main()
