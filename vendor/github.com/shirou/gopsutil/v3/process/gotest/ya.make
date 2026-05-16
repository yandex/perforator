GO_TEST_FOR(vendor/github.com/shirou/gopsutil/v3/process)

LICENSE(BSD-3-Clause)

VERSION(v3.24.5)

DATA(
    arcadia/vendor/github.com/shirou/gopsutil/v3/process
)

TEST_CWD(vendor/github.com/shirou/gopsutil/v3/process)

IF (NOT OPENSOURCE)
    INCLUDE(${ARCADIA_ROOT}/library/go/test/go_toolchain/recipe.inc)
ENDIF()

GO_SKIP_TESTS(
    Test_Children
    Test_Kill
    Test_IsRunning
    Test_Process_CmdLine
    Test_Process_Exe
    Test_Process_Name
    Test_Process_Nice
)

END()
