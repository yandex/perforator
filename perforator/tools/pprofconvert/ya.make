PROGRAM()

SRCS(
    main.cpp
)

PEERDIR(
    perforator/lib/profile

    library/cpp/digest/murmur
    library/cpp/dwarf_backtrace
    library/cpp/dwarf_backtrace/registry
    library/cpp/terminate_handler
)

END()
