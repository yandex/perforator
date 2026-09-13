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
    (bindir / "pnpm-workspace.yaml").write_text("packages:\n  - .\n")
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
    assert set((bindir / "pnpm-workspace.yaml").read_text().split()) >= {".", "../peer"}

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
    (peer / "package.json").write_text(json.dumps({"name": "@test/peer"}))
    tarball = peer / "__tarballs__" / "runtime-1.0.0.tgz"
    tarball.parent.mkdir()
    tarball.write_bytes(b"peer tarball")
    for directory in (consumer, peer):
        lockfile = Lockfile(str(directory / "pnpm-lock.yaml"))
        lockfile.data = {"lockfileVersion": "9.0", "importers": {".": {}}}
        if directory == peer:
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
        lf.data = {"lockfileVersion": "9.0", "importers": {".": importer}}
        lf.write()
    lockfile = Lockfile(str(consumer / "pnpm-lock.yaml"))
    lockfile.data = {"lockfileVersion": "9.0", "importers": {".": {"dependencies": {"long": {"version": "5.3.2"}}}}}
    generator = TsProtoGenerator(SimpleNamespace(bindir=str(consumer)))
    generator._add_peer_snapshot(lockfile, str(peer), set())
    assert lockfile.data["snapshots"] == {
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


def test_refresh_generated_peer_snapshot_preserves_consumer_dependencies(tmp_path, monkeypatch):
    from build.plugins.lib.nots.package_manager import Lockfile

    build_root = tmp_path / "build"
    consumer = build_root / "app"
    peer = build_root / "proto"
    nested = build_root / "nested"
    custom = build_root / "custom"
    for directory in (consumer, peer, nested, custom):
        directory.mkdir(parents=True)
    (consumer / "package.json").write_text(json.dumps({"dependencies": {"@test/alias": "workspace:../proto"}}))
    (consumer / "pnpm-workspace.yaml").write_text("packages:\n  - .\n  - ../proto\n  - ../nested\n  - ../custom\n")
    # A custom package's source manifest need not be present in distbuild.
    # Its existing snapshot must remain untouched without the auto-proto marker.
    (custom / "package.json").write_text(json.dumps({"name": "@test/custom"}))
    (custom / pm_constants.OUTPUT_TAR_UUID_FILENAME).write_text("output.tar:uuid")
    for directory, name, dependencies in (
        (peer, "@test/proto", {"@test/nested": "workspace:../nested"}),
        (nested, "@test/nested", {}),
    ):
        (directory / "package.json").write_text(
            json.dumps({"name": name, "dependencies": dependencies, "nots": {"tsProtoAuto": True}})
        )
        (directory / pm_constants.OUTPUT_TAR_UUID_FILENAME).write_text("output.tar:uuid")
        lf = Lockfile(str(directory / "pnpm-lock.yaml"))
        lf.data = {"lockfileVersion": "9.0", "importers": {".": {}}}
        lf.write()
    lf = Lockfile(str(consumer / "pnpm-lock.yaml"))
    root_importer = {
        "dependencies": {"@test/alias": {"specifier": "workspace:../proto", "version": "@test/proto@file:../proto"}}
    }
    lf.data = {
        "lockfileVersion": "9.0",
        "importers": {".": root_importer},
        "snapshots": {
            "@test/proto@file:../proto": {"dependencies": {"stale": "1.0.0"}},
            "@test/custom@file:../custom": {"dependencies": {"keep": "1.0.0"}},
        },
    }
    lf.write()
    monkeypatch.setattr(
        "devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator.extract_all_output_tars",
        lambda *args, **kwargs: None,
    )
    generator = TsProtoGenerator(
        SimpleNamespace(arcadia_root=str(tmp_path / "source"), arcadia_build_root=str(build_root), bindir=str(consumer))
    )
    generator.refresh_generated_peer_lockfile()
    result = Lockfile.load(str(consumer / "pnpm-lock.yaml"))
    assert result.get_importers()["."] == root_importer
    assert result.data["snapshots"]["@test/proto@file:../proto"] == {
        "dependencies": {"@test/nested": "file:../nested"}
    }
    assert result.data["snapshots"]["@test/nested@file:../nested"] == {}
    assert result.data["snapshots"]["@test/custom@file:../custom"] == {"dependencies": {"keep": "1.0.0"}}
