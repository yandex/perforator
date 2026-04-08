PY3_PROGRAM(load_offsets)

DEPENDS(
    perforator/internal/linguist/python/scripts/extract_offsets
)

PY_SRCS(
    MAIN load_offsets.py
)

END()
