import hashlib
import json

from devtools.frontend_build_platform.nots.builder.cli.main import __write_output_meta


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
