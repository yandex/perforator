import argparse


class YesNoAction(argparse.Action):
    """
    Read boolean values from the configuration language (yes, no, 0, 1, true, false, [empty])
    """

    def __init__(self, option_strings, dest, nargs=None, **kwargs):
        if nargs is not None:
            raise ValueError("nargs not allowed")
        super().__init__(option_strings, dest, **kwargs)

    def __call__(self, parser, namespace, values: str, option_string=None):
        match values.lower():
            case 'yes' | 'true' | 'on' | '1':
                value = True
            case _:
                value = False

        setattr(namespace, self.dest, value)
