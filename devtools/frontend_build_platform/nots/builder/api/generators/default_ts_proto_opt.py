from ..utils import parse_opt_to_dict

# Options are written in this format for the sake of grepability,
# so it is easy to find where "oneof=unions" comes from
DEFAULT_TS_PROTO_OPT = parse_opt_to_dict(
    [
        "env=node",
        "exportCommonSymbols=false",
        "oneof=unions",
        "forceLong=long",
        "esModuleInterop=true",
    ]
)
