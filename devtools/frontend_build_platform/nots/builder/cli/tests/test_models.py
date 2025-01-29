import argparse

import pytest

from devtools.frontend_build_platform.nots.builder.cli.models import YesNoAction


@pytest.mark.parametrize(
    'value, expected',
    [
        # True
        ('yes', True),
        ('YES', True),
        ('1', True),
        ('on', True),
        ('true', True),
        ('True', True),
        # False
        ('no', False),
        ('NO', False),
        ('0', False),
        ('off', False),
        ('false', False),
        ('False', False),
    ],
)
def test_yes_no_action(value, expected):
    # arrange
    parser = argparse.ArgumentParser()
    parser.add_argument("--trace", action=YesNoAction)

    # act
    args = parser.parse_args(['--trace', value])

    # assert
    assert args.trace is expected
