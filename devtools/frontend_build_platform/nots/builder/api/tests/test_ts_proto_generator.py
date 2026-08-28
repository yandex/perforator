import json

from devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator import (
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
    assert package_json["dependencies"] == {"runtime": "1.0.0"}
    assert package_json["devDependencies"] == {"compiler": "2.0.0"}
    assert "scripts" not in package_json


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
    )

    assert '"$PROTOC"' in command
    assert "env=browser" in command
    assert '"$ARCADIA_ROOT/project/proto/input.proto"' in command
    assert '"$ARCADIA_BUILD_ROOT/library/typescript/ts-proto-deps/tsconfig.cjs.json"' in command
    assert "--project tsconfig.cjs.json" in command
    assert "--project tsconfig.esm.json" in command
    assert "ignored.json" not in command
