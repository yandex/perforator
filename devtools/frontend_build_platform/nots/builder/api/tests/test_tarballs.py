import base64
import hashlib
import io
import json
import os
import tarfile
from types import SimpleNamespace

import pytest

from devtools.frontend_build_platform.nots.builder.api.create_node_modules import _link_tarballs
from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager
from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace


@pytest.mark.parametrize("algorithm", ["sha1", "sha512"])
@pytest.mark.parametrize("explicit_url", [False, True])
@pytest.mark.parametrize("global_store", [False, True])
@pytest.mark.parametrize("context", ["module", "auto_prepare", "auto_build", "library_for", "ram"])
def test_distbuild_installs_original_registry_lockfile_offline(
    tmp_path, algorithm, explicit_url, global_store, context
):
    source = tmp_path / "source" / "app"
    project = tmp_path / "build" / "app"
    source.mkdir(parents=True)
    project.mkdir(parents=True)
    archive_path = project / "__tarballs__" / "@probe" / "tool" / "-" / "tool-1.0.0.tgz"
    archive_path.parent.mkdir(parents=True)
    with tarfile.open(archive_path, "w:gz") as archive:
        for name, content in {
            "package.json": json.dumps({"name": "@probe/tool", "version": "1.0.0", "main": "index.js"}),
            "index.js": "module.exports = 42;\n",
        }.items():
            data = content.encode()
            info = tarfile.TarInfo("package/" + name)
            info.size = len(data)
            archive.addfile(info, io.BytesIO(data))
    integrity = algorithm + "-" + base64.b64encode(hashlib.new(algorithm, archive_path.read_bytes()).digest()).decode()
    resolution = {"integrity": integrity}
    if explicit_url:
        resolution["tarball"] = "https://npm.yandex-team.ru/@probe%2ftool/-/tool-1.0.0.tgz?rbtorrent=unused"
    manifest = {"name": "app", "version": "1.0.0", "dependencies": {"@probe/tool": "1.0.0"}}
    (source / "package.json").write_text(json.dumps(manifest))
    lock = {
        "lockfileVersion": "9.0",
        "settings": {"autoInstallPeers": True, "excludeLinksFromLockfile": False, "injectWorkspacePackages": True},
        "importers": {".": {"dependencies": {"@probe/tool": {"specifier": "1.0.0", "version": "1.0.0"}}}},
        "packages": {"@probe/tool@1.0.0": {"resolution": resolution}},
        "snapshots": {"@probe/tool@1.0.0": {}},
    }
    (source / "pnpm-lock.yaml").write_text(json.dumps(lock))
    pm = PackageManager(
        build_root=str(tmp_path / "build"),
        build_path=str(project),
        sources_path=str(source),
        sources_root=str(tmp_path / "source"),
        module_path="app",
        nodejs_bin_path=os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
        pnpm_root=os.environ["NOTS_TEST_PNPM_ROOT"],
    )
    pm.build_workspace()
    assert pm.load_lockfile(str(project / "pnpm-lock.yaml")).data == lock
    if context != "module":
        project = tmp_path / ("ram" if context == "ram" else "build") / "generated"
        project.mkdir(parents=True)
        (project / "package.json").write_text(json.dumps(manifest))
        pm.build_path = str(project)
        pm.module_path = "generated"
        pm.build_ts_proto_auto_workspace("app")
        args = SimpleNamespace(
            arcadia_build_root=str(tmp_path / "build"),
            bindir=str(project),
            moddir="app" if context == "ram" else "generated",
        )
        field = {
            "auto_prepare": "ts_proto_auto_deps_path",
            "auto_build": "auto_deps_path",
            "library_for": "prepared_deps_moddir",
        }.get(context)
        if field:
            setattr(args, field, "app")
        _link_tarballs(args)
        _link_tarballs(args)
        assert (project / "__tarballs__").is_symlink()
        assert (project / "__tarballs__").resolve() == archive_path.parents[3]
    original = (project / "pnpm-lock.yaml").read_bytes()
    workspace = PnpmWorkspace(str(project / "pnpm-workspace.yaml"))
    workspace.settings = {"enableGlobalVirtualStore": global_store}
    workspace.write()
    store = tmp_path / "store"
    pm.create_node_modules(local_cli=False, store_dir=str(store))
    assert (project / "node_modules/@probe/tool/index.js").read_text() == "module.exports = 42;\n"
    assert (project / "pnpm-lock.yaml").read_bytes() == original
    # A second install must use the populated store even without the delivered tgz.
    archive_path.unlink()
    pm._exec_command(["install", "--offline", "--frozen-lockfile", "--store-dir", str(store)], cwd=str(project))


def test_store_add_passes_one_directory(tmp_path):
    project = tmp_path / "app"
    directory = project / "__tarballs__"
    (directory / "nested").mkdir(parents=True)
    (directory / "nested/package.tgz").touch()
    (project / "unrelated.tgz").touch()
    pm = PackageManager(str(tmp_path), str(project), str(project), "/unused", "/unused")
    commands = []
    pm._exec_command = lambda args, cwd: commands.append((args, cwd))
    pm._populate_pnpm_store(str(tmp_path / "store"), str(project))
    assert commands == [(["store", "add", "--store-dir", str(tmp_path / "store"), str(directory)], str(project))]
