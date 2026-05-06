#!/usr/bin/env bash

# Script to generate types from proto files
# Examples:
# ./scripts/types.sh
# ./scripts/types.sh --protoc my_protoc --lib protobuf/src --dir perforator/proto/perforator

cd "$(dirname "$0")"

PERFORATOR_UI_SRC="$(npm prefix)"
PERFORATOR_SRC="$(dirname $(dirname "$PERFORATOR_UI_SRC"))"

PROTO_PATHS=()

while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            shift
            PROTO_DIR="$1"
            ;;
        -l|--lib)
            shift
            PROTO_PATHS+=("$1")
            ;;
        -c|--protoc)
            shift
            PROTOC="$1"
            ;;
    esac
    shift
done

: ${PROTOC=protoc}
: ${PROTO_DIR="$PERFORATOR_SRC/proto/perforator"}

OPTIONS='env=node,exportCommonSymbols=false,oneof=properties,forceLong=string,esModuleInterop=true,stringEnums=true,onlyTypes=true,useDate=string'
TS_PLUGIN="$PERFORATOR_UI_SRC/node_modules/.bin/protoc-gen-ts_proto"
OUTPUT_DIR="$PERFORATOR_UI_SRC/src/generated"

pnpx "$PROTOC" \
    --plugin "$TS_PLUGIN" \
    --ts_proto_opt "$OPTIONS" \
    --ts_proto_out "$OUTPUT_DIR" \
    $PROTO_DIR/*.proto \
    $(echo "$(for path in ${PROTO_PATHS[@]}; do echo "--proto_path $path"; done)")
