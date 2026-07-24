import json

from devtools.frontend_build_platform.nots.builder.api.generators.ts_proto_generator import (
    generate_ts_proto_auto_package,
)


def test_generate_ts_proto_auto_package_does_not_copy_tsconfigs(tmp_path):
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
    (deps_bindir / "tsconfig.json").write_text("{}")

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
    assert not (bindir / "tsconfig.json").exists()
