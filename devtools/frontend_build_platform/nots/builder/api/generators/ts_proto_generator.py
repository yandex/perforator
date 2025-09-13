from dataclasses import dataclass
import click
import os
import sys

from devtools.frontend_build_platform.libraries.logging import timeit

from build.plugins.lib.nots.package_manager import PackageJson, utils as pm_utils


from ..models import BuildError, BaseOptions
from ..utils import copy_if_not_exists, dict_to_ts_proto_opt, parse_opt_to_dict, popen, resolve_bin

from .default_ts_proto_opt import DEFAULT_TS_PROTO_OPT, DEFAULT_TS_PROTO_AUTO_OPT


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

    auto_package_name: str | None
    """Name for TS_PROTO_AUTO package"""

    auto_deps_path: str | None
    """Arcadia relative path to TS_PROTO_AUTO deps module"""


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

    @property
    def is_auto_package(self):
        return self.options.auto_package_name is not None and self.options.auto_deps_path is not None

    @timeit
    def generate_auto_package(self):
        if not self.is_auto_package:
            return

        auto_deps_build_path = os.path.join(self.options.arcadia_build_root, self.options.auto_deps_path)
        deps_pj = PackageJson.load(pm_utils.build_pj_path(auto_deps_build_path))
        pj = PackageJson(pm_utils.build_pj_path(self.options.bindir))
        gen_name = self.options.moddir.replace("/", "-")
        pj.data = {
            "name": self.options.auto_package_name.replace("*", gen_name),
            "version": "0.0.0",
            "type": "module",
            "files": ["build/"],
            "repository": {"type": "arc", "directory": self.options.moddir},
            "dependencies": deps_pj.data.get("dependencies", {}),
            "devDependencies": deps_pj.data.get("devDependencies", {}),
            "exports": {
                "./*": {
                    "import": os.path.join(".", "build", "esm", "generated", self.options.moddir, "*.js"),
                    "require": os.path.join(".", "build", "cjs", "generated", self.options.moddir, "*.js"),
                    "types": os.path.join(".", "build", "types", "generated", self.options.moddir, "*.d.ts"),
                    "default": os.path.join(".", "build", "esm", "generated", self.options.moddir, "*.js"),
                }
            },
        }
        pj.write()

        tsconfigs = ["tsconfig.json", "tsconfig.cjs.json", "tsconfig.esm.json"]
        for tsconfig in tsconfigs:
            copy_if_not_exists(
                os.path.join(auto_deps_build_path, tsconfig), os.path.join(self.options.bindir, tsconfig)
            )

    def get_auto_deps_lf_path(self) -> str | None:
        if not self.is_auto_package:
            return None
        return os.path.join(self.options.arcadia_build_root, self.options.auto_deps_path, "pnpm-lock.yaml")

    @timeit
    def generate_cjs_pj(self):
        cjs_outdir = os.path.join(self.options.bindir, "build", "cjs")
        if os.path.exists(cjs_outdir):
            pj = PackageJson(pm_utils.build_pj_path(cjs_outdir))
            pj.data = {"type": "commonjs"}
            pj.write()

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

        if self.is_auto_package:
            final_opt.update(DEFAULT_TS_PROTO_AUTO_OPT)

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
