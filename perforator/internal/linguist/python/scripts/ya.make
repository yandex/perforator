IF (NOT OPENSOURCE)
    RECURSE(
        gen_docker_images
    )
ENDIF()

RECURSE(
    extract_offsets
    load_offsets
)
