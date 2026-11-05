import json
from pathlib import Path
from types import SimpleNamespace

import pytest

from build.plugins.lib.nots.package_manager import constants as pm_constants
from devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator import (
    TsProtoGenerator,
    generate_ts_proto_auto_package,
    make_ts_proto_build_command,
)


def test_generate_ts_proto_auto_package(tmp_path):
    build_root = tmp_path / "build"
    bindir = build_root / "project" / "proto"
    deps_bindir = build_root / "library" / "typescript" / "ts-proto-deps"
    bindir.mkdir(parents=True)
    deps_bindir.mkdir(parents=True)
    (deps_bindir / "package.json").write_text(
        json.dumps(
            {
                "dependencies": {"runtime": "1.0.0"},
                "devDependencies": {"compiler": "2.0.0"},
            }
        )
    )
    generate_ts_proto_auto_package(
        str(build_root),
        str(bindir),
        "project/proto",
        "@yandex-proto/*",
        "library/typescript/ts-proto-deps",
    )

    package_json = json.loads((bindir / "package.json").read_text())
    assert package_json["name"] == "@yandex-proto/project-proto"
    assert package_json["nots"] == {"tsProtoAuto": True}
    assert package_json["dependencies"] == {"runtime": "1.0.0"}
    assert package_json["devDependencies"] == {"compiler": "2.0.0"}
    assert package_json["exports"]["./*"]["require"] == {
        "types": "./build/cjs/generated/project/proto/*.d.ts",
        "default": "./build/cjs/generated/project/proto/*.js",
    }
    assert "scripts" not in package_json
    assert package_json["files"] == ["build", "pnpm-lock.yaml", "ya.make"]
    assert package_json["typesVersions"] == {
        "*": {
            "*": ["build/types/generated/project/proto/*"],
            "generated/*": ["build/types/generated/*"],
        }
    }
    assert not (bindir / "tsconfig.json").exists()


def test_make_ts_proto_build_command_for_auto_package(tmp_path):
    source_root = tmp_path / "source"
    build_root = tmp_path / "build"
    moddir = "project/proto"

    command = make_ts_proto_build_command(
        str(source_root),
        str(build_root),
        str(source_root / moddir),
        [str(source_root), str(build_root)],
        [str(source_root / moddir / "input.proto")],
        ["env=browser"],
        ["ignored.json"],
        True,
        "library/typescript/ts-proto-deps",
        import_mappings={"proto/peer/model.proto": "@test/peer/model"},
    )

    assert '"$PROTOC"' in command
    assert "env=browser" in command
    assert "Mproto/peer/model.proto=@test/peer/model" in command
    assert '"$ARCADIA_ROOT/project/proto/input.proto"' in command
    assert '"$ARCADIA_BUILD_ROOT/library/typescript/ts-proto-deps/tsconfig.cjs.json"' in command
    assert "--project tsconfig.cjs.json --incremental false --composite false --pretty --declaration" in command
    assert "--project tsconfig.esm.json" in command
    assert "ignored.json" not in command


@pytest.mark.parametrize("namespace", ["", "api/proto"])
def test_import_mappings_include_only_built_ts_proto_peers(tmp_path, namespace):
    build_root = tmp_path / "build"
    source_root = tmp_path / "source"
    peer_moddir = (Path(namespace) / "proto/peer").as_posix()
    peer_dir = build_root / peer_moddir
    raw_dir = build_root / "proto/raw"
    generated_file = peer_dir / "build" / "esm" / "generated" / "proto/peer/nested/model.js"
    generated_file.parent.mkdir(parents=True)
    generated_file.write_text("export {}")
    peer_source = source_root / peer_moddir / "nested/model.proto"
    peer_source.parent.mkdir(parents=True)
    peer_source.write_text('syntax = "proto3";')
    # The peer artifact also contains a recursively generated raw dependency.
    raw_source = source_root / namespace / "proto/raw/model.proto"
    raw_source.parent.mkdir(parents=True)
    raw_source.write_text('syntax = "proto3";')
    raw_generated = peer_dir / "build/esm/generated/proto/raw/model.js"
    raw_generated.parent.mkdir(parents=True)
    raw_generated.write_text("export {}")
    raw_dir.mkdir(parents=True)
    (peer_dir / pm_constants.OUTPUT_TAR_UUID_FILENAME).write_text("output.tar:uuid")
    (peer_dir / "package.json").write_text(
        json.dumps({"name": "@test/peer", "exports": {"./generated/*": "./build/esm/generated/*.js"}})
    )
    (build_root / "package.json").write_text(json.dumps({"dependencies": {"@test/peer": "workspace:proto/peer"}}))

    generator = TsProtoGenerator(
        SimpleNamespace(
            arcadia_root=str(source_root),
            arcadia_build_root=str(build_root),
            bindir=str(build_root),
            proto_paths=[str(source_root / namespace), str(build_root)],
            proto_peers=[peer_moddir, "proto/raw"],
        )
    )

    assert generator._get_import_mappings() == {
        "proto/peer/nested/model.proto": "@test/peer/generated/proto/peer/nested/model",
    }
    reexports_file = generator.write_peer_reexports()
    assert json.loads((build_root / reexports_file).read_text()) == {
        "proto/peer/nested/model.proto": "@test/peer/generated/proto/peer/nested/model",
        "proto/raw/model.proto": "@test/peer/generated/proto/raw/model",
    }
    (build_root / "package.json").write_text(json.dumps({"dependencies": {}}))
    assert generator._get_import_mappings() == {}


def test_prepare_peer_libraries_generates_complete_metadata(tmp_path, monkeypatch):
    build_root = tmp_path / "build"
    bindir = build_root / "proto" / "consumer"
    peer_dir = build_root / "proto" / "peer"
    bindir.mkdir(parents=True)
    peer_dir.mkdir(parents=True)
    (bindir / "package.json").write_text(json.dumps({"name": "@test/consumer", "dependencies": {"runtime": "1.0.0"}}))
    (bindir / "pnpm-workspace.yaml").write_text("packages: []\n")
    (bindir / "pnpm-lock.yaml").write_text(
        "lockfileVersion: '9.0'\n"
        "importers:\n"
        "  .:\n"
        "    dependencies:\n"
        "      runtime:\n"
        "        specifier: 1.0.0\n"
        "        version: 1.0.0\n"
        "      '@foreign/peer':\n"
        "        specifier: workspace:../../foreign\n"
        "        version: file:../../foreign\n"
        "  ../../foreign: {}\n"
        "packages:\n"
        "  '@foreign/peer@file:../../foreign':\n"
        "    resolution:\n"
        "      directory: ../../foreign\n"
        "      type: directory\n"
        "  runtime@1.0.0: {}\n"
        "snapshots:\n"
        "  '@foreign/peer@file:../../foreign': {}\n"
        "  runtime@1.0.0: {}\n"
    )
    (peer_dir / "package.json").write_text(json.dumps({"name": "@test/peer"}))
    (peer_dir / "pnpm-workspace.yaml").write_text("packages:\n  - .\n")
    (peer_dir / "pnpm-lock.yaml").write_text(
        "lockfileVersion: '9.0'\nimporters:\n  .: {}\npackages: {}\nsnapshots: {}\n"
    )
    (peer_dir / pm_constants.OUTPUT_TAR_UUID_FILENAME).write_text("output.tar:uuid")

    generator = TsProtoGenerator(
        SimpleNamespace(
            arcadia_build_root=str(build_root),
            bindir=str(bindir),
            proto_peers=["proto/peer"],
        )
    )
    monkeypatch.setattr(
        "devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator.extract_all_output_tars",
        lambda *args, **kwargs: None,
    )

    generator.prepare_peer_libraries()

    package_json = json.loads((bindir / "package.json").read_text())
    assert package_json["dependencies"] == {"runtime": "1.0.0", "@test/peer": "workspace:../peer"}
    assert package_json["files"] == ["ya.make"]
    assert "packages: []" in (bindir / "pnpm-workspace.yaml").read_text()

    lockfile = (bindir / "pnpm-lock.yaml").read_text()
    assert "@test/peer" in lockfile
    assert "workspace:../peer" in lockfile
    assert "file:../peer" in lockfile
    assert "@foreign/peer" not in lockfile
    assert "../../foreign" not in lockfile


def test_peer_lockfile_preserves_tarball_locations(tmp_path):
    from build.plugins.lib.nots.package_manager import Lockfile

    consumer = tmp_path / "consumer"
    peer = tmp_path / "peer"
    peer.mkdir()
    consumer.mkdir()
    (peer / "package.json").write_text(json.dumps({"name": "@test/peer", "dependencies": {"runtime": "1.0.0"}}))
    tarball = peer / "__tarballs__" / "runtime-1.0.0.tgz"
    tarball.parent.mkdir()
    tarball.write_bytes(b"peer tarball")
    for directory in (consumer, peer):
        lockfile = Lockfile(str(directory / "pnpm-lock.yaml"))
        lockfile.data = {"lockfileVersion": "9.0", "importers": {".": {}}}
        if directory == peer:
            lockfile.data["importers"]["."] = {"dependencies": {"runtime": {"version": "1.0.0"}}}
            lockfile.data["snapshots"] = {"runtime@1.0.0": {}}
            lockfile.data["packages"] = {
                "runtime@1.0.0": {
                    "resolution": {"tarball": "file:__tarballs__/runtime-1.0.0.tgz", "integrity": "sha512-YQ=="}
                }
            }
        lockfile.write()

    generator = TsProtoGenerator(SimpleNamespace(bindir=str(consumer)))
    generator._update_peer_lockfile([str(peer)], {"@test/peer": "workspace:../peer"})
    merged = Lockfile.load(str(consumer / "pnpm-lock.yaml"))
    url = merged.data["packages"]["runtime@1.0.0"]["resolution"]["tarball"]
    assert (consumer / url.removeprefix("file:")).resolve() == tarball
    assert (consumer / url.removeprefix("file:")).read_bytes() == b"peer tarball"


def test_peer_snapshot_uses_own_runtime_and_transitive_workspace_dependencies(tmp_path):
    from build.plugins.lib.nots.package_manager import Lockfile

    build_root = tmp_path / "build"
    consumer = build_root / "consumer"
    peer = build_root / "peer"
    nested = build_root / "nested"
    for directory in (consumer, peer, nested):
        directory.mkdir(parents=True)
    (peer / "package.json").write_text(
        json.dumps(
            {
                "name": "@test/peer",
                "dependencies": {"long": "5.2.3", "@test/nested": "workspace:../nested"},
                "optionalDependencies": {"optional": "1.0.0"},
                "devDependencies": {"typescript": "5.3.3"},
            }
        )
    )
    (nested / "package.json").write_text(json.dumps({"name": "@test/nested", "dependencies": {"protobufjs": "7.2.6"}}))
    for directory, importer in [
        (
            peer,
            {
                "dependencies": {"long": {"version": "5.2.3"}},
                "optionalDependencies": {"optional": {"version": "1.0.0"}},
            },
        ),
        (nested, {"dependencies": {"protobufjs": {"version": "7.2.6"}}}),
    ]:
        lf = Lockfile(str(directory / "pnpm-lock.yaml"))
        versions = {f"{name}@{meta['version']}": {} for deps in importer.values() for name, meta in deps.items()}
        lf.data = {"lockfileVersion": "9.0", "importers": {".": importer}, "packages": versions, "snapshots": versions}
        lf.write()
    lockfile = Lockfile(str(consumer / "pnpm-lock.yaml"))
    lockfile.data = {"lockfileVersion": "9.0", "importers": {".": {"dependencies": {"long": {"version": "5.3.2"}}}}}
    from devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_lockfile import ProtoLockfileAdapter

    ProtoLockfileAdapter(lockfile).add_peer(str(peer))
    assert lockfile.data["snapshots"] == {
        "long@5.2.3": {},
        "optional@1.0.0": {},
        "protobufjs@7.2.6": {},
        "@test/peer@file:../peer": {
            "dependencies": {"long": "5.2.3", "@test/nested": "file:../nested"},
            "optionalDependencies": {"optional": "1.0.0"},
        },
        "@test/nested@file:../nested": {"dependencies": {"protobufjs": "7.2.6"}},
    }


def test_import_mappings_follow_custom_package_exports(tmp_path):
    peer = tmp_path / "proto" / "peer"
    generated = peer / "build/generated/proto/peer/nested/model.js"
    generated.parent.mkdir(parents=True)
    generated.write_text("exports.Model = {}")
    source = peer / "nested/model.proto"
    source.parent.mkdir(parents=True)
    source.write_text('syntax = "proto3";')
    (peer / pm_constants.OUTPUT_TAR_UUID_FILENAME).write_text("output.tar:uuid")
    (peer / "package.json").write_text(
        json.dumps(
            {
                "name": "@test/custom",
                "exports": {"./models/*": "./build/generated/proto/peer/*.js"},
            }
        )
    )
    (tmp_path / "package.json").write_text(json.dumps({"dependencies": {"@test/custom": "workspace:proto/peer"}}))
    generator = TsProtoGenerator(
        SimpleNamespace(
            arcadia_root=str(tmp_path / "source"),
            arcadia_build_root=str(tmp_path),
            bindir=str(tmp_path),
            proto_paths=[str(tmp_path)],
            proto_peers=["proto/peer"],
        )
    )
    assert generator._get_import_mappings() == {"proto/peer/nested/model.proto": "@test/custom/models/nested/model"}


def test_proto_adaptation_copies_only_runtime_closure_and_preserves_inputs(tmp_path):
    import copy
    from build.plugins.lib.nots.package_manager import Lockfile

    consumer = tmp_path / "app"
    peer = tmp_path / "proto" / "peer"
    peer.mkdir(parents=True)
    consumer.mkdir()
    (consumer / "package.json").write_text(json.dumps({"dependencies": {"@test/peer": "workspace:../proto/peer"}}))
    (peer / "package.json").write_text(
        json.dumps(
            {
                "name": "@test/peer",
                "nots": {"tsProtoAuto": True},
                "dependencies": {"alias": "npm:runtime@1.0.0"},
                "devDependencies": {"compiler": "9.0.0"},
            }
        )
    )
    peer_lf = Lockfile(str(peer / "pnpm-lock.yaml"))
    peer_lf.data = {
        "lockfileVersion": "9.0",
        "importers": {
            ".": {
                "dependencies": {"alias": {"version": "runtime@1.0.0(host@file:../../host)"}},
                "devDependencies": {"compiler": {"version": "9.0.0"}},
            },
            "../../unused-importer": {},
        },
        "packages": {
            "runtime@1.0.0": {"resolution": {"tarball": "file:__tarballs__/runtime.tgz", "integrity": "sha512-YQ=="}},
            "host@file:../../host": {"resolution": {"directory": "../../host", "type": "directory"}},
            "optional@1.0.0": {},
            "compiler@9.0.0": {},
            "unused@1.0.0": {},
        },
        "snapshots": {
            "runtime@1.0.0(host@file:../../host)": {
                "dependencies": {"host": "file:../../host"},
                "optionalDependencies": {"optional": "1.0.0"},
            },
            "host@file:../../host": {"dependencies": {"alias": "runtime@1.0.0(host@file:../../host)"}},
            "optional@1.0.0": {},
            "compiler@9.0.0": {},
            "unused@1.0.0": {},
        },
    }
    peer_lf.write()
    peer_bytes = (peer / "pnpm-lock.yaml").read_bytes()
    lf = Lockfile(str(consumer / "pnpm-lock.yaml"))
    lf.data = {
        "lockfileVersion": "9.0",
        "settings": {"autoInstallPeers": True},
        "importers": {
            ".": {
                "dependencies": {
                    "@test/peer": {"specifier": "workspace:../proto/peer", "version": "file:../proto/peer"},
                    "runtime": {"specifier": "2.0.0", "version": "2.0.0"},
                }
            }
        },
        "packages": {"runtime@2.0.0": {"resolution": {"tarball": "file:__tarballs__/keep.tgz"}}},
        "snapshots": {"runtime@2.0.0": {"dependencies": {"keep": "1.0.0"}}},
    }
    lf.write()
    original = copy.deepcopy(lf.data)
    generator = TsProtoGenerator(SimpleNamespace(bindir=str(consumer)))
    generator._update_peer_lockfile([str(peer)], {"@test/peer": "workspace:../proto/peer"})
    result = Lockfile.load(lf.path).data
    assert result["importers"] == original["importers"]
    assert result["settings"] == original["settings"]
    assert result["packages"]["runtime@2.0.0"] == original["packages"]["runtime@2.0.0"]
    assert result["snapshots"]["runtime@2.0.0"] == original["snapshots"]["runtime@2.0.0"]
    assert set(result["packages"]) == {
        "runtime@2.0.0",
        "runtime@1.0.0",
        "host@file:../host",
        "optional@1.0.0",
        "@test/peer@file:../proto/peer",
    }
    assert result["snapshots"]["runtime@1.0.0(host@file:../host)"]["dependencies"] == {"host": "file:../host"}
    assert result["snapshots"]["host@file:../host"]["dependencies"] == {"alias": "runtime@1.0.0(host@file:../host)"}
    assert result["packages"]["host@file:../host"]["resolution"]["directory"] == "../host"
    assert result["packages"]["runtime@1.0.0"]["resolution"]["tarball"] == "file:../proto/peer/__tarballs__/runtime.tgz"
    assert (peer / "pnpm-lock.yaml").read_bytes() == peer_bytes
    # A repeat must produce the same graph and locations.
    generator._update_peer_lockfile([str(peer)], {})
    assert Lockfile.load(lf.path).data == result


def test_auto_workspace_relocates_build_copy_without_changing_source(tmp_path):
    from build.plugins.lib.nots.package_manager import Lockfile
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager

    source = tmp_path / "deps"
    target = tmp_path / "proto" / "generated"
    source.mkdir()
    target.mkdir(parents=True)
    lf = Lockfile(str(source / "pnpm-lock.yaml"))
    lf.data = {
        "lockfileVersion": "9.0",
        "importers": {
            ".": {
                "dependencies": {
                    "local": {"specifier": "workspace:../local", "version": "file:../local(host@file:../host)"},
                }
            }
        },
        "packages": {
            "local@file:../local": {"resolution": {"type": "directory", "directory": "../local"}},
            "registry@1.0.0": {"resolution": {"tarball": "file:__tarballs__/registry.tgz", "integrity": "sha512-YQ=="}},
        },
        "snapshots": {"local@file:../local(host@file:../host)": {"dependencies": {"host": "link:../host"}}},
    }
    lf.write()
    (source / "package.json").write_text(json.dumps({"dependencies": {"local": "workspace:../local"}}))
    generate_ts_proto_auto_package(str(tmp_path), str(target), "proto/generated", "@test/*", "deps")
    assert json.loads((target / "package.json").read_text())["dependencies"] == {"local": "workspace:../../local"}
    original = (source / "pnpm-lock.yaml").read_bytes()
    pm = PackageManager(str(tmp_path), str(target), str(source), None, None)
    pm.build_ts_proto_auto_workspace("deps")
    TsProtoGenerator(SimpleNamespace(bindir=str(target)))._reset_peer_metadata()
    result = Lockfile.load(str(target / "pnpm-lock.yaml")).data
    assert result["importers"]["."]["dependencies"]["local"] == {
        "specifier": "workspace:../../local",
        "version": "file:../../local(host@file:../../host)",
    }
    assert result["packages"]["local@file:../../local"]["resolution"]["directory"] == "../../local"
    assert result["snapshots"]["local@file:../../local(host@file:../../host)"] == {
        "dependencies": {"host": "link:../../host"}
    }
    assert result["packages"]["registry@1.0.0"]["resolution"]["tarball"] == "file:../../deps/__tarballs__/registry.tgz"
    assert (source / "pnpm-lock.yaml").read_bytes() == original


@pytest.mark.parametrize(
    ("runtime_alias", "peer_lockfile"), [("runtime", True), ("runtime", False), ("alias", False)]
)
def test_auto_proto_lockfile_supports_offline_frozen_install(tmp_path, runtime_alias, peer_lockfile):
    import os
    from build.plugins.lib.nots.package_manager import Lockfile
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager
    from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace, pnpm_defaults

    app = tmp_path / "app"
    peer = tmp_path / "peer"
    runtime = tmp_path / "runtime"
    for directory, manifest in (
        (app, {"name": "app", "dependencies": {"proto": "workspace:../peer"}}),
        (peer, {"name": "proto", "version": "1.0.0", "nots": {"tsProtoAuto": True}}),
        (runtime, {"name": "runtime", "version": "1.0.0"}),
    ):
        directory.mkdir()
        (directory / "package.json").write_text(json.dumps(manifest))
        workspace = PnpmWorkspace(str(directory / "pnpm-workspace.yaml"))
        workspace.runtime_settings = pnpm_defaults()
        workspace.write()
    pm = PackageManager(
        build_root=str(tmp_path),
        build_path=str(app),
        sources_path=str(app),
        nodejs_bin_path=os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
        pnpm_root=os.environ["NOTS_TEST_PNPM_ROOT"],
    )
    store = str(tmp_path / "store")
    pm._exec_command(["install", "--lockfile-only", "--offline", "--store-dir", store], cwd=str(app))
    original_importer = Lockfile.load(str(app / "pnpm-lock.yaml")).get_importers()
    manifest = json.loads((peer / "package.json").read_text())
    manifest["dependencies"] = {runtime_alias: "file:../runtime" if peer_lockfile else "workspace:../runtime"}
    (peer / "package.json").write_text(json.dumps(manifest))
    if peer_lockfile:
        pm._exec_command(["install", "--lockfile-only", "--offline", "--store-dir", store], cwd=str(peer))
    original_peer = (peer / "pnpm-lock.yaml").read_bytes() if peer_lockfile else None
    TsProtoGenerator(SimpleNamespace(bindir=str(app)))._update_peer_lockfile([str(peer)], {})
    assert Lockfile.load(str(app / "pnpm-lock.yaml")).get_importers() == original_importer
    pm._exec_command(["install", "--frozen-lockfile", "--offline", "--store-dir", store], cwd=str(app))
    import subprocess

    subprocess.run(
        [
            os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
            "-e",
            f"require.resolve('{runtime_alias}/package.json', {{paths: [require('fs').realpathSync('node_modules/proto')]}})",
        ],
        cwd=str(app),
        check=True,
    )
    if peer_lockfile:
        assert (peer / "pnpm-lock.yaml").read_bytes() == original_peer
    else:
        assert not (peer / "pnpm-lock.yaml").exists()


@pytest.mark.parametrize("consumer_optional", [False, True])
@pytest.mark.parametrize("peer_optional", [False, True])
def test_proto_optional_runtime_is_merged_by_reachability(tmp_path, consumer_optional, peer_optional):
    import os
    import subprocess

    from build.plugins.lib.nots.package_manager import Lockfile
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager
    from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace, pnpm_defaults

    app = tmp_path / "app"
    peer = tmp_path / "peer"
    consumer_key = "optionalDependencies" if consumer_optional else "dependencies"
    peer_key = "optionalDependencies" if peer_optional else "dependencies"
    app_manifest = {"name": "app", "dependencies": {"proto": "workspace:../peer"}}
    app_manifest.setdefault(consumer_key, {})["runtime"] = "file:../runtime"
    for directory, manifest in (
        (app, app_manifest),
        (peer, {"name": "proto", "version": "1.0.0", "nots": {"tsProtoAuto": True}}),
        (
            tmp_path / "runtime",
            {"name": "runtime", "version": "1.0.0", "dependencies": {"leaf": "file:../leaf"}},
        ),
        (tmp_path / "leaf", {"name": "leaf", "version": "1.0.0"}),
    ):
        directory.mkdir()
        (directory / "package.json").write_text(json.dumps(manifest))
        workspace = PnpmWorkspace(str(directory / "pnpm-workspace.yaml"))
        workspace.runtime_settings = pnpm_defaults()
        workspace.write()
    pm = PackageManager(
        build_root=str(tmp_path),
        build_path=str(app),
        sources_path=str(app),
        nodejs_bin_path=os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
        pnpm_root=os.environ["NOTS_TEST_PNPM_ROOT"],
    )
    store = str(tmp_path / "store")
    install = ["install", "--lockfile-only", "--offline", "--store-dir", store]
    pm._exec_command(install, cwd=str(app))
    original_importers = Lockfile.load(str(app / "pnpm-lock.yaml")).get_importers()
    manifest = json.loads((peer / "package.json").read_text())
    manifest[peer_key] = {"runtime": "file:../runtime"}
    (peer / "package.json").write_text(json.dumps(manifest))
    pm._exec_command(install, cwd=str(peer))
    original_peer = (peer / "pnpm-lock.yaml").read_bytes()

    TsProtoGenerator(SimpleNamespace(bindir=str(app)))._update_peer_lockfile([str(peer)], {})

    result = Lockfile.load(str(app / "pnpm-lock.yaml"))
    assert result.get_importers() == original_importers
    for name in ("runtime", "leaf"):
        assert bool(result.data["snapshots"][f"{name}@file:../{name}"].get("optional")) == (
            consumer_optional and peer_optional
        )
    pm._exec_command(["install", "--frozen-lockfile", "--offline", "--store-dir", store], cwd=str(app))
    subprocess.run(
        [
            os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
            "-e",
            "const proto = require('fs').realpathSync('node_modules/proto');"
            "const runtime = require.resolve('runtime/package.json', {paths: [proto]});"
            "require.resolve('leaf/package.json', {paths: [require('path').dirname(runtime)]});",
        ],
        cwd=str(app),
        check=True,
    )
    assert (peer / "pnpm-lock.yaml").read_bytes() == original_peer


def test_proto_workspace_dependency_preserves_resolved_peer_context(tmp_path):
    import os
    import subprocess

    from build.plugins.lib.nots.package_manager import Lockfile
    from devtools.frontend_build_platform.nots.builder.api.package_manager import PackageManager
    from devtools.frontend_build_platform.nots.builder.api.pnpm_workspace import PnpmWorkspace, pnpm_defaults

    app = tmp_path / "app"
    peer = tmp_path / "peer"
    for directory, manifest in (
        (app, {"name": "app", "dependencies": {"proto": "workspace:../peer"}}),
        (peer, {"name": "proto", "version": "1.0.0", "nots": {"tsProtoAuto": True}}),
        (tmp_path / "nested", {"name": "nested", "version": "1.0.0", "peerDependencies": {"host": "*"}}),
        (tmp_path / "host", {"name": "host", "version": "1.0.0"}),
    ):
        directory.mkdir()
        (directory / "package.json").write_text(json.dumps(manifest))
        workspace = PnpmWorkspace(str(directory / "pnpm-workspace.yaml"))
        workspace.runtime_settings = pnpm_defaults()
        workspace.write()
    pm = PackageManager(
        build_root=str(tmp_path),
        build_path=str(app),
        sources_path=str(app),
        nodejs_bin_path=os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
        pnpm_root=os.environ["NOTS_TEST_PNPM_ROOT"],
    )
    store = str(tmp_path / "store")
    install = ["install", "--lockfile-only", "--offline", "--store-dir", store]
    pm._exec_command(install, cwd=str(app))
    manifest = json.loads((peer / "package.json").read_text())
    manifest["dependencies"] = {"nested": "workspace:../nested", "host": "file:../host"}
    (peer / "package.json").write_text(json.dumps(manifest))
    pm._exec_command(install, cwd=str(peer))
    original_peer = (peer / "pnpm-lock.yaml").read_bytes()

    TsProtoGenerator(SimpleNamespace(bindir=str(app)))._update_peer_lockfile([str(peer)], {})

    result = Lockfile.load(str(app / "pnpm-lock.yaml")).data
    assert result["packages"]["nested@file:../nested"]["peerDependencies"] == {"host": "*"}
    assert result["snapshots"]["nested@file:../nested(host@file:../host)"]["dependencies"] == {
        "host": "file:../host"
    }
    pm._exec_command(["install", "--frozen-lockfile", "--offline", "--store-dir", store], cwd=str(app))
    subprocess.run(
        [
            os.path.join(os.environ["NOTS_TEST_NODEJS_ROOT"], "node"),
            "-e",
            "const proto = require('fs').realpathSync('node_modules/proto');"
            "const nested = require.resolve('nested/package.json', {paths: [proto]});"
            "require.resolve('host/package.json', {paths: [require('path').dirname(nested)]});",
        ],
        cwd=str(app),
        check=True,
    )
    assert (peer / "pnpm-lock.yaml").read_bytes() == original_peer


def test_conflicting_runtime_snapshot_does_not_rewrite_consumer(tmp_path):
    from build.plugins.lib.nots.package_manager import Lockfile

    peer = tmp_path / "peer"
    app = tmp_path / "app"
    peer.mkdir()
    app.mkdir()
    (peer / "package.json").write_text(json.dumps({"name": "proto", "dependencies": {"runtime": "1.0.0"}}))
    for directory, dependency in ((peer, "incoming"), (app, "existing")):
        lf = Lockfile(str(directory / "pnpm-lock.yaml"))
        lf.data = {
            "lockfileVersion": "9.0",
            "importers": {".": {"dependencies": {"runtime": {"version": "1.0.0"}}}},
            "packages": {"runtime@1.0.0": {}},
            "snapshots": {"runtime@1.0.0": {"dependencies": {dependency: "1.0.0"}}},
        }
        lf.write()
    original = (app / "pnpm-lock.yaml").read_bytes()
    with pytest.raises(ValueError, match="Conflicting TS_PROTO snapshots entry runtime@1.0.0"):
        TsProtoGenerator(SimpleNamespace(bindir=str(app)))._update_peer_lockfile([str(peer)], {})
    assert (app / "pnpm-lock.yaml").read_bytes() == original
