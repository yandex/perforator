import dataclasses
import json
import os
import re
import threading
import time
from dataclasses import dataclass
from functools import wraps
from typing import Any, Callable, TypeVar

import click

start_unix_time_ms = time.time()
start_time = time.monotonic()
start_time_ns = time.monotonic_ns()


class TraceEvent:
    name: str
    ph: str  # 'B' (begin) or 'E' (end)
    ts: int

    def __init__(self, name: str, start: bool, args: list[any] = None, kwargs: dict[str, any] = None):
        super().__init__()
        self.name = name
        self.ph = 'B' if start else 'E'
        self.ts = round((time.monotonic_ns() - start_time_ns) / 1000)  # in ms

        if len(args or []) > 0 or len(kwargs or {}) > 0:
            self.args = dict(
                args=self.__repr_arg(args),
                kwargs=self.__repr_arg(kwargs),
            )

    def __repr_arg(self, a: any):
        # Primitive types
        if isinstance(a, str) or isinstance(a, int) or isinstance(a, float) or isinstance(a, bool) or a is None:
            return a

        # Collections
        if isinstance(a, list) or isinstance(a, tuple) or isinstance(a, set):
            return [self.__repr_arg(v) for v in a]

        if isinstance(a, dict):
            return {k: self.__repr_arg(v) for k, v in a.items()}

        # Other
        return repr(a)

    def to_json(self):
        result = dict(cat='PERF', name=self.name, ph=self.ph, pid=os.getpid(), tid=threading.get_ident(), ts=self.ts)
        if hasattr(self, 'args'):
            result['args'] = self.args

        return result


@dataclass
class TimeitRecord:
    name: str
    level: int
    duration = 0
    children: list['TimeitRecord'] = dataclasses.field(default_factory=lambda: [])

    parent: 'TimeitRecord' = None

    def to_json(self):
        jsoned = dict(
            name=self.name,
            level=self.level,
            duration=round(self.duration, 9),
        )
        if self.children:
            jsoned['children'] = [r.to_json() for r in self.children]

        return jsoned


class TimeitOptions:
    indent_size = 4
    json_indent_size = None  # file is not ready to read by eyes)
    level = 0
    enabled = False
    use_dumper = False
    use_stderr = False
    silent = False

    current_record = TimeitRecord('root', 0)
    trace_events: list[TraceEvent] = []

    def start_level(self, name: str, args, kwargs):
        self.level += 1

        if self.use_dumper:
            new_record = TimeitRecord(name, self.level, parent=self.current_record)
            self.current_record.children.append(new_record)

            self.current_record = new_record

            self.trace_events.append(TraceEvent(self.current_record.name, start=True, args=args, kwargs=kwargs))

    def end_level(self, duration: float):
        self.level -= 1
        if self.use_dumper:
            self.trace_events.append(TraceEvent(self.current_record.name, start=False))

            self.current_record.duration = duration

            self.current_record = self.current_record.parent

    def indent(self):
        return ' ' * self.indent_size * self.level

    def enable(self, silent=False, use_dumper=False, use_stderr=False):
        self.enabled = True

        self.silent = silent
        self.use_dumper = use_dumper
        self.use_stderr = use_stderr

    def disable(self):
        self.enabled = False

    def dump_json(self, json_filename: str):
        if not self.use_dumper:
            raise RuntimeError('please call `timeit_options.enabled(use_dumper=True)` for using `dump_json` feature')

        root_record = self.current_record
        if root_record.name != 'root':
            raise RuntimeError(
                'please check @timeit logic, you must not measure the function with `timeit_options.enable()` inside'
            )

        root_record.duration = time.monotonic() - start_time

        self.__safe_dump_json(json_filename, root_record.to_json())

    def dump_trace(self, json_filename: str, otherData: dict = {}):
        if not self.use_dumper:
            raise RuntimeError('please call `timeit_options.enabled(use_dumper=True)` for using `dump_trace` feature')

        trace = dict(
            otherData=dict(
                # useful for merging traces
                start_time=start_unix_time_ms,
                **otherData,
            ),
            traceEvents=[e.to_json() for e in self.trace_events],
        )

        self.__safe_dump_json(json_filename, trace)

    def __safe_dump_json(self, json_filename: str, obj: Any):
        dirname = os.path.dirname(json_filename)
        if not os.path.exists(dirname):
            os.makedirs(dirname, exist_ok=True)

        with open(json_filename, 'w') as f:
            json.dump(obj, f, indent=self.json_indent_size)


options = TimeitOptions()

RT = TypeVar('RT')  # return type


def _extract_name(name: str) -> str:
    # <function ExternalResource.from_json at 0x1100e25f0>
    # <function register_outdated.<locals>.outdated at 0x10b1f2840>

    method_re = re.compile(r'<function ([\w.<>]+?) at 0x[\da-f]+?>')
    match = method_re.match(name)
    if match:
        name = match[1]

    return name


def timeit(method: Callable[..., RT]) -> Callable[..., RT]:
    """
    Decorator for measuring execution time of methods/functions
    """

    @wraps(method)
    def timed(*args, **kwargs) -> RT:
        if not options.enabled:
            return method(*args, **kwargs)

        # noinspection PyUnresolvedReferences
        name = _extract_name(method.__repr__())

        if not options.silent:
            click.secho(f"{options.indent()}<{name}>", fg='magenta', err=options.use_stderr)

        now = time.monotonic()
        options.start_level(name, args=args, kwargs=kwargs)

        result = method(*args, **kwargs)

        duration = time.monotonic() - now

        options.end_level(duration)

        if duration > 1:
            # seconds
            duration_str = click.style("%2.3f s" % duration, fg='red')
        elif duration * 1000 > 1:
            # milliseconds
            duration_str = click.style("%2.0f ms" % (duration * 1e3), fg='yellow')
        else:
            # microseconds
            duration_str = click.style("%2.0f µs" % (duration * 1e6), fg='green')

        if not options.silent:
            click.secho(f"{options.indent()}</{name}> duration {duration_str}", fg='magenta', err=options.use_stderr)

            if options.indent() == '':
                click.echo(err=options.use_stderr)

        return result

    return timed
