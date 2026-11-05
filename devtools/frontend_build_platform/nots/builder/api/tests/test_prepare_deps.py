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
        pm_root="pnpm",
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
