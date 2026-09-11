import os
import shlex

from build.plugins.lib.nots.package_manager import PackageJson, utils as pm_utils

from ..utils import dict_to_ts_proto_opt, parse_opt_to_dict

from .default_ts_proto_opt import DEFAULT_TS_PROTO_OPT, DEFAULT_TS_PROTO_AUTO_OPT


def generate_ts_proto_auto_package(
    build_root: str,
    bindir: str,
    moddir: str,
    auto_package_name: str,
    auto_deps_path: str,
) -> None:
    auto_deps_build_path = os.path.join(build_root, auto_deps_path)
    deps_pj = PackageJson.load(pm_utils.build_pj_path(auto_deps_build_path))
    pj = PackageJson(pm_utils.build_pj_path(bindir))
    gen_name = moddir.replace("/", "-")
    pj.data = {
        "name": auto_package_name.replace("*", gen_name),
        "version": "0.0.0",
        "type": "module",
        "files": ["build", "pnpm-lock.yaml"],
        "repository": {"type": "arc", "directory": moddir},
        "dependencies": deps_pj.data.get("dependencies", {}),
        "devDependencies": deps_pj.data.get("devDependencies", {}),
        "exports": {
            "./*": {
                "import": os.path.join(".", "build", "esm", "generated", moddir, "*.js"),
                "require": os.path.join(".", "build", "cjs", "generated", moddir, "*.js"),
                "types": os.path.join(".", "build", "types", "generated", moddir, "*.d.ts"),
                "default": os.path.join(".", "build", "esm", "generated", moddir, "*.js"),
            },
            "./generated/*": {
                "import": os.path.join(".", "build", "esm", "generated", "*.js"),
                "require": os.path.join(".", "build", "cjs", "generated", "*.js"),
                "types": os.path.join(".", "build", "types", "generated", "*.d.ts"),
                "default": os.path.join(".", "build", "esm", "generated", "*.js"),
            },
        },
    }
    pj.write()


def _portable_path(path: str, arcadia_root: str, build_root: str, curdir: str) -> str:
    if not os.path.isabs(path):
        return os.path.normpath(path)
    for root, variable in ((curdir, None), (arcadia_root, "$ARCADIA_ROOT"), (build_root, "$ARCADIA_BUILD_ROOT")):
        try:
            relative = os.path.relpath(path, root)
        except ValueError:
            continue
        if relative != os.pardir and not relative.startswith(os.pardir + os.sep):
            return relative if variable is None else os.path.join(variable, relative)
    return path


def _shell_arg(value: str) -> str:
    if "$ARCADIA_ROOT" in value or "$ARCADIA_BUILD_ROOT" in value:
        return '"{}"'.format(value.replace('"', '\\"'))
    return shlex.quote(value)


def _portable_proto_source(path: str, arcadia_root: str, build_root: str) -> str:
    if not os.path.isabs(path):
        return os.path.normpath(path)
    for root, variable in ((arcadia_root, "$ARCADIA_ROOT"), (build_root, "$ARCADIA_BUILD_ROOT")):
        relative = os.path.relpath(path, root)
        if relative != os.pardir and not relative.startswith(os.pardir + os.sep):
            return os.path.join(variable, relative)
    return path


def make_ts_proto_build_command(
    arcadia_root: str,
    build_root: str,
    curdir: str,
    proto_paths: list[str],
    proto_srcs: list[str],
    ts_proto_opt: list[str],
    tsconfigs: list[str],
    is_auto_package: bool,
    auto_deps_path: str | None,
) -> str:
    user_opt = parse_opt_to_dict(ts_proto_opt)
    final_opt = DEFAULT_TS_PROTO_OPT.copy()
    if is_auto_package:
        final_opt.update(DEFAULT_TS_PROTO_AUTO_OPT)
    final_opt.update(user_opt)

    args = [
        '"$PROTOC"',
        "--plugin",
        "node_modules/.bin/protoc-gen-ts_proto",
        "--ts_proto_opt",
        dict_to_ts_proto_opt(final_opt),
        "--ts_proto_out",
        "src/generated",
    ]
    args.extend("-I={}".format(_portable_path(path, arcadia_root, build_root, curdir)) for path in proto_paths)
    args.extend(_portable_proto_source(path, arcadia_root, build_root) for path in proto_srcs)
    protoc_command = " ".join(arg if arg == '"$PROTOC"' else _shell_arg(arg) for arg in args)
    commands = []
    if is_auto_package:
        assert auto_deps_path is not None
        for tsconfig in ["tsconfig.json", "tsconfig.cjs.json", "tsconfig.esm.json"]:
            source = os.path.join("$ARCADIA_BUILD_ROOT", auto_deps_path, tsconfig)
            commands.append(
                "node -e \"require('fs').copyFileSync(process.argv[1],process.argv[2])\" {} {}".format(
                    _shell_arg(source), shlex.quote(tsconfig)
                )
            )
        tsconfigs = ["tsconfig.cjs.json", "tsconfig.esm.json"]
    commands.extend(['node -e "require(\'fs\').mkdirSync(\'src/generated\', {recursive:true})"', protoc_command])
    commands.extend(
        "node_modules/.bin/tsc --project {} --incremental false --composite false --pretty".format(
            shlex.quote(tsconfig)
        )
        for tsconfig in tsconfigs
    )
    if is_auto_package:
        commands.append(
            "node -e \"const fs=require('fs');fs.mkdirSync('build/cjs',{recursive:true});"
            "fs.writeFileSync('build/cjs/package.json','{\\\"type\\\":\\\"commonjs\\\"}\\n')\""
        )
    return " && ".join(commands)
