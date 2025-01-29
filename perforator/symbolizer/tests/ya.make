GTEST()

PEERDIR(
    perforator/symbolizer/lib/symbolize
)

SRCS(
    test.cpp
)

DATA(
    arcadia/perforator/symbolizer/tests/libsample.so.elf
    arcadia/perforator/symbolizer/tests/sample_program.elf
)

END()
