import hashlib
import json
from types import SimpleNamespace

from devtools.frontend_build_platform.nots.builder.cli.main import __write_output_meta, _postprocess_output


def test_write_output_meta(tmp_path):
    output_tar = tmp_path / "output.tar"
    output_tar.write_bytes(b"archive contents")

    __write_output_meta(
        str(tmp_path),
        str(output_tar),
        ["nested\\dist/", "build", "./build"],
    )

    meta = json.loads((tmp_path / "output.tar.uuid").read_text())
    assert meta == {
        "outputTar": {"sha256": hashlib.sha256(b"archive contents").hexdigest()},
        "buildOutputs": ["build", "nested/dist"],
    }


def test_write_prefixed_output_meta(tmp_path):
    output_tar = tmp_path / "output.tar"
    output_tar.write_bytes(b"prefixed archive")
    __write_output_meta(str(tmp_path), str(output_tar), ["dist", "types"], "app/service")
    meta = json.loads((tmp_path / "output.tar.uuid").read_text())
    assert meta["outputTar"]["prefix"] == "app/service"
    assert meta["buildOutputs"] == ["dist", "types"]


def test_library_for_writes_metadata_beside_its_archive(tmp_path):
    bindir = tmp_path / "project"
    bindir.mkdir()
    output_dir = tmp_path / "project__aux"
    output_dir.mkdir()
    output = output_dir / "output.tar"
    output.write_bytes(b"archive")
    _postprocess_output(
        SimpleNamespace(
            bindir=str(bindir), prepared_deps_moddir="project", output_file=str(output), node_modules_bundle=False
        ),
        ["dist"],
    )
    assert (output_dir / "output.tar.uuid").is_file()
    assert not (bindir / "output.tar.uuid").exists()
