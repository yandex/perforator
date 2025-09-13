import os
from dataclasses import dataclass

import library.python.fs
from build.plugins.lib.nots.package_manager import get_package_manager_type, BaseLockfile
from yalibrary.fetcher.uri_parser import parse_resource_uri

from .models import BaseOptions


@dataclass
class PrepareDepsOptions(BaseOptions):
    resource_root: str | None
    """Root location of build node resources"""

    tarballs_store: str
    """Path to tarballs store, related to $CURDIR"""

    ts_proto_auto_deps_path: str | None
    """Path to ts-proto deps module, related to $ARCADIA_ROOT"""


def prepare_deps(args: PrepareDepsOptions):
    PackageManager = get_package_manager_type(args.pm_type)

    pm = PackageManager(
        build_root=args.arcadia_build_root,
        build_path=args.bindir,
        sources_path=args.curdir,
        nodejs_bin_path=args.nodejs_bin,
        script_path=args.pm_script,
    )

    if args.ts_proto_auto_deps_path:
        pm.build_ts_proto_auto_workspace(args.ts_proto_auto_deps_path)
    else:
        pm.build_workspace(args.tarballs_store, args.local_cli)
        if not args.local_cli:
            _copy_tarballs(args, pm.load_lockfile_from_dir(args.curdir))


def _get_resource_path(args: PrepareDepsOptions, pkg) -> str:
    parsed_uri = parse_resource_uri(pkg.to_uri())
    return os.path.join(args.resource_root, "http", parsed_uri.resource_id, "resource")


def _copy_tarballs(args: PrepareDepsOptions, lf: BaseLockfile):
    # Tarballs can be used several times in a single pnpm-lock.yaml by different keys
    # We need to remove duplicates
    tarball_paths = {pkg.tarball_path: pkg for pkg in lf.get_packages_meta()}

    for pkg in tarball_paths.values():
        resource_tarball_path = _get_resource_path(args, pkg)
        local_tarball_path = os.path.join(args.bindir, args.tarballs_store, pkg.tarball_path)
        os.makedirs(os.path.dirname(local_tarball_path), exist_ok=True)
        library.python.fs.hardlink_or_copy(resource_tarball_path, local_tarball_path)
