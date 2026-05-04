RECURSE(
    agent
    binproc
    cli
    gc
    jvm_scanner
    migrate
    offline_processing
    proxy
    quality_monitoring
    storage
    web
)

IF (NOT OPENSOURCE)
    RECURSE(
        yandex-specific
    )
ENDIF()
