import logging
import re
import time
from contextlib import contextmanager

RESOURCE_GLOBAL_PATTERN = re.compile(r"\$[\w\d_]+_RESOURCE_GLOBAL")
logger = logging.getLogger("nots builder recipe")


def resolve_recipe_arg(
    arg: str,
    build_root: str,
    source_root: str,
    global_resources: dict[str, str],
) -> str:
    arg = arg.replace("$ARCADIA_BUILD_ROOT", build_root).replace("$ARCADIA_ROOT", source_root)

    def replace_global_resource(match):
        key = match.group(0)[1:]
        if key not in global_resources:
            raise KeyError("Global resource '{}' is not provided.".format(key))
        return global_resources[key]

    return RESOURCE_GLOBAL_PATTERN.sub(replace_global_resource, arg)


@contextmanager
def duration_log(label: str):
    started_at = time.monotonic()
    logger.debug("%s: started", label)
    failed = True
    try:
        yield
        failed = False
    finally:
        result = "failed" if failed else "finished"
        logger.debug("%s: %s in %s seconds", label, result, int(time.monotonic() - started_at))
