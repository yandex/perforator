GO_LIBRARY()

LICENSE(Apache-2.0)

VERSION(v0.0.0-20250317173921-a4b03ec1a45e)

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
    driver_test.go
    fetch_test.go
    interactive_test.go
    settings_test.go
    tagroot_test.go
    tempfile_test.go
    # webui_test.go
)

GO_EMBED_PATTERN(html)

END()

RECURSE(
    gotest
)
