import logging
from pprint import pformat
from typing import Any

import click

prefixes_map = {
    'build.internal.plugins._lib.': 'BUILD._lib',
    'devtools.frontend_build_platform.libraries.': 'FBP.libs',
    'devtools.frontend_build_platform.erm.': 'FBP.erm',
    'devtools.frontend_build_platform.nots.': 'FBP.nots',
    'devtools.frontend_build_platform.quantum_arc.': 'FBP.quantum_arc',
}


def get_logger(name: str):
    for prefix, shortening in prefixes_map.items():
        if name.startswith(prefix):
            prefix_len = len(prefix)
            name = f'<{shortening}>.' + name[prefix_len:]
            break

    return logging.getLogger(name)


def __custom_dict_format(d: dict[str, Any], use_colors: bool) -> str:
    result: list[str] = list()
    for k, v in d.items():
        k_styled = click.style(k, fg='blue') if use_colors else k
        v_styled = click.style(v, fg='green') if use_colors else v
        result.append(f'{k_styled}={v_styled}')

    return ', '.join(result)


def safe_log_dict(locals_dump: dict[str, Any], use_pformat=False, use_colors=False):
    secrets = {'token', 'key', 'secret', 'password'}
    keys_to_ignore = {'self', 'cls', 'env'}  # `env` contains tokens or other sensitive data

    def is_secret(key: str):
        for s in secrets:
            if key.lower().endswith(s):
                return True

        return False

    result: dict[str, Any] = dict()
    for k, v in locals_dump.items():
        if k.startswith('_') or k in keys_to_ignore:
            continue
        sanitized_value = repr('***') if is_secret(k) else repr(v)
        result[k] = sanitized_value

    return pformat(result, compact=True) if use_pformat else __custom_dict_format(result, use_colors)
