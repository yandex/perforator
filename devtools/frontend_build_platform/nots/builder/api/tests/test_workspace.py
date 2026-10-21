import copy

import yaml

from build.plugins.lib.nots.package_manager import PackageJson
from build.plugins.lib.nots.package_manager.common_config import load_tier0_settings
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


def test_workspace_descriptions_cover_policy_and_runtime_settings():
    policy = load_tier0_settings()["pnpm"]
    required = set(policy["defaults"]) | set(policy["supportedCustomSettings"]) | {"packages", "virtualStoreDir"}
    for key in required:
        assert "https://pnpm.io/" in policy["workspaceComments"]["descriptions"][key], key


def test_workspace_comments_preserve_values_and_regenerate_after_runtime_settings(tmp_path):
    policy = load_tier0_settings()["pnpm"]
    comments = policy["workspaceComments"]
    workspace_path = tmp_path / "pnpm-workspace.yaml"
    workspace = PnpmWorkspace(str(workspace_path))
    workspace.settings = {
        "strictPeerDependencies": True,
        "overrides": {"foo@*": "npm:bar@1.0.0"},
        "patchedDependencies": {"foo@1.0.0": "patches/foo.patch"},
        "packageExtensions": {"foo@*": {"peerDependencies": {"bar": "^1"}}},
        "peerDependencyRules": {"ignoreMissing": ["react"]},
    }
    workspace.catalogs = {"project/common": {"colors": "1.4.0"}}
    workspace.write()
    expected = {**workspace.settings, "packages": [], "catalogs": workspace.catalogs}
    assert yaml.safe_load(workspace_path.read_text()) == expected

    restored = PnpmWorkspace.load(str(workspace_path))
    restored.runtime_settings = {
        **policy["defaults"],
        "strictPeerDependencies": True,
        "virtualStoreDir": str(tmp_path / "node_modules/.pnpm"),
        "modulesDir": "../ram/node_modules",
        "packageImportMethod": "copy",
    }
    restored.write()
    text = workspace_path.read_text()
    assert yaml.safe_load(text) == {**expected, **restored.runtime_settings}
    assert text.count("# ВНИМАНИЕ:") == 1
    assert "# НЕ РЕДАКТИРУЙТЕ ЕГО ВРУЧНУЮ" in text
    assert "package.json#pnpm" in text
    assert "nots.commonConfigPath" in text
    assert "workspaceComments:" not in text
    for key in yaml.safe_load(text):
        description = PnpmWorkspace._comment(comments["descriptions"][key])
        assert description in text, key
    for key, rule in (
        ("strictPeerDependencies", "configurable"),
        ("ignoreScripts", "managed"),
        ("catalogs", "catalogs"),
        ("packages", "generated"),
        ("virtualStoreDir", "generated"),
    ):
        assert PnpmWorkspace._comment(comments["descriptions"][key] + "\n" + comments[rule]) + key + ":" in text
    restored.write()
    assert workspace_path.read_text() == text


def test_workspace_preserves_overrides_only_order(tmp_path):
    package_json = PackageJson(str(tmp_path / "package.json"))
    package_json.data = {
        "pnpm": {
            "overrides": {
                "js-yaml@>=3.0.0 <3.13.0": "3.13.0",
                "js-yaml@<3.13.1": "3.13.1",
            }
        }
    }
    original = copy.deepcopy(package_json.data)
    workspace = PnpmWorkspace(str(tmp_path / "pnpm-workspace.yaml"))
    workspace.set_from_package_json(package_json)
    workspace.write()
    written = PnpmWorkspace.load(workspace.path)
    assert list(written.settings["overrides"].items()) == [
        ("js-yaml@>=3.0.0 <3.13.0", "3.13.0"),
        ("js-yaml@<3.13.1", "3.13.1"),
    ]
    assert package_json.data == original
