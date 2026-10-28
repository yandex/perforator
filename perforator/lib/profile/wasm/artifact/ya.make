UNION()

BUILD_ONLY_IF(OS_EMSCRIPTEN)

BUNDLE(perforator/lib/profile/wasm NAME profile.raw.wasm)

RUN_PROGRAM(
    contrib/restricted/binaryen/src/tools/wasm-opt
    profile.raw.wasm -O3 -o profile.wasm
    IN profile.raw.wasm
    OUT profile.wasm
)

EXPLICIT_OUTPUTS(profile.wasm)

END()
