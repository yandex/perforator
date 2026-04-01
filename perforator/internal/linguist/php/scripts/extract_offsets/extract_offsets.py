#!/usr/bin/env python3
"""
Extract PHP (Zend Engine) struct offsets for a given version.

Usage:
    extract_offsets --php-version 7.4.0
    extract_offsets --php-version 7.0.0 --output-format plain
"""

import argparse

from perforator.internal.linguist.common.scripts.extract_offsets_lib import run


def main():
    parser = argparse.ArgumentParser(description="Extract PHP struct offsets")
    parser.add_argument("--php-version", required=True, help="PHP version (e.g., 7.4.0)")
    parser.add_argument("--offsets-c", default="offsets.c", help="Path to offsets.c")
    parser.add_argument("--output-format", choices=["json", "plain"], default="json")
    parser.add_argument("--force-download", action="store_true")
    parser.add_argument("--cache-dir", default="~/.offset_sources")
    args = parser.parse_args()

    run(
        repo="https://github.com/php/php-src.git",
        tag=f"php-{args.php_version}",
        offsets_c=args.offsets_c,
        include_dirs=[".", "main", "Zend", "TSRM"],
        configure_commands=[
            ["./buildconf", "--force"],
            ["./configure", "--disable-all", "--disable-cgi", "--config-cache"],
        ],
        cache_dir=args.cache_dir,
        force=args.force_download,
        output_format=args.output_format,
    )


if __name__ == "__main__":
    main()
