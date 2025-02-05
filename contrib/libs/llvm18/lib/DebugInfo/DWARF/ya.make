# Generated by devtools/yamaker.

LIBRARY()

VERSION(18.1.8)

LICENSE(Apache-2.0 WITH LLVM-exception)

LICENSE_TEXTS(.yandex_meta/licenses.list.txt)

PEERDIR(
    contrib/libs/llvm18
    contrib/libs/llvm18/lib/BinaryFormat
    contrib/libs/llvm18/lib/Object
    contrib/libs/llvm18/lib/Support
    contrib/libs/llvm18/lib/TargetParser
)

ADDINCL(
    contrib/libs/llvm18/lib/DebugInfo/DWARF
)

NO_COMPILER_WARNINGS()

NO_UTIL()

SRCS(
    DWARFAbbreviationDeclaration.cpp
    DWARFAcceleratorTable.cpp
    DWARFAddressRange.cpp
    DWARFCompileUnit.cpp
    DWARFContext.cpp
    DWARFDataExtractor.cpp
    DWARFDebugAbbrev.cpp
    DWARFDebugAddr.cpp
    DWARFDebugArangeSet.cpp
    DWARFDebugAranges.cpp
    DWARFDebugFrame.cpp
    DWARFDebugInfoEntry.cpp
    DWARFDebugLine.cpp
    DWARFDebugLoc.cpp
    DWARFDebugMacro.cpp
    DWARFDebugPubTable.cpp
    DWARFDebugRangeList.cpp
    DWARFDebugRnglists.cpp
    DWARFDie.cpp
    DWARFExpression.cpp
    DWARFFormValue.cpp
    DWARFGdbIndex.cpp
    DWARFListTable.cpp
    DWARFLocationExpression.cpp
    DWARFTypePrinter.cpp
    DWARFTypeUnit.cpp
    DWARFUnit.cpp
    DWARFUnitIndex.cpp
    DWARFVerifier.cpp
)

END()
