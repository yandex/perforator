import importlib
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
        pm_script="pnpm",
        inject_peers=False,
        ts_proto_auto_deps_path=None,
        tarballs_store="__tarballs__",
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

    monkeypatch.setattr(prepare_deps_module, "PackageManager", PackageManager)

    with pytest.raises(
        prepare_deps_module.PackageManagerError,
        match=r"pnpm-lock\.yaml is out of date: package\.json declares no dependencies.*stale",
    ):
        prepare_deps_module.prepare_deps(args)


def test_prepare_deps_accepts_dependency_free_lockfile(monkeypatch, tmp_path):
    args = _prepare_deps_args(tmp_path)
    (tmp_path / "project" / "pnpm-lock.yaml").write_text("lockfileVersion: '9.0'\nimporters:\n  .: {}\n")

    class PackageManager:
        def __init__(self, **kwargs):
            pass

        def load_package_json_from_dir(self, path):
            return SimpleNamespace(has_dependencies=lambda: False)

        def load_lockfile(self, path):
            return prepare_deps_module.Lockfile.load(path)

        def build_workspace(self, tarballs_store, local_cli):
            pass

    monkeypatch.setattr(prepare_deps_module, "PackageManager", PackageManager)
    monkeypatch.setattr(
        prepare_deps_module,
        "_copy_tarballs",
        lambda *args: pytest.fail("tarballs must not be copied without package.json dependencies"),
    )

    prepare_deps_module.prepare_deps(args)
