from dataclasses import dataclass
import json
import os
import shlex
from pathlib import Path

from devtools.frontend_build_platform.libraries.logging import timeit
from build.plugins.lib.nots.package_manager import (
    Lockfile,
    PackageJson,
    constants as pm_constants,
    utils as pm_utils,
)

from ..models import BaseOptions
from ..package_manager import PackageManager
from ..pnpm_workspace import PnpmWorkspace
from ..utils import (
    dict_to_ts_proto_opt,
    extract_all_output_tars,
    parse_opt_to_dict,
)

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
        "files": ["build", "pnpm-lock.yaml", "ya.make"],
        "repository": {"type": "arc", "directory": moddir},
        "nots": {"tsProtoAuto": True},
        "dependencies": deps_pj.data.get("dependencies", {}),
        "devDependencies": deps_pj.data.get("devDependencies", {}),
        "exports": {
            "./*": {
                "import": os.path.join(".", "build", "esm", "generated", moddir, "*.js"),
                "require": {
                    "types": os.path.join(".", "build", "cjs", "generated", moddir, "*.d.ts"),
                    "default": os.path.join(".", "build", "cjs", "generated", moddir, "*.js"),
                },
                "types": os.path.join(".", "build", "types", "generated", moddir, "*.d.ts"),
                "default": os.path.join(".", "build", "esm", "generated", moddir, "*.js"),
            },
            "./generated/*": {
                "import": os.path.join(".", "build", "esm", "generated", "*.js"),
                "require": {
                    "types": os.path.join(".", "build", "cjs", "generated", "*.d.ts"),
                    "default": os.path.join(".", "build", "cjs", "generated", "*.js"),
                },
                "types": os.path.join(".", "build", "types", "generated", "*.d.ts"),
                "default": os.path.join(".", "build", "esm", "generated", "*.js"),
            },
        },
        "typesVersions": {
            "*": {
                "*": [os.path.join("build", "types", "generated", moddir, "*")],
                "generated/*": [os.path.join("build", "types", "generated", "*")],
            }
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
    import_mappings: dict[str, str] | None = None,
    peer_reexports_file: str | None = None,
) -> str:
    user_opt = parse_opt_to_dict(ts_proto_opt)
    final_opt = DEFAULT_TS_PROTO_OPT.copy()
    if is_auto_package:
        final_opt.update(DEFAULT_TS_PROTO_AUTO_OPT)
    final_opt.update(user_opt)
    for proto_path, package_import in (import_mappings or {}).items():
        final_opt["M" + proto_path] = package_import

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
    if peer_reexports_file:
        commands.append(
            "node -e "
            + shlex.quote(
                "const fs=require('fs'),path=require('path');"
                "for(const [proto,target] of Object.entries(JSON.parse(fs.readFileSync(process.argv[1],'utf8')))){"
                "const file=path.join('src/generated',proto.replace(/\\.proto$/,'.ts'));"
                "if(!fs.existsSync(file)){fs.mkdirSync(path.dirname(file),{recursive:true});"
                "fs.writeFileSync(file,'export * from '+JSON.stringify(target)+';\\n');}}"
            )
            + " "
            + shlex.quote(peer_reexports_file)
        )
    commands.extend(
        "node_modules/.bin/tsc --project {} --incremental false --composite false --pretty{}".format(
            shlex.quote(tsconfig),
            " --declaration" if is_auto_package and tsconfig == "tsconfig.cjs.json" else "",
        )
        for tsconfig in tsconfigs
    )
    # Explicit package scopes keep generated JS independent of the root manifest.
    # Custom packages may generate only one of these directories.
    commands.append(
        "node -e "
        + shlex.quote(
            "const fs=require('fs');"
            "for(const [dir,type] of [['build/cjs','commonjs'],['build/esm','module']]){"
            "if(fs.existsSync(dir)){"
            "fs.writeFileSync(dir+'/package.json',JSON.stringify({type})+'\\n');"
            "}}"
        )
    )
    return " && ".join(commands)


@dataclass
class TsProtoGeneratorOptions(BaseOptions):
    proto_paths: list[str]
    """List for --proto-path (-I) argument"""

    proto_peers: list[str]
    """Arcadia-relative PEERDIR candidates; built TS_PROTO peers have output.tar.uuid"""


class TsProtoGenerator:
    options: TsProtoGeneratorOptions

    @timeit
    def __init__(self, options: TsProtoGeneratorOptions):
        self.options = options

    @timeit
    def prepare_peer_libraries(self) -> None:
        """Materialize built TS_PROTO peers and add them to the local pnpm workspace."""
        self._reset_peer_metadata()
        peer_dirs = self._get_ts_proto_peer_dirs()
        if not peer_dirs:
            return

        for peer_dir in peer_dirs:
            extract_all_output_tars(peer_dir, build_root=self.options.arcadia_build_root)

        package_json_path = pm_utils.build_pj_path(self.options.bindir)
        peer_dependencies = self._get_peer_package_dependencies(peer_dirs)
        self._update_package_json(package_json_path, peer_dependencies)

        workspace_path = pm_utils.build_ws_config_path(self.options.bindir)
        workspace = PnpmWorkspace.load(workspace_path)
        for peer_dir in peer_dirs:
            peer_workspace_path = pm_utils.build_ws_config_path(peer_dir)
            if os.path.isfile(peer_workspace_path):
                workspace.merge(PnpmWorkspace.load(peer_workspace_path))
            workspace.packages.add(os.path.relpath(peer_dir, self.options.bindir))
        workspace.write()

        self._update_peer_lockfile(peer_dirs, peer_dependencies)

    def _reset_peer_metadata(self) -> None:
        """Drop workspace metadata inherited from another generated TS_PROTO graph."""
        package_json = PackageJson.load(pm_utils.build_pj_path(self.options.bindir))
        lockfile = Lockfile.load(pm_utils.build_lockfile_path(self.options.bindir))

        importers = lockfile.get_importers()
        root_importer = importers.setdefault(".", {})
        for dependency_key in (
            PackageJson.DEP_KEY,
            PackageJson.DEV_DEP_KEY,
            PackageJson.PEER_DEP_KEY,
            PackageJson.OPT_DEP_KEY,
        ):
            declared_dependencies = package_json.data.get(dependency_key, {})
            if dependency_key in root_importer:
                root_importer[dependency_key] = {
                    name: metadata
                    for name, metadata in root_importer[dependency_key].items()
                    if name in declared_dependencies
                }
        lockfile.data["importers"] = {".": root_importer}

        packages = lockfile.data.get("packages", {})
        workspace_package_keys = {
            package_key
            for package_key, metadata in packages.items()
            if metadata.get("resolution", {}).get("type") == "directory"
        }
        for package_key in workspace_package_keys:
            packages.pop(package_key, None)
            lockfile.data.get("snapshots", {}).pop(package_key, None)

        lockfile.write()

    @staticmethod
    def _update_package_json(package_json_path: str, peer_dependencies: dict[str, str]) -> None:
        package_json = PackageJson.load(package_json_path)
        dependencies = package_json.data.setdefault(PackageJson.DEP_KEY, {})
        dependencies.update(peer_dependencies)
        files = package_json.data.setdefault("files", [])
        if "ya.make" not in files:
            files.append("ya.make")
        package_json.write()

    def _get_ts_proto_peer_dirs(self) -> list[str]:
        peer_dirs = []
        for peer in self.options.proto_peers:
            peer_dir = peer
            if not os.path.isabs(peer_dir):
                peer_dir = os.path.join(self.options.arcadia_build_root, peer_dir)
            peer_dir = os.path.normpath(peer_dir)
            marker = os.path.join(peer_dir, pm_constants.OUTPUT_TAR_UUID_FILENAME)
            if os.path.isfile(marker):
                peer_dirs.append(peer_dir)
        return sorted(set(peer_dirs))

    def _get_peer_package_dependencies(self, peer_dirs: list[str]) -> dict[str, str]:
        dependencies = {}
        for peer_dir in peer_dirs:
            peer_package = PackageJson.load(pm_utils.build_pj_path(peer_dir))
            dependencies[peer_package.data["name"]] = PackageJson.WORKSPACE_SCHEMA + os.path.relpath(
                peer_dir, self.options.bindir
            )
        return dependencies

    def refresh_generated_peer_lockfile(self) -> None:
        """Refresh dynamic proto snapshots in an ordinary consumer's frozen lockfile."""
        workspace = PnpmWorkspace.load(pm_utils.build_ws_config_path(self.options.bindir))
        peer_dirs = []
        for peer_dir in sorted(workspace.get_paths(ignore_self=True)):
            manifest_path = pm_utils.build_pj_path(peer_dir)
            if not os.path.isfile(manifest_path):
                continue
            metadata = PackageJson.load(manifest_path).data.get("nots")
            if not isinstance(metadata, dict) or metadata.get("tsProtoAuto") is not True:
                continue
            extract_all_output_tars(peer_dir, build_root=self.options.arcadia_build_root)
            peer_dirs.append(peer_dir)
        if peer_dirs:
            self._update_peer_lockfile(peer_dirs, {})

    def _update_peer_lockfile(self, peer_dirs: list[str], peer_dependencies: dict[str, str]) -> None:
        lockfile_path = pm_utils.build_lockfile_path(self.options.bindir)
        lockfile = Lockfile.load(lockfile_path)

        for peer_dir in peer_dirs:
            peer_lockfile_path = pm_utils.build_lockfile_path(peer_dir)
            if os.path.isfile(peer_lockfile_path):
                peer_lockfile = Lockfile.load(peer_lockfile_path)
                PackageManager._rebase_file_tarball_resolutions(peer_lockfile, self.options.bindir)
                lockfile.merge(peer_lockfile)

        importer = lockfile.get_importers().setdefault(".", {})
        dependencies = importer.setdefault(PackageJson.DEP_KEY, {})
        packages = lockfile.data.setdefault("packages", {})

        for package_name, specifier in peer_dependencies.items():
            relative_path = specifier[len(PackageJson.WORKSPACE_SCHEMA) :]
            version = "file:" + relative_path
            package_key = f"{package_name}@{version}"
            dependencies[package_name] = {"specifier": specifier, "version": version}
            packages.setdefault(
                package_key,
                {"resolution": {"directory": relative_path, "type": "directory"}},
            )

        for peer_dir in peer_dirs:
            self._add_peer_snapshot(lockfile, peer_dir, set())

        lockfile.write()

    def _add_peer_snapshot(self, lockfile: Lockfile, peer_dir: str, visited: set[str]) -> None:
        if peer_dir in visited:
            return
        visited.add(peer_dir)
        package = PackageJson.load(pm_utils.build_pj_path(peer_dir))
        relative_path = os.path.relpath(peer_dir, self.options.bindir)
        package_key = f"{package.data['name']}@file:{relative_path}"
        peer_lockfile_path = pm_utils.build_lockfile_path(peer_dir)
        peer_importer = (
            Lockfile.load(peer_lockfile_path).get_importers().get(".", {}) if os.path.isfile(peer_lockfile_path) else {}
        )
        snapshot = {}
        for dependency_key in (PackageJson.DEP_KEY, PackageJson.OPT_DEP_KEY):
            resolved = {}
            for name, specifier in package.data.get(dependency_key, {}).items():
                if specifier.startswith(PackageJson.WORKSPACE_SCHEMA):
                    dependency_dir = os.path.normpath(
                        os.path.join(peer_dir, specifier[len(PackageJson.WORKSPACE_SCHEMA) :])
                    )
                    self._add_peer_snapshot(lockfile, dependency_dir, visited)
                    resolved[name] = "file:" + os.path.relpath(dependency_dir, self.options.bindir)
                else:
                    metadata = peer_importer.get(dependency_key, {}).get(name, {})
                    if "version" not in metadata:
                        raise ValueError(f"Cannot resolve {name} of TS_PROTO peer {peer_dir}: missing lockfile entry")
                    resolved[name] = metadata["version"]
            if resolved:
                snapshot[dependency_key] = resolved
        lockfile.data.setdefault("packages", {})[package_key] = {
            "resolution": {"directory": relative_path, "type": "directory"}
        }
        lockfile.data.setdefault("snapshots", {})[package_key] = snapshot

    def get_import_mappings(self) -> dict[str, str]:
        for peer_dir in self._get_ts_proto_peer_dirs():
            extract_all_output_tars(peer_dir, build_root=self.options.arcadia_build_root)
        return self._get_import_mappings()

    def _get_import_mappings(self) -> dict[str, str]:
        mappings = {}
        for proto_path, package_import, peer_moddir in self._get_peer_proto_exports():
            if self._is_peer_proto_source(proto_path, peer_moddir):
                mappings.setdefault(proto_path, package_import)
        return mappings

    def write_peer_reexports(self) -> str:
        # Keep existing generated/* entry points without duplicating peer code.
        # Include transitive exports too: an intermediate proto package may have
        # exposed these paths before it switched to non-recursive generation.
        mappings = {}
        for proto_path, package_import, _ in self._get_peer_proto_exports():
            mappings.setdefault(proto_path, package_import)
        filename = "ts-proto-reexports.json"
        with open(os.path.join(self.options.bindir, filename), "w") as output:
            json.dump(mappings, output)
        return filename

    def _get_peer_proto_exports(self):
        package = PackageJson.load(pm_utils.build_pj_path(self.options.bindir))
        for peer_dir in self._get_ts_proto_peer_dirs():
            peer_package = PackageJson.load(pm_utils.build_pj_path(peer_dir))
            package_name = peer_package.data["name"]
            # Custom manifests may leave proto peers as recursively generated sources.
            # Only import a built package when it is also installed for this consumer.
            if package.get_dep_specifier(package_name) is None:
                continue
            peer_moddir = os.path.relpath(peer_dir, self.options.arcadia_build_root)
            generated_marker = "/generated/"
            exports = peer_package.data.get("exports", {})
            if not isinstance(exports, dict):
                continue
            for subpath, target in exports.items():
                if not subpath.startswith("./"):
                    continue
                while isinstance(target, dict):
                    target = target.get("import") or target.get("require") or target.get("default")
                if not isinstance(target, str) or not target.endswith(".js"):
                    continue
                pattern = target.removeprefix("./")
                for generated_file in sorted(Path(peer_dir).glob(pattern.replace("*", "**/*"))):
                    generated_path = generated_file.relative_to(peer_dir).as_posix()
                    if generated_marker not in "/" + generated_path:
                        continue
                    proto_path = ("/" + generated_path).split(generated_marker, 1)[1].removesuffix(".js") + ".proto"
                    if "*" in pattern:
                        prefix, suffix = pattern.split("*", 1)
                        matched = generated_path[len(prefix) : -len(suffix) if suffix else None]
                        package_subpath = subpath[2:].replace("*", matched)
                    else:
                        package_subpath = subpath[2:]
                    yield proto_path, f"{package_name}/{package_subpath}", peer_moddir

    def _is_peer_proto_source(self, proto_path: str, peer_moddir: str) -> bool:
        # Generated paths use protoc's canonical import names, which omit
        # PROTO_NAMESPACE. Resolve them with the same include roots and keep
        # only the peer's own sources, not recursively generated dependencies.
        peer_roots = [Path(root) / peer_moddir for root in (self.options.arcadia_root, self.options.arcadia_build_root)]
        for include_root in self.options.proto_paths:
            source = Path(os.path.normpath(os.path.join(self.options.bindir, include_root, proto_path)))
            if source.is_file():
                return any(source.is_relative_to(peer_root) for peer_root in peer_roots)
        return False
