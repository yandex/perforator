import importlib
from pathlib import Path
from types import SimpleNamespace

import pytest

prepare_deps_module = importlib.import_module("devtools.frontend_build_platform.nots.builder.api.prepare_deps")


def _prepare_deps_args(tmp_path):
    curdir = tmp_path / "project"
    curdir.mkdir()
    return SimpleNamespace(
        arcadia_build_root=str(tmp_path / "build"),
        bindir=str(tmp_path / "build" / "project"),
        curdir=str(curdir),
        nodejs_bin="node",
        pm_root="pnpm",
        ts_proto_auto_deps_path=None,
        tarballs_store="__tarballs__",
        tarballs_platform=None,
        local_cli=False,
    )


def test_prepare_deps_rejects_lockfile_dependencies_missing_from_manifest(monkeypatch, tmp_path):
    args = _prepare_deps_args(tmp_path)
    lockfile_path = tmp_path / "project" / "pnpm-lock.yaml"
    lockfile_path.write_text("""
lockfileVersion: '9.0'
importers:
  .:
    dependencies:
      stale:
        specifier: 1.0.0
        version: 1.0.0
""")

    class PackageManager:
        def __init__(self, **kwargs):
            pass

        def load_package_json_from_dir(self, path):
            return SimpleNamespace(has_dependencies=lambda: False)

        def load_lockfile(self, path):
            return prepare_deps_module.Lockfile.load(path)

    monkeypatch.setattr(
        prepare_deps_module,
        "TsProtoGenerator",
        lambda args: pytest.fail("ordinary consumers must use their source lockfile"),
    )
    monkeypatch.setattr(prepare_deps_module, "PackageManager", PackageManager)

    with pytest.raises(
        prepare_deps_module.PackageManagerError,
        match=r"pnpm-lock\.yaml is out of date: package\.json declares no dependencies.*stale",
    ):
        prepare_deps_module.prepare_deps(args)


@pytest.mark.parametrize(
    "content",
    [None, "", " \n\t\n", "# no dependencies\n", "{}\n", "lockfileVersion: '9.0'\nimporters:\n  .: {}\n"],
)
def test_prepare_deps_accepts_dependency_free_lockfile(tmp_path, content):
    args = _prepare_deps_args(tmp_path)
    project = tmp_path / "project"
    (project / "package.json").write_text('{"name": "project"}\n')
    source_lockfile = project / "pnpm-lock.yaml"
    if content is not None:
        source_lockfile.write_text(content)

    prepare_deps_module.prepare_deps(args)

    built_lockfile = prepare_deps_module.Lockfile.load(str(tmp_path / "build" / "project" / "pnpm-lock.yaml"))
    assert built_lockfile.data["lockfileVersion"] == "9.0"
    assert built_lockfile.get_importers().get(".", {}) == {}
    if content is None:
        assert not source_lockfile.exists()
    else:
        assert source_lockfile.read_text() == content


@pytest.mark.parametrize("supported_os, expected_packages", [(None, {"mac"}), (["linux"], {"mac", "linux"})])
def test_copy_tarballs_uses_exactly_graph_selected_packages(monkeypatch, tmp_path, supported_os, expected_packages):
    from build.plugins.lib.nots.package_manager.lockfile import Lockfile
    from build.plugins.lib.nots.package_manager.package_manager import PackageManager

    args = _prepare_deps_args(tmp_path)
    args.resource_root = str(tmp_path / "resources")
    args.tarballs_platform = ["darwin", "arm64", "unknown"]
    lock = Lockfile(str(tmp_path / "project/pnpm-lock.yaml"))
    lock.data = {
        "lockfileVersion": "9.0",
        "packages": {
            name + "@1.0.0": {"resolution": {"integrity": "sha512-AA=="}, "os": [os_name]}
            for name, os_name in (("mac", "darwin"), ("linux", "linux"), ("windows", "win32"))
        },
        "snapshots": {name + "@1.0.0": {"optional": True} for name in ("mac", "linux", "windows")},
    }
    lock.write()
    (tmp_path / "project/package.json").write_text('{"dependencies":{"mac":"1.0.0"}}')
    pm = PackageManager(args.arcadia_build_root, args.bindir, args.curdir)
    supported_architectures = None if supported_os is None else {"os": supported_os}
    _, outputs, resources = pm.calc_prepare_deps_inouts_and_resources(
        "__tarballs__", True, False, platform=args.tarballs_platform, supported_architectures=supported_architectures
    )
    copies = []
    monkeypatch.setattr(
        prepare_deps_module.library.python.fs,
        "hardlink_or_copy",
        lambda source, target: copies.append((source, target)),
    )
    monkeypatch.setattr(pm, "load_lockfile", lambda _: pytest.fail("already loaded lockfile must not be read again"))
    packages = pm.extract_packages_meta_from_lockfiles([lock], args.tarballs_platform, supported_architectures)
    prepare_deps_module._copy_tarballs(args, packages)
    assert {resource.split("/")[3] for resource in resources} == expected_packages
    expected_suffixes = {f"{name}/-/{name}-1.0.0.tgz" for name in expected_packages}
    assert {out.split("__tarballs__/", 1)[1] for out in outputs if "__tarballs__/" in out} == expected_suffixes
    assert {target.split("__tarballs__/", 1)[1] for _, target in copies} == expected_suffixes


def test_offline_install_skips_foreign_optional_tarball_without_resource(tmp_path):
    import base64
    import hashlib
    import io
    import json
    import os
    import platform
    import sys
    import tarfile

    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager

    args = _prepare_deps_args(tmp_path)
    args.resource_root = str(tmp_path / "resources")
    cpu = {"aarch64": "arm64", "arm64": "arm64", "x86_64": "x64", "AMD64": "x64"}.get(platform.machine(), "unknown")
    args.tarballs_platform = [sys.platform, cpu, "unknown"]
    args.nodejs_bin = os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node")
    args.pm_root = os.environ["NOTS_TEST_PNPM_ROOT"]
    manifest = {
        "name": "probe",
        "version": "1.0.0",
        "dependencies": {"portable": "1.0.0"},
        "optionalDependencies": {"foreign": "1.0.0"},
    }
    lock = {
        "lockfileVersion": "9.0",
        "settings": {"autoInstallPeers": True, "excludeLinksFromLockfile": False, "injectWorkspacePackages": True},
        "importers": {
            ".": {
                "dependencies": {"portable": {"specifier": "1.0.0", "version": "1.0.0"}},
                "optionalDependencies": {"foreign": {"specifier": "1.0.0", "version": "1.0.0"}},
            }
        },
        "packages": {},
        "snapshots": {"portable@1.0.0": {}, "foreign@1.0.0": {"optional": True}},
    }
    archive_path = tmp_path / "portable.tgz"
    with tarfile.open(archive_path, "w:gz") as archive:
        payload = json.dumps({"name": "portable", "version": "1.0.0"}).encode()
        info = tarfile.TarInfo("package/package.json")
        info.size = len(payload)
        archive.addfile(info, io.BytesIO(payload))
    integrity = "sha512-" + base64.b64encode(hashlib.sha512(archive_path.read_bytes()).digest()).decode()
    lock["packages"] = {
        "portable@1.0.0": {"resolution": {"integrity": integrity}},
        "foreign@1.0.0": {"resolution": {"integrity": "sha512-AA=="}, "os": ["not-this-os"]},
    }
    source = tmp_path / "project"
    (source / "package.json").write_text(json.dumps(manifest))
    lock_path = source / "pnpm-lock.yaml"
    original = json.dumps(lock)
    lock_path.write_text(original)
    lf = prepare_deps_module.Lockfile.load(str(lock_path))
    portable = next(p for p in lf.get_packages_meta() if p.key == "portable@1.0.0")
    resource = Path(prepare_deps_module._get_resource_path(args, portable))
    resource.parent.mkdir(parents=True)
    archive_path.rename(resource)
    prepare_deps_module.prepare_deps(args)
    assert lock_path.read_text() == original
    assert not (Path(args.bindir) / "__tarballs__/foreign/-/foreign-1.0.0.tgz").exists()
    pm = PackageManager(args.arcadia_build_root, args.bindir, args.curdir, args.nodejs_bin, args.pm_root)
    pm.create_node_modules(local_cli=False)
    assert (Path(args.bindir) / "node_modules/portable/package.json").is_file()
    assert not (Path(args.bindir) / "node_modules/foreign").exists()
