import logging
import sys
import warnings

import click
from click.utils import strip_ansi


def reduce_noisy_logs():
    tracker_logger = logging.getLogger('yandex_tracker_client.objects')
    tracker_logger.setLevel(logging.WARNING)

    # Supress warnings from libraries https://st.yandex-team.ru/FBP-494
    warnings.filterwarnings('ignore', module='contrib.python')
    warnings.filterwarnings('ignore', module='library.python')
    warnings.filterwarnings('ignore', module='sandbox.common')
    warnings.filterwarnings('ignore', module='ya.yalibrary')
    warnings.filterwarnings('ignore', module='devtools.ya')


def init_logging(verbose: bool):
    log_format = " ".join(
        [
            click.style("%(asctime)s", dim=True),
            click.style("%(levelname)7s", fg='cyan'),
            click.style("%(relativeCreated)dms", dim=True),
            click.style("%(name)s", bold=True),
            click.style("%(funcName)s:%(lineno)d", underline=True),
            "%(message)s",
        ]
    )

    if not sys.stdout.isatty():
        log_format = strip_ansi(log_format)

    logging.basicConfig(
        format=log_format,
        level=logging.DEBUG if verbose else logging.WARNING,
        stream=sys.stderr
    )
    reduce_noisy_logs()
