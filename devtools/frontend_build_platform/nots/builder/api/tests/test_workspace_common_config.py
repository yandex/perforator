import json

import yaml

from build.plugins.lib.nots.package_manager import PackageJson
from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace


def manifest(root, module, data):
    filename = root / module / "package.json"
    filename.parent.mkdir(parents=True, exist_ok=True)
    filename.write_text(json.dumps(data))
    return PackageJson.load(str(filename))


def test_workspace_roundtrip_contains_only_final_pnpm_configuration(tmp_path):
    parent = PnpmWorkspace(str(tmp_path / "pnpm-workspace.yaml"))
    parent.catalogs = {"project/common": {"colors": "1.4.0"}}
    parent.settings = {"overrides": {"colors": "1.4.0"}}
    parent.packages = {"."}
    parent.write()
    assert yaml.safe_load((tmp_path / "pnpm-workspace.yaml").read_text()) == {
        "packages": ["."],
        "catalogs": parent.catalogs,
        "overrides": {"colors": "1.4.0"},
    }
    restored = PnpmWorkspace.load(parent.path)
    assert restored.settings == parent.settings
    assert restored.catalogs == parent.catalogs


def test_peer_workspace_merge_does_not_import_peer_local_settings(tmp_path):
    parent = PnpmWorkspace(str(tmp_path / "pnpm-workspace.yaml"))
    parent.settings = {"overrides": {"foo": "1"}}
    peer = PnpmWorkspace(str(tmp_path / "peer/pnpm-workspace.yaml"))
    peer.settings = {"overrides": {"foo": "9", "peer-only": "9"}}
    peer.packages = {".", "../nested"}
    parent.merge(peer)
    assert parent.settings == {"overrides": {"foo": "1"}}
    assert parent.packages == set()


def test_build_workspace_uses_only_target_config(tmp_path):
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager

    root = tmp_path / "sources"
    source = root / "project/target"
    build = tmp_path / "build/project/target"
    manifest(
        root,
        "project/target",
        {
            "name": "target",
            "dependencies": {"peer": "workspace:../peer"},
            "nots": {"commonConfigPath": "../common.yaml"},
            "pnpm": {"overrides": {"duplicate": "2", "local": "3"}, "ignoredOptionalDependencies": ["local-optional"]},
        },
    )
    (root / "project/common.yaml").write_text(
        "catalogs: {project/common: {foo: '1.0.0'}}\n"
        "overrides: {duplicate: '1', shared: '1'}\n"
        "patchedDependencies: {foo: patches/foo.patch}\n"
        "ignoredOptionalDependencies: [common-optional]\n"
    )
    peer_path = tmp_path / "build/project/peer"
    peer_path.mkdir(parents=True)
    peer = PnpmWorkspace(str(peer_path / "pnpm-workspace.yaml"))
    peer.packages = {".", "../transitive"}
    peer.catalogs = {"peer/common": {"foo": "9"}}
    peer.settings = {"overrides": {"shared": "9", "peer-only": "9"}}
    peer.write()
    pm = PackageManager(
        str(tmp_path / "build"),
        str(build),
        str(source),
        None,
        None,
        module_path="project/target",
        sources_root=str(root),
    )
    pm._build_merged_lockfile = lambda *args: None
    pm.build_workspace("__tarballs__", True)
    result = yaml.safe_load((build / "pnpm-workspace.yaml").read_text())
    assert result == {
        "packages": [],
        "ignoredOptionalDependencies": ["common-optional", "local-optional"],
        "catalogs": {"project/common": {"foo": "1.0.0"}, "peer/common": {"foo": "9"}},
        "overrides": {"duplicate": "2", "local": "3", "shared": "1"},
        "patchedDependencies": {"foo": "../patches/foo.patch"},
    }
    assert PackageJson.load(str(source / "package.json")).data["pnpm"] == {
        "overrides": {"duplicate": "2", "local": "3"},
        "ignoredOptionalDependencies": ["local-optional"],
    }


def test_build_workspace_imports_prepared_override_peer_catalogs_without_peer_sources(tmp_path):
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager

    root = tmp_path / "sources"
    source = root / "project/target"
    build_root = tmp_path / "build"
    manifest(
        root,
        "project/target",
        {
            "name": "target",
            "dependencies": {"peer": "1.0.0"},
            "nots": {"commonConfigPath": "../common.yaml"},
        },
    )
    (root / "project/common.yaml").write_text("overrides: {peer: 'workspace:./peer'}\n")
    peer_path = build_root / "project/peer"
    peer_path.mkdir(parents=True)
    peer = PnpmWorkspace(str(peer_path / "pnpm-workspace.yaml"))
    peer.catalogs = {
        "project/peer/common": {"direct": "1.0.0"},
        "project/leaf/common": {"transitive": "2.0.0"},
    }
    peer.settings = {"overrides": {"peer-only": "9"}, "strictPeerDependencies": True}
    peer.write()
    assert not (root / "project/peer/package.json").exists()
    pm = PackageManager(
        str(build_root),
        str(build_root / "project/target"),
        str(source),
        None,
        None,
        module_path="project/target",
        sources_root=str(root),
    )
    pm._build_merged_lockfile = lambda *args: None
    result = pm.build_workspace("__tarballs__", True)
    assert result.catalogs == peer.catalogs
    assert result.settings == {"overrides": {"peer": "workspace:../peer"}}
