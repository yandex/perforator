import { execFileSync } from "node:child_process";
import fs from "node:fs";
import path from "node:path";

function parseOptions(args) {
    const options = new Map();
    for (let index = 0; index < args.length; index += 2) {
        options.set(args[index], args[index + 1]);
    }

    const required = [
        "--node", "--bsdtar", "--archiver", "--rsync", "--synchronizer",
        "--builder", "--recipe", "--runtime", "--module",
    ];
    for (const name of required) {
        if (!options.get(name)) {
            throw new Error(`Missing ${name}`);
        }
    }
    if (Boolean(options.get("--libatomic")) !== Boolean(options.get("--runtime-libatomic"))) {
        throw new Error("Expected --libatomic and --runtime-libatomic together");
    }
    return options;
}

function stageBundle(options, modulePath) {
    fs.mkdirSync(path.dirname(modulePath), { recursive: true });

    if (options.has("--archive")) {
        const archive = options.get("--archive");
        const output = fs.openSync(modulePath, "w");
        try {
            // bsdtar treats build/nots.mjs and ./build/nots.mjs as the same path.
            execFileSync(options.get("--bsdtar"), ["-xOf", archive, "--", "build/nots.mjs"], {
                stdio: ["ignore", output, "inherit"],
            });
        } finally {
            fs.closeSync(output);
        }
        return;
    }

    if (options.has("--bundle")) {
        fs.copyFileSync(options.get("--bundle"), modulePath);
        return;
    }

    throw new Error("Expected --archive or --bundle");
}

function stageExecutable(source, destination) {
    fs.copyFileSync(source, destination);
    fs.chmodSync(destination, 0o755);
}

function writeLauncher(filename, staged, mode) {
    const libraryPath = staged.libatomic ? 'export LD_LIBRARY_PATH="$dir${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}"\n' : "";
    // Keep yatest's --env-file argument out of Node's early option parser.
    const contents = `#!/bin/sh
set -eu
dir=$(CDPATH= cd -- "\${0%/*}" && pwd)
export NOTS_BSDTAR_PATH="\${NOTS_BSDTAR_PATH:-$dir/${path.basename(staged.archiver)}}"
export NOTS_RSYNC_PATH="$dir/${path.basename(staged.synchronizer)}"
${libraryPath}exec "$dir/${path.basename(staged.runtime)}" -- "$dir/${path.basename(staged.module)}" ${mode} "$@"
`;
    fs.writeFileSync(filename, contents);
    fs.chmodSync(filename, 0o755);
}

function main(args) {
    const options = parseOptions(args);
    const staged = {
        runtime: options.get("--runtime"),
        module: options.get("--module"),
        archiver: options.get("--archiver"),
        synchronizer: options.get("--synchronizer"),
        libatomic: options.get("--libatomic"),
    };

    stageBundle(options, staged.module);
    stageExecutable(options.get("--node"), staged.runtime);
    stageExecutable(options.get("--bsdtar"), staged.archiver);
    stageExecutable(options.get("--rsync"), staged.synchronizer);
    if (staged.libatomic) {
        fs.copyFileSync(staged.libatomic, options.get("--runtime-libatomic"));
    }

    writeLauncher(options.get("--builder"), staged, "--internal-builder");
    writeLauncher(options.get("--recipe"), staged, "--internal-recipe");
}

main(process.argv.slice(2));
