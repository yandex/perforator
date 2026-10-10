import copy
import hashlib
import os
import shutil
import subprocess
import sys
import time

from build.plugins.lib.nots.package_manager import PackageManager as BasePackageManager, PackageManagerError
from build.plugins.lib.nots.package_manager.constants import (
    VIRTUAL_STORE_DIRNAME,
)
from build.plugins.lib.nots.package_manager.lockfile import Lockfile
from build.plugins.lib.nots.package_manager.utils import (
    build_lockfile_path,
    build_ws_config_path,
    build_nm_path,
    build_nots_path,
    build_pj_path,
)
from .pnpm_workspace import PnpmWorkspace, pnpm_defaults
from .utils import copy_writable_file
from build.plugins.lib.nots.package_manager.common_config import (
    load_common_config,
    rebase_pnpm_settings,
    join_pnpm_settings,
    PNPM_SETTINGS,
)
from build.plugins.lib.nots.package_manager.timeit import timeit

LOCAL_PNPM_INSTALL_CONCURRENCY = 4
LOCAL_PNPM_INSTALL_MUTEX_FILENAME = ".__install_mutex__"


def _same_filesystem(source: str, destination: str) -> bool:
    return os.stat(source).st_dev == os.stat(os.path.dirname(destination)).st_dev


class PackageManagerCommandError(PackageManagerError):
    def __init__(self, cmd, code, stdout, stderr):
        self.cmd = cmd
        self.code = code
        self.stdout = stdout
        self.stderr = stderr

        msg = "package manager exited with code {} while running {}:\n{}\n{}".format(code, cmd, stdout, stderr)
        super(PackageManagerCommandError, self).__init__(msg)


"""
Creates a decorator that limits concurrent access to a function using mutex files.

The decorator uses non-blocking file locks (fcntl.LOCK_EX) as semaphore slots.
At most ``concurrency`` processes can execute the decorated function at a time.
The acquired lock is released (fcntl.LOCK_UN) when the function completes.

Args:
    mutex_filename (str): Base path for the files used as semaphore slots.
    concurrency (int): Maximum number of concurrent function executions.

Returns:
    function: A decorator function that applies the synchronization logic.
"""


@timeit
def _wait_for_pnpm_lock(mutexes):
    import fcntl

    while True:
        for mutex in mutexes:
            try:
                fcntl.lockf(mutex, fcntl.LOCK_EX | fcntl.LOCK_NB)
            except BlockingIOError:
                continue

            return mutex

        time.sleep(0.1)


def sync_mutex_file(mutex_filename, concurrency=LOCAL_PNPM_INSTALL_CONCURRENCY):
    if concurrency < 1:
        raise ValueError("concurrency must be at least 1")

    def decorator(function):
        def wrapper(*args, **kwargs):
            import fcntl

            mutexes = [open("{}.{}".format(mutex_filename, slot), "w+") for slot in range(concurrency)]
            try:
                mutex = _wait_for_pnpm_lock(mutexes)
                try:
                    return function(*args, **kwargs)
                finally:
                    fcntl.lockf(mutex, fcntl.LOCK_UN)
            finally:
                for mutex in mutexes:
                    mutex.close()

        return wrapper

    return decorator


"""
Calculates the MD5 hash of multiple files.

Reads files in chunks of 64KB and updates the MD5 hash incrementally. Files are processed in sorted order to ensure consistent results.

Args:
    files (list): List of file paths to be hashed.

Returns:
    str: Hexadecimal MD5 hash digest of the concatenated file contents.
"""


def hash_files(files):
    BUF_SIZE = 65536  # read in 64kb chunks
    md5 = hashlib.md5()
    for filename in sorted(files):
        with open(filename, 'rb') as f:
            while True:
                data = f.read(BUF_SIZE)
                if not data:
                    break
                md5.update(data)

    return md5.hexdigest()


"""
Creates a decorator that runs the decorated function only if specified files have changed.

The decorator checks the hash of provided files against a saved hash from previous runs.
If hashes differ (files changed) or no saved hash exists, runs the decorated function
and updates the saved hash. If hashes are the same, skips the function execution.

Args:
    files_to_hash: List of files to track for changes.
    hash_storage_filename: Path to file where hash state is stored.

Returns:
    A decorator function that implements the described behavior.
"""


def hashed_by_files(files_to_hash, paths_to_exist, hash_storage_filename):
    def decorator(function):
        def wrapper(*args, **kwargs):
            all_paths_exist = True
            for p in paths_to_exist:
                if not os.path.exists(p):
                    all_paths_exist = False
                    break

            current_state_hash = hash_files(files_to_hash)
            saved_hash = None
            if all_paths_exist and os.path.exists(hash_storage_filename):
                with open(hash_storage_filename, "r") as f:
                    saved_hash = f.read()

            if saved_hash == current_state_hash:
                return None
            else:
                result = function(*args, **kwargs)
                with open(hash_storage_filename, "w+") as f:
                    f.write(current_state_hash)

            return result

        return wrapper

    return decorator


class PackageManager(BasePackageManager):
    def __init__(
        self,
        build_root,
        build_path,
        sources_path,
        nodejs_bin_path,
        script_path,
        module_path=None,
        sources_root=None,
        verbose=False,
        ld_library_path=None,
    ):
        super(PackageManager, self).__init__(
            build_root=build_root,
            build_path=build_path,
            sources_path=sources_path,
            module_path=module_path,
            sources_root=sources_root,
        )
        self.nodejs_bin_path = nodejs_bin_path
        self.script_path = script_path
        self.verbose = verbose
        self.ld_library_path = ld_library_path

    def _build_package_json(self):
        """
        :rtype: PackageJson
        """
        source_path = build_pj_path(self.sources_path)
        build_path = build_pj_path(self.build_path)
        os.makedirs(self.build_path, exist_ok=True)

        package_json = self.load_package_json(source_path)
        should_write = False

        if "name" not in package_json.data:
            parts = self.module_path.split("/")
            package_json.data["name"] = "@{}/{}".format(parts[0], parts[-1])
            should_write = True
        if "version" not in package_json.data:
            package_json.data["version"] = "0.0.0"
            should_write = True

        if should_write:
            package_json.path = build_path
            package_json.write()
        else:
            shutil.copyfile(source_path, build_path)
            package_json.path = build_path

        return package_json

    @timeit
    def _exec_command(self, args, cwd: str, include_defaults=True, script_path=None, env={}):
        if not self.nodejs_bin_path:
            raise PackageManagerError("Unable to execute command: nodejs_bin_path is not configured")

        cmd_env = env.copy()
        cmd_env["PNPM_MAX_WORKERS"] = os.environ.get("PNPM_MAX_WORKERS", "4")

        if self.ld_library_path:
            cmd_env["LD_LIBRARY_PATH"] = self.ld_library_path

        cmd = (
            [self.nodejs_bin_path, script_path or self.script_path]
            + args
            + (self._get_default_options() if include_defaults else [])
        )
        p = subprocess.Popen(
            cmd,
            cwd=cwd,
            stdin=None,
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            env=cmd_env,
            text=True,
            encoding="utf-8",
        )
        stdout, stderr = p.communicate()

        if self.verbose:
            for key, value in cmd_env.items():
                escaped_value = value.replace('"', '\\"').replace("$", "\\$")
                print(f'export {key}="{escaped_value}', file=sys.stderr)
            print(f'cd {cwd} && {" ".join(cmd)}', file=sys.stderr)
            print(f'stdout: {stdout}', file=sys.stderr) if stdout else None
            print(f'stderr: {stderr}', file=sys.stderr) if stderr else None

        if p.returncode != 0:
            self._dump_debug_log()

            raise PackageManagerCommandError(cmd, p.returncode, stdout, stderr)

    def _nm_path(self, *parts):
        return os.path.join(build_nm_path(self.build_path), *parts)

    def _get_default_options(self):
        return ["--stream", "--reporter", "append-only", "--no-color"]

    def _get_debug_log_path(self):
        return self._nm_path(".pnpm-debug.log")

    def _dump_debug_log(self):
        log_path = self._get_debug_log_path()

        if not log_path:
            return

        try:
            with open(log_path) as f:
                sys.stderr.write("Package manager log {}:\n{}\n".format(log_path, f.read()))
        except Exception:
            sys.stderr.write("Failed to dump package manager log {}.\n".format(log_path))

    @timeit
    def _get_pnpm_store(self):
        return os.path.join(build_nots_path(self.build_root), "pnpm_store")

    @timeit
    def _get_file_hash(self, path: str):
        sha256 = hashlib.sha256()

        with open(path, "rb") as f:
            # Read the file in chunks
            for chunk in iter(lambda: f.read(4096), b""):
                sha256.update(chunk)

        return sha256.hexdigest()

    @timeit
    def create_node_modules(
        self,
        yatool_prebuilder_path=None,
        local_cli=False,
        node_modules_path=None,
        store_dir=None,
        prod=False,
    ):
        """
        Creates node_modules directory according to the lockfile.
        """
        ws = self._prepare_workspace()

        self._copy_pnpm_patches()

        # Pure `tier 0` logic - isolated stores in the `build_root` (works in `distbuild` and `CI autocheck`)
        store_dir = store_dir or self._get_pnpm_store()
        default_node_modules_path = build_nm_path(self.build_path)
        node_modules_path = node_modules_path or default_node_modules_path
        os.makedirs(store_dir, exist_ok=True)
        os.makedirs(os.path.dirname(node_modules_path), exist_ok=True)
        virtual_store_dir = os.path.join(node_modules_path, VIRTUAL_STORE_DIRNAME)

        self._run_pnpm_install(
            store_dir,
            self.build_path,
            local_cli,
            virtual_store_dir,
            node_modules_path,
            prod=prod,
        )

        self._run_apply_addons_if_need(yatool_prebuilder_path, virtual_store_dir)

        return ws

    @timeit
    def prune_node_modules(self, yatool_prebuilder_path=None, local_cli=False):
        """Reinstall only production dependencies before bundling injected node_modules."""
        node_modules_path = build_nm_path(self.build_path)
        # A restored layer can live on a RAM disk behind this symlink.
        real_node_modules_path = os.path.realpath(node_modules_path)
        if os.path.isdir(real_node_modules_path):
            shutil.rmtree(real_node_modules_path)
        if os.path.islink(node_modules_path):
            os.unlink(node_modules_path)

        self.create_node_modules(
            yatool_prebuilder_path=yatool_prebuilder_path,
            local_cli=local_cli,
            prod=True,
        )

    """
    Runs pnpm install command with specified parameters in an exclusive and hashed manner.

    This method executes the pnpm install command with various flags and options, ensuring it's run exclusively
    using a mutex file and only if the specified files have changed (using a hash check). The command is executed
    in the given working directory (cwd) with the provided store and virtual store directories.

    Args:
        store_dir (str): Path to the store directory where packages will be stored.
        cwd (str): Working directory where the command will be executed.

    Note:
        Uses file locking via fcntl to ensure exclusive execution.
        The command execution is hashed based on the pnpm-lock.yaml file.
    """

    @timeit
    def _run_pnpm_install(
        self,
        store_dir: str,
        cwd: str,
        local_cli: bool,
        virtual_store_dir: str,
        node_modules_path: str,
        prod: bool = False,
    ):
        # Use fcntl to lock a temp file

        def execute_install_cmd():
            default_node_modules_path = build_nm_path(cwd)
            custom_node_modules_path = os.path.normpath(node_modules_path) != os.path.normpath(
                default_node_modules_path
            )
            package_import_method = "hardlink" if _same_filesystem(store_dir, node_modules_path) else "copy"
            ws = PnpmWorkspace.load(build_ws_config_path(cwd))
            defaults = pnpm_defaults()
            ws.runtime_settings = {
                **defaults,
                **{key: value for key, value in ws.settings.items() if key in defaults},
                "modulesDir": os.path.relpath(node_modules_path, cwd) if custom_node_modules_path else "node_modules",
                "packageImportMethod": package_import_method,
                "virtualStoreDir": virtual_store_dir,
            }
            copy_writable_file(ws.path, ws.path)
            ws.write()
            install_cmd = [
                "install",
                "--store-dir",
                store_dir,
                "--frozen-lockfile",
                "--prefer-offline" if local_cli else "--offline",
            ]
            if prod:
                install_cmd.append("--prod")

            self._exec_command(install_cmd, cwd=cwd)

        mutex_file = os.path.join(store_dir, LOCAL_PNPM_INSTALL_MUTEX_FILENAME)
        os.makedirs(os.path.dirname(mutex_file), exist_ok=True)
        execute_hashed_cmd_exclusively = sync_mutex_file(mutex_file)(execute_install_cmd)
        execute_hashed_cmd_exclusively()

    """
    Calculate inputs, outputs and resources for dependency preparation phase.

    Args:
        store_path: Path to the store where tarballs will be stored.
        has_deps: Boolean flag indicating whether the module has dependencies.

    Returns:
        tuple[list[str], list[str], list[str]]: A tuple containing three lists:
            - ins: List of input file paths
            - outs: List of output file paths
            - resources: List of package URIs (when has_deps is True)

    Note:
        Uses @timeit decorator to measure execution time of this method.
    """

    @timeit
    def _prepare_workspace(self):
        return PnpmWorkspace.load(build_ws_config_path(self.build_path))

    @timeit
    def build_workspace(self, tarballs_store: str, local_cli: bool):
        """
        :rtype: PnpmWorkspace
        """
        pj = self._build_package_json()

        ws = PnpmWorkspace(build_ws_config_path(self.build_path))
        ws.set_from_package_json(pj)
        source_pj = self.load_package_json_from_dir(self.sources_path)
        config_path, ws.catalogs, settings = load_common_config(source_pj, self.sources_root, include_settings=True)
        if config_path:
            common_settings = rebase_pnpm_settings(
                settings,
                os.path.join(
                    self.build_path,
                    os.path.relpath(os.path.join(self.sources_root, os.path.dirname(config_path)), self.sources_path),
                ),
                self.build_path,
            )
            ws.settings = join_pnpm_settings(common_settings, ws.settings)
        # Distbuild delivers prepared peer workspaces, not their source configs.
        # A peer workspace already contains its transitive catalog closure.
        for peer_path in (self.get_local_peers_from_package_json() if pj.has_dependencies() else []):
            peer = PnpmWorkspace.load(build_ws_config_path(os.path.join(self.build_root, peer_path)))
            for group, entries in peer.catalogs.items():
                if group in ws.catalogs and ws.catalogs[group] != entries:
                    raise ValueError("Conflicting catalog group {}".format(group))
                ws.catalogs[group] = copy.deepcopy(entries)
        ws.write()
        # pnpm 10 prefers package.json#pnpm over workspace YAML for these fields.
        # Only the generated manifest receives shared settings.
        shared_fields = {key: ws.settings[key] for key in PNPM_SETTINGS if key in ws.settings}
        if shared_fields != pj.data.get("pnpm", {}):
            if shared_fields:
                pj.data["pnpm"] = shared_fields
            else:
                pj.data.pop("pnpm", None)
            pj.write()
        dep_paths = ws.get_paths(ignore_self=True)
        self._build_merged_lockfile(tarballs_store, dep_paths, local_cli, pj.has_dependencies())

        return ws

    @timeit
    def build_ts_proto_auto_workspace(self, deps_mod: str):
        """
        :rtype: PnpmWorkspace
        """

        ws = PnpmWorkspace(build_ws_config_path(self.build_path))
        ws.write()

        deps_lockfile_path = build_lockfile_path(os.path.join(self.build_root, deps_mod))
        lockfile_path = build_lockfile_path(self.build_path)
        lf = self.load_lockfile(deps_lockfile_path)
        self._rebase_file_tarball_resolutions(lf, self.build_path)
        lf.write(lockfile_path)

        return ws

    @timeit
    def _build_merged_lockfile(self, tarballs_store, dep_paths, local_cli: bool, has_deps: bool):
        """
        :type dep_paths: list of str
        :rtype: PnpmLockfile
        """
        lockfile_path = build_lockfile_path(self.sources_path)
        if has_deps or os.path.exists(lockfile_path):
            lf = self.load_lockfile(lockfile_path)
        else:
            lf = Lockfile(lockfile_path)
            lf.data = {"lockfileVersion": "9.0"}
        # Change to the output path for correct path calcs on merging.
        lf.path = build_lockfile_path(self.build_path)
        if not local_cli:
            lf.update_tarball_resolutions(
                lambda p: "file:"
                + os.path.relpath(
                    os.path.join(self.build_root, self._tarballs_store_path(p, tarballs_store)),
                    self.build_path,
                )
            )

        lf.write()

    @staticmethod
    def _rebase_file_tarball_resolutions(lf, target_dir):
        source_dir = os.path.dirname(lf.path)

        def rebase(pkg):
            if not pkg.tarball_url.startswith("file:"):
                return pkg.tarball_url

            source_path = os.path.normpath(os.path.join(source_dir, pkg.tarball_url[len("file:") :]))
            return "file:" + os.path.relpath(source_path, target_dir)

        lf.update_tarball_resolutions(rebase)

    @timeit
    def _run_apply_addons_if_need(self, yatool_prebuilder_path, virtual_store_dir):
        if not yatool_prebuilder_path:
            return

        self._exec_command(
            [
                "apply-addons",
                "--virtual-store",
                virtual_store_dir,
            ],
            cwd=self.build_path,
            include_defaults=False,
            script_path=os.path.join(yatool_prebuilder_path, "build", "bin", "prebuilder.js"),
        )

    @timeit
    def _copy_pnpm_patches(self):
        patched_dependencies = self._prepare_workspace().settings.get("patchedDependencies", {})

        for p in patched_dependencies.values():
            patch_source_path = os.path.join(self.sources_path, p)
            patch_build_path = os.path.join(self.build_path, p)
            os.makedirs(os.path.dirname(patch_build_path), exist_ok=True)
            shutil.copyfile(patch_source_path, patch_build_path)
