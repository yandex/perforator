from build.plugins.lib.nots.package_manager import PackageJson
from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace


def test_workspace_set_from_package_json():
    ws = PnpmWorkspace(path="/packages/foo/pnpm-workspace.yaml")
    pj = PackageJson(path="/packages/foo/package.json")
    pj.data = {
        "dependencies": {
            "@a/bar": "workspace:../bar",
        },
        "devDependencies": {
            "@a/baz": "workspace:../../another/baz",
        },
        "peerDependencies": {
            "@a/qux": "workspace:../../another/qux",
        },
        "optionalDependencies": {
            "@a/quux": "workspace:../../another/quux",
        },
    }

    ws.set_from_package_json(pj)

    assert ws.packages == set()


def test_workspace_set_from_package_json_writes_only_supported_settings(tmp_path):
    workspace_path = tmp_path / "pnpm-workspace.yaml"
    package_json = PackageJson(path=str(tmp_path / "package.json"))
    package_json.data = {
        "pnpm": {
            "overrides": {"foo": "1.0.0"},
            "packageExtensions": {"bar": {"peerDependencies": {"baz": "2.0.0"}}},
            "patchedDependencies": {"qux@3.0.0": "patches/qux.patch"},
            "peerDependencyRules": {"ignoreMissing": ["react"]},
            "onlyBuiltDependencies": ["esbuild", "sharp"],
            "neverBuiltDependencies": ["core-js"],
            "ignoredBuiltDependencies": ["sharp"],
            "allowNonAppliedPatches": True,
            "managePackageManagerVersions": False,
            "packageManagerStrict": False,
            "packageManagerStrictVersion": True,
        }
    }
    workspace = PnpmWorkspace(path=str(workspace_path))

    workspace.set_from_package_json(package_json)
    workspace.write()

    written_workspace = PnpmWorkspace.load(str(workspace_path))
    assert written_workspace.packages == set()
    assert written_workspace.settings == {
        "overrides": {"foo": "1.0.0"},
        "packageExtensions": {"bar": {"peerDependencies": {"baz": "2.0.0"}}},
        "patchedDependencies": {"qux@3.0.0": "patches/qux.patch"},
        "peerDependencyRules": {"ignoreMissing": ["react"]},
    }


def test_workspace_ignores_unknown_settings(tmp_path):
    package_json = PackageJson(path=str(tmp_path / "package.json"))
    package_json.data = {
        "pnpm": {
            "onlyBuiltDependenciesFile": "allowed-builds.json",
            "ignoreDepScripts": True,
            "ignorePatchFailures": True,
            "useNodeVersion": "22.0.0",
            "executionEnv": {"nodeVersion": "22.0.0"},
            "auditConfig": {"ignoreCves": ["CVE-2025-0001"]},
        }
    }
    workspace = PnpmWorkspace(path=str(tmp_path / "pnpm-workspace.yaml"))

    workspace.set_from_package_json(package_json)

    assert workspace.settings == {}


def test_workspace_read_write_preserves_settings(tmp_path):
    workspace_path = tmp_path / "pnpm-workspace.yaml"
    workspace_path.write_text("packages:\n  - .\noverrides:\n  foo: 1.0.0\n")

    workspace = PnpmWorkspace.load(str(workspace_path))
    workspace.packages.add("../bar")
    workspace.write()

    written_workspace = PnpmWorkspace.load(str(workspace_path))
    assert written_workspace.packages == {".", "../bar"}
    assert written_workspace.settings == {"overrides": {"foo": "1.0.0"}}


def test_workspace_merge_imports_catalogs_without_peer_settings_or_packages():
    ws1 = PnpmWorkspace(path="/packages/foo/pnpm-workspace.yaml")
    ws1.settings = {"overrides": {"own": "1.0.0"}}
    ws2 = PnpmWorkspace(path="/another/baz/pnpm-workspace.yaml")
    ws2.packages = {".", "../qux"}
    ws2.settings = {"overrides": {"peer": "2.0.0"}}
    ws2.catalogs = {"another/common": {"colors": "1.4.0"}}

    ws1.merge(ws2)

    assert ws1.packages == set()
    assert ws1.settings == {"overrides": {"own": "1.0.0"}}
    assert ws1.catalogs == ws2.catalogs
