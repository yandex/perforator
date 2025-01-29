from dataclasses import dataclass
import click
import os
import sys

from devtools.frontend_build_platform.libraries.logging import timeit

from ..models import BuildError, BaseOptions
from ..utils import copy_if_not_exists, dict_to_ts_proto_opt, parse_opt_to_dict, popen, resolve_bin

from .default_ts_proto_opt import DEFAULT_TS_PROTO_OPT


@dataclass
class TsProtoGeneratorOptions(BaseOptions):
    protoc_bin: str
    """Path to protoc binary"""

    proto_paths: list[str]
    """List for --proto-path (-I) argument"""

    proto_srcs: list[str]
    """List of .proto sources"""

    ts_proto_opt: list[str]
    """List for --ts_proto_opt"""


class TsProtoGenerator:
    options: TsProtoGeneratorOptions

    @timeit
    def __init__(self, options: TsProtoGeneratorOptions):
        self.options = options

    @timeit
    def generate(self):
        # We should copy src in advance.
        # This is because we generate src/generated folder that
        # blocks coping src dir in TscBuilder
        self._copy_src_dir()
        # `ts-proto` expects that out dir exits
        # Otherwise it throws "No such file or directory"
        self._make_out_dir()
        self._exec()

    def _copy_src_dir(self):
        curdir_src = os.path.join(self.options.curdir, "src")
        if not os.path.exists(curdir_src):
            return

        bindir_src = os.path.normpath(os.path.join(self.options.bindir, "src"))
        copy_if_not_exists(curdir_src, bindir_src)

    def _get_out_dir(self):
        return os.path.join(self.options.bindir, "src", "generated")

    def _resolve_ts_proto_plugin(self):
        return resolve_bin(self.options.bindir, "ts-proto", "protoc-gen-ts_proto")

    def _make_out_dir(self):
        os.makedirs(self._get_out_dir(), exist_ok=True)

    def _get_ts_proto_opt(self) -> str:
        user_opt = parse_opt_to_dict(self.options.ts_proto_opt)
        final_opt = DEFAULT_TS_PROTO_OPT.copy()
        final_opt.update(user_opt)
        return dict_to_ts_proto_opt(final_opt)

    def _get_exec_args(self) -> list[str]:
        return (
            [
                "--plugin",
                self._resolve_ts_proto_plugin(),
                "--ts_proto_opt",
                self._get_ts_proto_opt(),
                "--ts_proto_out",
                self._get_out_dir(),
            ]
            + [f"-I={p}" for p in self.options.proto_paths]
            + self.options.proto_srcs
        )

    def _get_envs(self) -> dict[str, str]:
        return {"PATH": os.path.dirname(self.options.nodejs_bin)}

    @timeit
    def _exec(self):
        args = [self.options.protoc_bin] + self._get_exec_args()

        if self.options.verbose:
            sys.stderr.write(
                f"cd {click.style(self.options.bindir, fg='cyan')} && {click.style(' '.join(args), fg='magenta')}\n"
            )

        return_code, stdout, stderr = popen(args, env=self._get_envs(), cwd=self.options.bindir)

        if self.options.verbose:
            if stdout:
                sys.stderr.write(f"_exec stdout:\n{click.style(stdout, fg='green')}\n")
            if stderr:
                sys.stderr.write(f"_exec stderr:\n{click.style(stderr, fg='yellow')}\n")

        if return_code != 0:
            raise BuildError(self.options.command, return_code, stdout, stderr)
