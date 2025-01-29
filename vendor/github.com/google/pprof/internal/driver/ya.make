GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20240424215950-a892ee059fd6)

SRCS(
    cli.go
    commands.go
    config.go
    driver.go
    driver_focus.go
    fetch.go
    flags.go
    interactive.go
    options.go
    settings.go
    stacks.go
    svg.go
    tagroot.go
    tempfile.go
    webhtml.go
    webui.go
)

GO_TEST_SRCS(
    # browser_test.go
    driver_test.go
    fetch_test.go
    interactive_test.go
    settings_test.go
    tagroot_test.go
    tempfile_test.go
    # webui_test.go
)

GO_EMBED_PATTERN(html)

GO_TEST_EMBED_PATTERN(testdata/testfixture.js)

GO_TEST_EMBED_PATTERN(testdata/testflame.js)

END()

RECURSE(
    gotest
)
