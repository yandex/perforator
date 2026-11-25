import fs from "node:fs";
import path from "node:path";

// Platform hooks shared by every project in one mount.
// pnpm 12.23 hashes local pnpmfiles only; a global-only hook has this stable checksum.
const GLOBAL_ONLY_CHECKSUM = "sha256-47DEQpj8HBSa+/TImW+5JCeuQeRkm5NMpJWZG3hSuFU=";

// Normalize only spelling changes that preserve the actual filesystem target.
// Each pnpm hook invocation has its own process, so normalize config and lockfiles
// independently; pnpm still compares their effective overrides afterwards.
function normalizeOverrides(overrides, directory) {
    return Object.fromEntries(
        Object.entries(overrides).map(([selector, specifier]) => {
            const match = /^(file:|link:|workspace:)(\.\.?\/.*)$/.exec(specifier);
            if (!match) return [selector, specifier];
            const target = path.relative(directory, path.resolve(directory, match[2]));
            const rawPath = `${directory}${path.sep}${match[2]}`;
            if (
                !fs.existsSync(rawPath) ||
                !fs.existsSync(path.resolve(directory, target)) ||
                fs.realpathSync.native(rawPath) !== fs.realpathSync.native(path.resolve(directory, target))
            ) {
                return [selector, specifier];
            }
            return [selector, match[1] + (target.startsWith(".") ? target : `./${target}`)];
        })
    );
}

export const hooks = {
    updateConfig(config) {
        if (!config.overrides) return config;
        return { ...config, overrides: normalizeOverrides(config.overrides, config.lockfileDir ?? config.dir) };
    },
    preResolution({ wantedLockfile, currentLockfile, lockfileDir }) {
        // Old source lockfiles predate the platform hook. Bridge only a missing field,
        // in memory; pnpm writes it normally when the user updates their lockfile.
        // User pnpmfiles produce a different checksum and still require a lockfile update.
        for (const lockfile of [wantedLockfile, currentLockfile]) {
            if (!lockfile) continue;
            if (lockfile.overrides) {
                lockfile.overrides = normalizeOverrides(lockfile.overrides, lockfileDir);
            }
            if (lockfile.lockfileVersion !== undefined && lockfile.pnpmfileChecksum === undefined) {
                lockfile.pnpmfileChecksum = GLOBAL_ONLY_CHECKSUM;
            }
        }
    },
    beforePacking(pkg) {
        // Allow packing workspace dependencies that are themselves private.
        return { ...pkg, private: true };
    },
};
