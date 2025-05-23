import argparse
import os
import shutil
import subprocess
import tarfile
import typing as t


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser()
    parser.add_argument('--curdir', required=True)
    parser.add_argument('--bindir', required=True)
    parser.add_argument('--node-dir', required=True)
    parser.add_argument('--pnpm-dir', required=True)
    return parser.parse_args()


def bytes2str(text: bytes) -> str:
    return text.decode('utf-8')


def run_command(cmd: t.List[str]) -> None:
    process = subprocess.Popen(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE)
    out, err = process.communicate()
    if process.returncode:
        message = (
            f'Process `{process}` exited with code ${process.returncode}'
            f'\n\nstderr:\n{bytes2str(err)}'
            f'\n\nstdout:\n{bytes2str(out)}'
            f'\n\ncwd: {os.getcwd()}'
        )
        raise RuntimeError(message)
    print(bytes2str(out))


def main():
    args = parse_args()

    original_ui = os.path.join(args.curdir, '..')
    ui = os.path.join(args.bindir, 'ui')
    app = os.path.join(args.bindir, 'ui', 'app')
    shutil.copytree(
        original_ui,
        ui,
        ignore=shutil.ignore_patterns('node_modules', 'dist'),
    )

    os.chdir(app)

    # for `node` executable file
    os.environ['PATH'] = args.node_dir + ':' + os.environ.get('PATH', '')

    node = os.path.join(args.node_dir, 'node')
    pnpm = os.path.join(args.pnpm_dir, 'node_modules', 'pnpm', 'dist', 'pnpm.cjs')

    run_command([node, pnpm, 'install'])
    run_command([node, pnpm, '--filter', '../packages/**', 'run', 'build'])
    run_command([node, pnpm, 'run', 'build'])

    with tarfile.open(os.path.join(args.bindir, 'output.tar'), 'w') as tar:
        tar.add(os.path.join(app, 'dist'), arcname='dist')


if __name__ == '__main__':
    main()
