#!/usr/bin/env python3
"""
Extract CPython struct offsets for a given version.

Usage:
    extract_offsets --cpython-version 3.12.1
    extract_offsets --cpython-version 3.11.0 --output-format plain
"""

import argparse

from perforator.internal.linguist.common.scripts.extract_offsets_lib import run


def main():
    parser = argparse.ArgumentParser(description="Extract CPython struct offsets")
    parser.add_argument("--cpython-version", required=True, help="CPython version (e.g., 3.12.1)")
    parser.add_argument("--offsets-c", required=True, help="Path to offsets.c")
    parser.add_argument("--output-format", choices=["json", "plain"], default="json")
    parser.add_argument("--force-download", action="store_true")
    parser.add_argument("--cache-dir", default="~/.offset_sources")
    args = parser.parse_args()

    run(
        repo="https://github.com/python/cpython.git",
        tag=f"v{args.cpython_version}",
        offsets_c=args.offsets_c,
        include_dirs=["Include", "."],
        configure_commands=[
            ["./configure", "--without-doc-strings", "--config-cache"],
        ],
        cache_dir=args.cache_dir,
        force=args.force_download,
        output_format=args.output_format,
    )


if __name__ == "__main__":
    main()
