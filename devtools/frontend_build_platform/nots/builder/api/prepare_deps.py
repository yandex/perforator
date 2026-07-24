import os
from dataclasses import dataclass

import library.python.fs
from build.plugins.lib.nots.package_manager import (
    Lockfile,
    PackageJson,
    PackageManager,
    PackageManagerError,
    utils as pm_utils,
)
from yalibrary.fetcher.uri_parser import parse_resource_uri

from .models import BaseOptions
from .generators.ts_proto_generator import generate_ts_proto_auto_package


@dataclass
class PrepareDepsOptions(BaseOptions):
    resource_root: str | None
    """Root location of build node resources"""

    tarballs_store: str
    """Path to tarballs store, related to $CURDIR"""

    ts_proto_auto_deps_path: str | None
    """Path to ts-proto deps module, related to $ARCADIA_ROOT"""

    ts_proto_auto_package_name: str | None
    """Generated TS_PROTO package name"""


def prepare_deps(args: PrepareDepsOptions):
    pm = PackageManager(
        build_root=args.arcadia_build_root,
        build_path=args.bindir,
        sources_path=args.curdir,
        nodejs_bin_path=args.nodejs_bin,
        script_path=args.pm_script,
        inject_peers=args.inject_peers,
    )

    if args.ts_proto_auto_deps_path:
        generate_ts_proto_auto_package(
            args.arcadia_build_root,
            args.bindir,
            args.moddir,
            args.ts_proto_auto_package_name,
            args.ts_proto_auto_deps_path,
        )
        pm.build_ts_proto_auto_workspace(args.ts_proto_auto_deps_path)
    else:
        has_dependencies = pm.load_package_json_from_dir(args.curdir).has_dependencies()
        lockfile_path = pm_utils.build_lockfile_path(args.curdir)
        if not has_dependencies and os.path.exists(lockfile_path):
            _validate_dependency_free_lockfile(pm.load_lockfile(lockfile_path))

        pm.build_workspace(args.tarballs_store, args.local_cli)
        if has_dependencies and not args.local_cli and os.path.exists(lockfile_path):
            _copy_tarballs(args, pm.load_lockfile(lockfile_path))


def _validate_dependency_free_lockfile(lockfile: Lockfile) -> None:
    importer = lockfile.get_importers().get(".", {})
    dependency_keys = (
        PackageJson.DEP_KEY,
        PackageJson.DEV_DEP_KEY,
        PackageJson.PEER_DEP_KEY,
        PackageJson.OPT_DEP_KEY,
    )
    dependencies = sorted(
        dependency for dependency_key in dependency_keys for dependency in importer.get(dependency_key, {})
    )
    if dependencies:
        raise PackageManagerError(
            "{} is out of date: package.json declares no dependencies, "
            "but its importer contains {}. Update the lockfile with "
            "`ya tool nots update-lockfile`.".format(lockfile.path, ", ".join(dependencies))
        )


def _get_resource_path(args: PrepareDepsOptions, pkg) -> str:
    parsed_uri = parse_resource_uri(pkg.to_uri())
    return os.path.join(args.resource_root, "http", parsed_uri.resource_id, "resource")


def _copy_tarballs(args: PrepareDepsOptions, lf: Lockfile):
    # Tarballs can be used several times in a single pnpm-lock.yaml by different keys
    # We need to remove duplicates
    tarball_paths = {pkg.tarball_path: pkg for pkg in lf.get_packages_meta()}

    for pkg in tarball_paths.values():
        resource_tarball_path = _get_resource_path(args, pkg)
        local_tarball_path = os.path.join(args.bindir, args.tarballs_store, pkg.tarball_path)
        os.makedirs(os.path.dirname(local_tarball_path), exist_ok=True)
        library.python.fs.hardlink_or_copy(resource_tarball_path, local_tarball_path)
