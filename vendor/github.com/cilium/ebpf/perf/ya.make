GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.17.1)

# requires root

GO_SKIP_TESTS(
    TestPerfReader
    TestPerfReaderLostSample
    TestPerfReaderClose
    TestReaderSetDeadline
    TestPause
    TestCreatePerfEvent
    TestPerfEventRing
)

SRCS(
    doc.go
    reader.go
    ring.go
)

END()
