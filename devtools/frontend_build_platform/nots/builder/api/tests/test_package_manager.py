import importlib
import json
import os
from pathlib import Path
from unittest.mock import Mock

import yaml
import fcntl
import pytest

from build.plugins.lib.nots.package_manager import PackageJson

package_manager_module = importlib.import_module("devtools.frontend_build_platform.nots.builder.api.package_manager")


@pytest.mark.parametrize("override_registry", [False, True])
def test_exec_command_passes_platform_defaults_to_pnpm(monkeypatch, tmp_path, override_registry):
    defaults = {
        "managePackageManagerVersions": False,
        "pmOnFail": "warn",
        "registry": "https://npm.example.test/CaseSensitive",
        "trustLockfile": True,
        "verifyDepsBeforeRun": False,
    }
    monkeypatch.setattr(
        "devtools.frontend_build_platform.nots.builder.api.pnpm_workspace.pnpm_defaults",
        Mock(return_value=defaults),
    )
    process = Mock(returncode=0)
    process.communicate.return_value = ("", "")
    popen = Mock(return_value=process)
    monkeypatch.setattr(package_manager_module.subprocess, "Popen", popen)
    package_manager = package_manager_module.PackageManager(
        build_root=str(tmp_path),
        build_path=str(tmp_path / "build"),
        sources_path=str(tmp_path / "source"),
        nodejs_bin_path="/node",
        script_path="/pnpm.cjs",
        module_path="module",
    )
    env = {"CUSTOM_SETTING": "preserved"}
    if override_registry:
        env.update(npm_config_registry="https://registry.npmjs.org", pnpm_config_registry="https://registry.npmjs.org")
    original_env = env.copy()

    package_manager._exec_command(["install"], cwd=str(tmp_path), env=env)

    command_env = popen.call_args.kwargs["env"]
    for prefix in ("npm_config_", "pnpm_config_"):
        assert command_env[prefix + "manage_package_manager_versions"] == "false"
        assert command_env[prefix + "pm_on_fail"] == "warn"
        assert command_env[prefix + "registry"] == "https://npm.example.test/CaseSensitive"
        assert command_env[prefix + "trust_lockfile"] == "true"
        assert command_env[prefix + "verify_deps_before_run"] == "false"
    assert command_env["CUSTOM_SETTING"] == "preserved"
    assert env == original_env


def test_sync_mutex_file_uses_four_slots_by_default(monkeypatch, tmp_path):
    opened_paths = []
    locked_slots = []

    class Mutex:
        def __init__(self, path):
            self.path = path

        def close(self):
            pass

    def open_mutex(path, mode):
        assert mode == "w+"
        opened_paths.append(path)
        return Mutex(path)

    def lock_mutex(mutex, operation):
        if operation & fcntl.LOCK_UN:
            return
        locked_slots.append(mutex.path)
        if len(locked_slots) < package_manager_module.LOCAL_PNPM_INSTALL_CONCURRENCY:
            raise BlockingIOError

    monkeypatch.setattr(package_manager_module, "open", open_mutex, raising=False)
    monkeypatch.setattr(fcntl, "lockf", lock_mutex)
    mutex_path = str(tmp_path / "install_mutex")

    result = package_manager_module.sync_mutex_file(mutex_path)(lambda: "installed")()

    assert result == "installed"
    assert opened_paths == [
        "{}.{}".format(mutex_path, slot) for slot in range(package_manager_module.LOCAL_PNPM_INSTALL_CONCURRENCY)
    ]
    assert locked_slots == opened_paths


@pytest.mark.parametrize("busy", [False, True])
@pytest.mark.parametrize("install_fails", [False, True])
def test_pnpm_lock_wait_trace_excludes_install_and_release(monkeypatch, tmp_path, busy, install_fails):
    tracing = importlib.import_module("devtools.frontend_build_platform.libraries.logging.timeit")
    options = tracing.TimeitOptions()
    options.current_record = tracing.TimeitRecord("root", 0)
    options.trace_events = []
    options.enable(silent=True, use_dumper=True)
    monkeypatch.setattr(tracing, "options", options)
    operations = []
    attempts = 0

    class Mutex:
        def close(self):
            operations.append("close")

    monkeypatch.setattr(package_manager_module, "open", lambda path, mode: Mutex(), raising=False)

    def lock_mutex(mutex, operation):
        nonlocal attempts
        if operation & fcntl.LOCK_UN:
            assert [event.ph for event in options.trace_events] == ["B", "E"]
            operations.append("release")
            return
        attempts += 1
        if busy and attempts == 1:
            raise BlockingIOError
        operations.append("acquire")

    def wait(interval):
        assert interval == 0.1
        assert [event.ph for event in options.trace_events] == ["B"]
        operations.append("wait")

    def install():
        assert [event.ph for event in options.trace_events] == ["B", "E"]
        operations.append("install")
        if install_fails:
            raise RuntimeError("install failed")
        return "installed"

    monkeypatch.setattr(fcntl, "lockf", lock_mutex)
    monkeypatch.setattr(package_manager_module.time, "sleep", wait)
    run = package_manager_module.sync_mutex_file(str(tmp_path / "install_mutex"), concurrency=1)(install)

    if install_fails:
        with pytest.raises(RuntimeError, match="install failed"):
            run()
    else:
        assert run() == "installed"

    assert operations == (["wait"] if busy else []) + ["acquire", "install", "release", "close"]
    assert [event.name for event in options.trace_events] == ["_wait_for_pnpm_lock"] * 2
    assert options.trace_events[0].ts <= options.trace_events[1].ts
    assert options.current_record.name == "root"


def _package_manager(monkeypatch):
    package_manager = object.__new__(package_manager_module.PackageManager)
    commands = []
    package_manager._exec_command = lambda args, cwd: commands.append((args, cwd))
    monkeypatch.setattr(package_manager_module, "sync_mutex_file", lambda path: lambda function: function)
    return package_manager, commands


def test_pnpm_install_copies_external_node_modules_across_filesystems(monkeypatch, tmp_path):
    package_manager, commands = _package_manager(monkeypatch)
    cwd = str(tmp_path / "build" / "project" / "module")
    node_modules_path = str(tmp_path / "ram" / "project" / "module" / "node_modules")
    os.makedirs(tmp_path / "build" / "store")
    os.makedirs(os.path.dirname(node_modules_path))
    monkeypatch.setattr(package_manager_module, "_same_filesystem", lambda source, destination: False)

    os.makedirs(cwd, exist_ok=True)
    package_manager_module.PnpmWorkspace(os.path.join(cwd, "pnpm-workspace.yaml")).write()
    package_manager._run_pnpm_install(
        str(tmp_path / "build" / "store"),
        cwd,
        False,
        node_modules_path,
    )

    command, actual_cwd = commands[0]
    assert actual_cwd == cwd
    settings = yaml.safe_load(Path(cwd, "pnpm-workspace.yaml").read_text())
    assert settings["modulesDir"] == os.path.relpath(node_modules_path, cwd)
    assert "virtualStoreDir" not in settings
    assert settings["packageImportMethod"] == "copy"
    assert settings["registry"] == "https://npm.yandex-team.ru"
    assert settings["dedupePeerDependents"] is True
    assert settings["allowUnusedPatches"] is True
    assert "dedupePeers" not in settings


def test_pnpm_install_hardlinks_external_node_modules_on_same_filesystem(monkeypatch, tmp_path):
    package_manager, commands = _package_manager(monkeypatch)
    cwd = str(tmp_path / "build" / "project" / "module")
    store_dir = str(tmp_path / "ram" / "pnpm-ca-store")
    node_modules_path = str(tmp_path / "ram" / "project" / "module" / "node_modules")
    os.makedirs(store_dir)
    os.makedirs(os.path.dirname(node_modules_path))

    os.makedirs(cwd, exist_ok=True)
    package_manager_module.PnpmWorkspace(os.path.join(cwd, "pnpm-workspace.yaml")).write()
    package_manager._run_pnpm_install(
        store_dir,
        cwd,
        False,
        node_modules_path,
    )

    command, actual_cwd = commands[0]
    assert actual_cwd == cwd
    settings = yaml.safe_load(Path(cwd, "pnpm-workspace.yaml").read_text())
    assert settings["modulesDir"] == os.path.relpath(node_modules_path, cwd)
    assert settings["packageImportMethod"] == "hardlink"


@pytest.mark.parametrize("global_virtual_store", [False, True])
def test_pnpm_install_leaves_virtual_store_layout_to_pnpm(monkeypatch, tmp_path, global_virtual_store):
    package_manager, commands = _package_manager(monkeypatch)
    cwd = str(tmp_path / "build" / "project" / "module")
    node_modules_path = os.path.join(cwd, "node_modules")
    os.makedirs(tmp_path / "build" / "store")
    os.makedirs(cwd)

    os.makedirs(cwd, exist_ok=True)
    workspace = package_manager_module.PnpmWorkspace(os.path.join(cwd, "pnpm-workspace.yaml"))
    workspace.settings = {"enableGlobalVirtualStore": global_virtual_store}
    workspace.write()
    package_manager._run_pnpm_install(
        str(tmp_path / "build" / "store"),
        cwd,
        False,
        node_modules_path,
    )

    command, actual_cwd = commands[0]
    assert actual_cwd == cwd
    assert "--modules-dir" not in command
    settings = yaml.safe_load(Path(cwd, "pnpm-workspace.yaml").read_text())
    assert "--virtual-store-dir" not in command
    assert "virtualStoreDir" not in settings
    assert settings["enableGlobalVirtualStore"] is global_virtual_store
    assert settings["packageImportMethod"] == "hardlink"


def test_prepare_deps_publishes_package_json(tmp_path):
    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.module_path = "project/module"
    package_manager.sources_path = str(tmp_path / "project" / "module")

    inputs, outputs, _ = package_manager.calc_prepare_deps_inouts_and_resources(
        store_path="__tarballs__",
        has_deps=False,
        local_cli=True,
    )

    assert "$S/project/module/pnpm-lock.yaml" not in inputs
    assert "$B/project/module/package.json" in outputs


def test_build_package_json_preserves_source_bytes(tmp_path):
    source_path = tmp_path / "source"
    build_path = tmp_path / "build"
    source_path.mkdir()
    package_json_content = '{\n\t"name": "keep-original-formatting",\n\t"version": "1.0.0",\n\t"private": true\n}\n'
    (source_path / "package.json").write_text(package_json_content)

    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.sources_path = str(source_path)
    package_manager.build_path = str(build_path)
    package_manager.module_path = "project/module"
    package_manager.load_package_json = lambda path: _load_package_json(path)

    package_json = package_manager._build_package_json()

    assert (build_path / "package.json").read_text() == package_json_content
    assert package_json.path == str(build_path / "package.json")
    assert package_json.data["name"] == "keep-original-formatting"


def test_build_package_json_adds_missing_pack_metadata(tmp_path):
    source_path = tmp_path / "source"
    build_path = tmp_path / "build"
    source_path.mkdir()
    (source_path / "package.json").write_text('{"private": true, "files": ["build/"]}\n')

    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.sources_path = str(source_path)
    package_manager.build_path = str(build_path)
    package_manager.module_path = "project/module"
    package_manager.load_package_json = lambda path: _load_package_json(path)

    package_json = package_manager._build_package_json()

    assert package_json.path == str(build_path / "package.json")
    assert package_json.data == {
        "private": True,
        "files": ["build/"],
        "name": "@project/module",
        "version": "0.0.0",
    }


@pytest.mark.parametrize("indent", ["  ", "    ", "\t"])
@pytest.mark.parametrize("pnpm_version", ["10.33.4", "11.27.0", "11.28.0"])
@pytest.mark.parametrize(
    "fields, expected_fields",
    [
        ({}, {}),
        ({"engines": {"node": ">=22.13"}}, {"engines": {"node": ">=22.13"}}),
        ({"packageManager": "pnpm@10.33.4"}, {"packageManager": "pnpm@{version}"}),
        ({"engines": {"pnpm": "10"}}, {"engines": {"pnpm": "{version}"}}),
        (
            {"packageManager": "pnpm@10.33.4+sha512.old", "engines": {"pnpm": "<11", "node": ">=22.13"}},
            {"packageManager": "pnpm@{version}", "engines": {"pnpm": "{version}", "node": ">=22.13"}},
        ),
    ],
)
def test_build_package_json_uses_resource_pnpm_version(tmp_path, pnpm_version, fields, expected_fields, indent):
    source_path = tmp_path / "source"
    build_path = tmp_path / "build"
    pnpm_path = tmp_path / "resource" / "node_modules" / "pnpm"
    source_path.mkdir()
    pnpm_path.mkdir(parents=True)
    (pnpm_path / "package.json").write_text(json.dumps({"version": pnpm_version}))
    source_data = {"name": "example", "version": "1.0.0", **fields}
    source_content = json.dumps(source_data, indent=indent) + "\n"
    (source_path / "package.json").write_text(source_content)

    package_manager = package_manager_module.PackageManager(
        build_root=str(tmp_path),
        build_path=str(build_path),
        sources_path=str(source_path),
        module_path="project/module",
        nodejs_bin_path="/node",
        script_path=str(pnpm_path / "dist" / "pnpm.mjs"),
    )
    package_manager._build_package_json()

    expected = json.loads(json.dumps(expected_fields).replace("{version}", pnpm_version))
    assert (build_path / "package.json").read_text() == json.dumps(
        {"name": "example", "version": "1.0.0", **expected}, indent=indent
    ) + "\n"
    assert (source_path / "package.json").read_text() == source_content


def test_build_workspace_without_lockfile(tmp_path):
    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.sources_path = str(tmp_path / "source")
    package_manager.sources_root = str(tmp_path)
    package_manager.build_path = str(tmp_path / "build")
    package_manager.module_path = "project/module"

    os.makedirs(package_manager.sources_path)
    os.makedirs(package_manager.build_path)
    with open(os.path.join(package_manager.sources_path, "package.json"), "w") as f:
        f.write("{}")

    package_manager.build_workspace(tarballs_store="__tarballs__", local_cli=True)

    lockfile = package_manager.load_lockfile(os.path.join(package_manager.build_path, "pnpm-lock.yaml"))
    assert lockfile.data["lockfileVersion"] == "9.0"


def test_build_workspace_writes_pnpm_settings_to_workspace_config(tmp_path):
    source_path = tmp_path / "source"
    build_path = tmp_path / "build"
    source_path.mkdir()
    (source_path / "package.json").write_text(
        '{"pnpm":{'
        '"overrides":{"foo":"1.0.0"},'
        '"packageExtensions":{"bar":{"peerDependencies":{"baz":"2.0.0"}}},'
        '"neverBuiltDependencies":["esbuild"],'
        '"allowNonAppliedPatches":true'
        '}}\n'
    )
    (source_path / "pnpm-lock.yaml").write_text("lockfileVersion: '9.0'\n")

    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.sources_path = str(source_path)
    package_manager.sources_root = str(tmp_path)
    package_manager.build_path = str(build_path)
    package_manager.module_path = "project/module"
    package_manager.load_package_json = lambda path: _load_package_json(path)

    package_manager.build_workspace(tarballs_store="__tarballs__", local_cli=True)

    workspace = package_manager_module.PnpmWorkspace.load(str(build_path / "pnpm-workspace.yaml"))
    assert workspace.packages == set()
    assert workspace.settings == {
        "overrides": {"foo": "1.0.0"},
        "packageExtensions": {"bar": {"peerDependencies": {"baz": "2.0.0"}}},
    }
    assert _load_package_json(str(build_path / "package.json")).data["pnpm"] == {
        "overrides": {"foo": "1.0.0"},
        "packageExtensions": {"bar": {"peerDependencies": {"baz": "2.0.0"}}},
    }
    assert _load_package_json(str(source_path / "package.json")).data["pnpm"]["neverBuiltDependencies"] == ["esbuild"]


def test_build_workspace_does_not_merge_peer_lockfiles(tmp_path):
    source_path = tmp_path / "source" / "consumer"
    build_path = tmp_path / "build" / "consumer"
    reporter_path = tmp_path / "build" / "reporter"
    ci_reporter_path = tmp_path / "build" / "ci-reporter"
    source_path.mkdir(parents=True)
    reporter_path.mkdir(parents=True)
    ci_reporter_path.mkdir(parents=True)

    (source_path / "package.json").write_text('{"dependencies":{"reporter":"workspace:../reporter"}}\n')
    source_lockfile = package_manager_module.Lockfile(str(source_path / "pnpm-lock.yaml"))
    source_lockfile.data = {
        "lockfileVersion": "9.0",
        "importers": {
            ".": {"dependencies": {"reporter": {"specifier": "workspace:../reporter", "version": "link:../reporter"}}}
        },
    }
    source_lockfile.write()

    reporter_workspace = package_manager_module.PnpmWorkspace(str(reporter_path / "pnpm-workspace.yaml"))
    reporter_workspace.packages = {".", "../ci-reporter"}
    reporter_workspace.write()

    reporter_lockfile = package_manager_module.Lockfile(str(reporter_path / "pnpm-lock.yaml"))
    reporter_lockfile.data = {
        "lockfileVersion": "9.0",
        "importers": {".": {"dependencies": {"ci-reporter": {"specifier": "workspace:../ci-reporter"}}}},
    }
    reporter_lockfile.write()

    ci_reporter_lockfile = package_manager_module.Lockfile(str(ci_reporter_path / "pnpm-lock.yaml"))
    ci_reporter_lockfile.data = {
        "lockfileVersion": "9.0",
        "importers": {".": {"devDependencies": {"typescript": {"specifier": "5.9.3", "version": "5.9.3"}}}},
    }
    ci_reporter_lockfile.write()

    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.sources_path = str(source_path)
    package_manager.build_path = str(build_path)
    package_manager.build_root = str(tmp_path / "build")
    package_manager.module_path = "consumer"
    package_manager.sources_root = str(tmp_path / "source")

    package_manager.build_workspace(tarballs_store="__tarballs__", local_cli=True)

    lockfile = package_manager.load_lockfile(str(build_path / "pnpm-lock.yaml"))
    assert set(lockfile.get_importers()) == {"."}


def test_rebase_file_tarball_resolutions(tmp_path):
    source_dir = tmp_path / "source" / "module"
    target_dir = tmp_path / "target" / "module"
    lockfile = package_manager_module.Lockfile(str(source_dir / "pnpm-lock.yaml"))
    lockfile.data = {
        "lockfileVersion": "9.0",
        "packages": {
            "pkg@1.0.0": {
                "resolution": {
                    "integrity": "sha512-YQ==",
                    "tarball": "file:__tarballs__/pkg/-/pkg-1.0.0.tgz",
                }
            }
        },
    }

    package_manager_module.PackageManager._rebase_file_tarball_resolutions(lockfile, str(target_dir))

    tarball = lockfile.data["packages"]["pkg@1.0.0"]["resolution"]["tarball"]
    assert tarball == "file:../../source/module/__tarballs__/pkg/-/pkg-1.0.0.tgz"


def _load_package_json(path):
    package_json = PackageJson(path)
    package_json.read()
    return package_json


def test_production_install_reuses_install_flags(monkeypatch, tmp_path):
    package_manager, commands = _package_manager(monkeypatch)
    store = str(tmp_path / "store")
    os.makedirs(store)
    cwd = str(tmp_path)
    node_modules = str(tmp_path / "node_modules")
    package_manager_module.PnpmWorkspace(os.path.join(cwd, "pnpm-workspace.yaml")).write()
    for prod in [False, True]:
        ws = package_manager_module.PnpmWorkspace(os.path.join(cwd, "pnpm-workspace.yaml"))
        ws.settings = {"strictPeerDependencies": prod}
        ws.write()
        package_manager._run_pnpm_install(store, cwd, False, node_modules, prod=prod)
        settings = yaml.safe_load(Path(cwd, "pnpm-workspace.yaml").read_text())
        assert settings["strictPeerDependencies"] is prod
    ordinary, production = [cmd for cmd, cwd in commands]
    assert "--prod" in production
    assert [arg for arg in production if arg != "--prod"] == ordinary


def test_production_reinstall_removes_restored_node_modules(monkeypatch, tmp_path):
    package_manager = object.__new__(package_manager_module.PackageManager)
    package_manager.build_path = str(tmp_path / "build")
    os.makedirs(package_manager.build_path)
    restored = tmp_path / "ram" / "node_modules"
    restored.mkdir(parents=True)
    (restored / ".pnpm").mkdir()
    (restored / ".pnpm" / "dev-package").write_text("stale")
    node_modules = tmp_path / "build" / "node_modules"
    node_modules.symlink_to(restored)
    calls = []

    def create_node_modules(**kwargs):
        assert not restored.exists()
        assert not os.path.lexists(node_modules)
        calls.append(kwargs)

    package_manager.create_node_modules = create_node_modules
    package_manager.prune_node_modules(yatool_prebuilder_path="/prebuilder", local_cli=False)
    assert calls == [{"yatool_prebuilder_path": "/prebuilder", "local_cli": False, "prod": True}]


def test_pnpm_install_detaches_readonly_workspace_input(monkeypatch, tmp_path):
    package_manager, commands = _package_manager(monkeypatch)
    cwd = tmp_path / "module"
    cwd.mkdir()
    source = tmp_path / "workspace-input.yaml"
    source.write_text("overrides: {foo: '1.0.0'}\n")
    source.chmod(0o444)
    workspace = cwd / "pnpm-workspace.yaml"
    os.link(source, workspace)
    original = source.read_bytes()
    store = tmp_path / "store"
    store.mkdir()
    package_manager._run_pnpm_install(
        str(store), str(cwd), False, str(cwd / "node_modules")
    )
    assert source.read_bytes() == original
    assert source.stat().st_mode & 0o222 == 0
    assert source.stat().st_ino != workspace.stat().st_ino
    settings = yaml.safe_load(workspace.read_text())
    assert settings["overrides"] == {"foo": "1.0.0"}
    assert "storeDir" not in settings
    args, _ = commands[0]
    assert args[args.index("--store-dir") + 1] == str(store)
    assert len(commands) == 1


def test_build_workspace_preserves_manifest_format_without_pnpm_changes(tmp_path):
    source = tmp_path / "source"
    source.mkdir()
    manifest = '{\n  "name": "pkg",\n  "version": "1.0.0",\n  "files": ["build"]\n}\n'
    (source / "package.json").write_text(manifest)
    pm = package_manager_module.PackageManager(
        str(tmp_path / "build"),
        str(tmp_path / "build/module"),
        str(source),
        None,
        None,
        module_path="module",
        sources_root=str(tmp_path),
    )
    pm.build_workspace("__tarballs__", True)
    assert Path(pm.build_path, "package.json").read_text() == manifest


def test_build_workspace_rebases_common_file_override(tmp_path):
    source_root = tmp_path / "source"
    source_path = source_root / "project/pkg"
    build_path = tmp_path / "build/project/pkg"
    source_path.mkdir(parents=True)
    (source_path.parent / "common.yaml").write_text("overrides: {toolkit: 'file:./toolkit'}\n")
    (source_path / "package.json").write_text('{"nots":{"commonConfigPath":"../common.yaml"}}')
    (source_path / "pnpm-lock.yaml").write_text("lockfileVersion: '9.0'\n")
    pm = object.__new__(package_manager_module.PackageManager)
    pm.sources_path = str(source_path)
    pm.sources_root = str(source_root)
    pm.build_path = str(build_path)
    pm.module_path = "project/pkg"
    pm.build_root = str(tmp_path / "build")
    peer_path = build_path.parent / "toolkit"
    peer_path.mkdir(parents=True)
    (peer_path / "pnpm-workspace.yaml").write_text("packages: []\ncatalogs: {peer/common: {foo: '1.0.0'}}")
    pm.build_workspace("__tarballs__", True)
    workspace = yaml.safe_load((build_path / "pnpm-workspace.yaml").read_text())
    assert workspace["overrides"] == {"toolkit": "file:../toolkit"}
    assert workspace["catalogs"] == {"peer/common": {"foo": "1.0.0"}}
    assert workspace["packages"] == []
    assert _load_package_json(str(build_path / "package.json")).data["pnpm"]["overrides"] == workspace["overrides"]


@pytest.mark.parametrize("lockfile_version", ["6.0", "9.0"])
@pytest.mark.parametrize(
    "name, version, package_path",
    [
        ("@emnapi/runtime", "1.11.2", "@emnapi%2fruntime/-/runtime-1.11.2.tgz"),
        ("lodash", "4.17.21", "lodash/-/lodash-4.17.21.tgz"),
    ],
)
def test_distbuild_without_tarball_uses_same_resource_and_local_path(
    tmp_path, lockfile_version, name, version, package_path
):
    source = tmp_path / "source" / "module"
    build = tmp_path / "build" / "module"
    source.mkdir(parents=True)
    (source / "package.json").write_text('{"dependencies": {"' + name + '": "' + version + '"}}')
    key = ("/" if lockfile_version == "6.0" else "") + name + "@" + version
    data = {
        "lockfileVersion": lockfile_version,
        "packages": {key: {"resolution": {"integrity": "sha512-aGFzaA=="}}},
    }
    source_lockfile = yaml.safe_dump(data)
    (source / "pnpm-lock.yaml").write_text(source_lockfile)
    pm = package_manager_module.PackageManager(
        nodejs_bin_path="/unused/node",
        script_path="/unused/pnpm",
        build_root=str(tmp_path / "build"),
        build_path=str(build),
        sources_path=str(source),
        module_path="module",
        sources_root=str(tmp_path / "source"),
    )

    _, outputs, resources = pm.calc_prepare_deps_inouts_and_resources("__tarballs__", True, False)
    pm.build_workspace("__tarballs__", False)

    tarball_path = "__tarballs__/" + package_path.replace("%2f", "/")
    assert resources == ["https://npm.yandex-team.ru/" + package_path + "#integrity=sha512-aGFzaA=="]
    assert "$B/module/" + tarball_path in outputs
    generated = yaml.safe_load((build / "pnpm-lock.yaml").read_text())
    assert generated["packages"][name + "@" + version]["resolution"] == {
        "tarball": "file:" + tarball_path,
        "integrity": "sha512-aGFzaA==",
    }
    assert (source / "pnpm-lock.yaml").read_text() == source_lockfile


@pytest.mark.parametrize(
    "key",
    [
        "foo?bar@1.2.3",
        "@scope/foo#bar@1.2.3",
        "@scope%2fname/foo@1.2.3",
        "pkg@01.2.3",
        "pkg@1.2.3-..",
        "pkg@1.2.3+build..2",
    ],
)
def test_distbuild_rejects_invalid_keys_before_declaring_or_rewriting_tarballs(tmp_path, key):
    source = tmp_path / "source" / "module"
    source.mkdir(parents=True)
    (source / "pnpm-lock.yaml").write_text(
        yaml.safe_dump(
            {
                "lockfileVersion": "9.0",
                "packages": {key: {"resolution": {"integrity": "sha512-aGFzaA=="}}},
            }
        )
    )
    pm = package_manager_module.PackageManager(
        nodejs_bin_path="/unused/node",
        script_path="/unused/pnpm",
        build_root=str(tmp_path / "build"),
        build_path=str(tmp_path / "build" / "module"),
        sources_path=str(source),
        module_path="module",
        sources_root=str(tmp_path / "source"),
    )

    with pytest.raises(package_manager_module.PackageManagerError, match="cannot infer tarball URL"):
        pm.calc_prepare_deps_inouts_and_resources("__tarballs__", True, False)
    with pytest.raises(TypeError, match="cannot infer tarball URL"):
        pm._build_merged_lockfile("__tarballs__", [], False, True)
    assert not (tmp_path / "build" / "module" / "pnpm-lock.yaml").exists()


@pytest.mark.parametrize("common", [False, True])
@pytest.mark.parametrize("value", ["workspace:../replacement", "$colors"])
def test_build_workspace_rejects_workspace_overrides(tmp_path, common, value):
    source = tmp_path / "source"
    source.mkdir()
    manifest = {"dependencies": {"colors": "workspace:../replacement"}}
    if common:
        (source / "common.yaml").write_text("overrides: {colors: '" + value + "'}\n")
        manifest["nots"] = {"commonConfigPath": "./common.yaml"}
    else:
        manifest["pnpm"] = {"overrides": {"colors": value}}
    (source / "package.json").write_text(json.dumps(manifest))
    lockfile = "lockfileVersion: '9.0'\n"
    (source / "pnpm-lock.yaml").write_text(lockfile)
    pm = package_manager_module.PackageManager(
        str(tmp_path / "build"),
        str(tmp_path / "build/module"),
        str(source),
        None,
        None,
        module_path="module",
        sources_root=str(tmp_path),
    )
    with pytest.raises(ValueError, match="overrides colors: workspace: протокол не поддерживается"):
        pm.build_workspace("__tarballs__", True)
    assert (source / "pnpm-lock.yaml").read_text() == lockfile
