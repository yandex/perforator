"""Adapt build lockfiles for generated proto packages without merging workspaces."""

import copy
import os
import re

from build.plugins.lib.nots.package_manager import Lockfile, PackageJson, utils as pm_utils

_RUNTIME_KEYS = (PackageJson.DEP_KEY, PackageJson.OPT_DEP_KEY)
_LOCAL_REFERENCE = re.compile(r"(?:file|link):[^()]+")


def rebase_proto_specifier(value: str, source_dir: str, target_dir: str) -> str:
    for prefix in (PackageJson.WORKSPACE_SCHEMA, "file:", "link:"):
        if value.startswith(prefix):
            path = value[len(prefix) :]
            if prefix == PackageJson.WORKSPACE_SCHEMA and not path.startswith((".", "/")):
                return value
            relative = os.path.relpath(os.path.normpath(os.path.join(source_dir, path)), target_dir)
            return prefix + (relative if relative.startswith(".") else "./" + relative)
    return value


def relocate_proto_lockfile(lockfile: Lockfile, target_dir: str) -> None:
    """Relocate a build copy, including local IDs nested inside peer contexts."""
    source_dir = os.path.dirname(lockfile.path)

    def path(value):
        return os.path.relpath(os.path.normpath(os.path.join(source_dir, value)), target_dir)

    def reference(value):
        return _LOCAL_REFERENCE.sub(
            lambda match: match[0].split(":", 1)[0] + ":" + path(match[0].split(":", 1)[1]), value
        )

    def dependencies(metadata):
        for key in (*_RUNTIME_KEYS, PackageJson.DEV_DEP_KEY):
            for name, entry in metadata.get(key, {}).items():
                if isinstance(entry, dict):
                    entry["version"] = reference(entry["version"])
                    if "specifier" in entry:
                        entry["specifier"] = rebase_proto_specifier(entry["specifier"], source_dir, target_dir)
                else:
                    metadata[key][name] = reference(entry)

    data = copy.deepcopy(lockfile.data)
    importers = data.get("importers", {})
    for importer in importers.values():
        dependencies(importer)
    # Only the root importer belongs to the generated package. Peer workspaces
    # are injected packages, never additional install roots.
    data["importers"] = {".": importers.get(".", {})}
    for section in ("packages", "snapshots"):
        relocated = {}
        for package_id, metadata in data.get(section, {}).items():
            resolution = metadata.get("resolution", {})
            if "directory" in resolution:
                resolution["directory"] = path(resolution["directory"])
            tarball = resolution.get("tarball", "")
            if tarball.startswith("file:"):
                resolution["tarball"] = reference(tarball)
            dependencies(metadata)
            relocated[reference(package_id)] = metadata
        data[section] = relocated
    lockfile.data = data
    lockfile.path = pm_utils.build_lockfile_path(target_dir)


class ProtoLockfileAdapter:
    """Import only the runtime closure required by explicitly selected proto peers."""

    def __init__(self, lockfile: Lockfile):
        self.lockfile = lockfile
        self.directory = os.path.dirname(lockfile.path)
        self.peers = set()
        self.copied = {}

    def add_peer(self, directory: str) -> str:
        directory = os.path.normpath(directory)
        package = PackageJson.load(pm_utils.build_pj_path(directory))
        relative = os.path.relpath(directory, self.directory)
        package_id = f"{package.data['name']}@file:{relative}"
        if directory in self.peers:
            return package_id
        self.peers.add(directory)
        path = pm_utils.build_lockfile_path(directory)
        source = Lockfile.load(path) if os.path.isfile(path) else Lockfile(path)
        if source.data is None:
            source.data = {"lockfileVersion": "9.0"}
        relocate_proto_lockfile(source, self.directory)
        importer = source.get_importers().get(".", {})
        snapshot = {}
        for key in _RUNTIME_KEYS:
            resolved = {}
            for name, specifier in package.data.get(key, {}).items():
                version = importer.get(key, {}).get(name, {}).get("version")
                if specifier.startswith(PackageJson.WORKSPACE_SCHEMA):
                    dependency_dir = os.path.normpath(
                        os.path.join(directory, specifier[len(PackageJson.WORKSPACE_SCHEMA) :])
                    )
                    # Seed the resolved context and complete package metadata before
                    # recursively refreshing the generated package's runtime edges.
                    if version is not None and "(" in version:
                        self._copy_dependency(source, name, version)
                    dependency_id = self.add_peer(dependency_dir)
                    # Preserve the resolved peer context and alias, if present.
                    if version is None:
                        version = (
                            dependency_id[len(name) + 1 :] if dependency_id.startswith(name + "@") else dependency_id
                        )
                elif version is None:
                    raise ValueError(f"Cannot resolve {name} of TS_PROTO peer {directory}: missing lockfile entry")
                else:
                    self._copy_dependency(source, name, version)
                resolved[name] = version
            if resolved:
                snapshot[key] = resolved
        packages = self.lockfile.data.setdefault("packages", {})
        snapshots = self.lockfile.data.setdefault("snapshots", {})
        packages.setdefault(package_id, {})["resolution"] = {"directory": relative, "type": "directory"}
        # Existing consumers may resolve the same generated package in multiple
        # peer contexts. Refresh each while retaining its contextual peer edges.
        contexts = [key for key in snapshots if key == package_id or key.startswith(package_id + "(")]
        for context in set(contexts + [package_id]):
            contextual = copy.deepcopy(snapshot)
            for name in package.data.get(PackageJson.PEER_DEP_KEY, {}):
                version = snapshots.get(context, {}).get(PackageJson.DEP_KEY, {}).get(name)
                if version is None:
                    version = importer.get(PackageJson.DEP_KEY, {}).get(name, {}).get("version")
                    if version is not None:
                        self._copy_dependency(source, name, version)
                if version is not None:
                    contextual.setdefault(PackageJson.DEP_KEY, {})[name] = version
            snapshots[context] = contextual
        return package_id

    def _copy_dependency(self, source: Lockfile, name: str, version: str) -> None:
        if version.startswith("link:"):
            return
        snapshots = source.data.get("snapshots", {})
        snapshot_id = f"{name}@{version}"
        if snapshot_id not in snapshots and version in snapshots:
            snapshot_id = version  # npm alias
        copied = self.copied.setdefault(source, set())
        if snapshot_id in copied:
            return
        copied.add(snapshot_id)
        package_id = snapshot_id.split("(", 1)[0]
        packages = source.data.get("packages", {})
        if snapshot_id not in snapshots or package_id not in packages:
            raise ValueError(
                f"Cannot resolve TS_PROTO dependency {snapshot_id}: missing package or snapshot in {source.path}"
            )
        self._insert("packages", package_id, packages[package_id])
        self._insert("snapshots", snapshot_id, snapshots[snapshot_id])
        for key in _RUNTIME_KEYS:
            for dependency, resolution in snapshots[snapshot_id].get(key, {}).items():
                self._copy_dependency(source, dependency, resolution)

    def _insert(self, section: str, key: str, metadata: dict) -> None:
        target = self.lockfile.data.setdefault(section, {})
        if key in target:
            existing = copy.deepcopy(target[key])
            incoming = copy.deepcopy(metadata)
            # Optional records reachability, not a different resolution. A required
            # path in either graph keeps the combined dependency required.
            existing_optional = existing.pop("optional", False)
            incoming_optional = incoming.pop("optional", False)
            if section == "packages":
                # Each build package has its own tarball store. Equal integrity
                # allows retaining the consumer's already prepared location.
                old_resolution = existing.get("resolution", {})
                new_resolution = incoming.get("resolution", {})
                if old_resolution.get("integrity") and old_resolution.get("integrity") == new_resolution.get(
                    "integrity"
                ):
                    old_resolution.pop("tarball", None)
                    new_resolution.pop("tarball", None)
            if existing != incoming:
                raise ValueError(f"Conflicting TS_PROTO {section} entry {key}; cannot preserve both resolutions")
            if not (existing_optional and incoming_optional):
                target[key].pop("optional", None)
        else:
            target[key] = copy.deepcopy(metadata)
