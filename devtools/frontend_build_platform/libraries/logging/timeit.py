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

start_time = time.monotonic()
start_time_ns = round(time.monotonic_ns() / 1000)


class TraceEvent:
    name: str
    ph: str  # 'B' (begin) or 'E' (end)
    ts: int

    def __init__(self, name: str, start: bool):
        super().__init__()
        self.name = name
        self.ph = 'B' if start else 'E'
        self.ts = round(time.monotonic_ns() / 1000 - start_time_ns)

    def to_json(self):
        return dict(
            cat='PERF',
            name=self.name,
            ph=self.ph,
            pid=os.getpid(),
            tid=threading.get_ident(),
            ts=self.ts,
        )


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
    json_indent_size = 2
    level = 0
    enabled = False
    use_dumper = False
    use_stderr = False
    silent = False

    current_record = TimeitRecord('root', 0)
    trace_events: list[TraceEvent] = []

    def start_level(self, name: str):
        self.level += 1

        if self.use_dumper:
            new_record = TimeitRecord(name, self.level, parent=self.current_record)
            self.current_record.children.append(new_record)

            self.current_record = new_record

            self.trace_events.append(TraceEvent(self.current_record.name, start=True))

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
            raise RuntimeError('please check @timeit logic, you must not measure the function with `timeit_options.enable()` inside')

        root_record.duration = time.monotonic() - start_time

        self.__safe_dump_json(json_filename, root_record.to_json())

    def dump_trace(self, json_filename: str):
        if not self.use_dumper:
            raise RuntimeError('please call `timeit_options.enabled(use_dumper=True)` for using `dump_trace` feature')

        self.__safe_dump_json(json_filename, dict(traceEvents=[e.to_json() for e in self.trace_events]))

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
    def timed(*args, **kw) -> RT:
        if not options.enabled:
            return method(*args, **kw)

        now = time.monotonic()

        # noinspection PyUnresolvedReferences
        name = _extract_name(method.__repr__())

        if not options.silent:
            click.secho(f"{options.indent()}<{name}>", fg='magenta', err=options.use_stderr)

        options.start_level(name)

        result = method(*args, **kw)

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
