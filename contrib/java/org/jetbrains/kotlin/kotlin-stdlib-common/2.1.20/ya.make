# Manually made file as workaround for peerdirs to kotlin-stdlib-common in previous version to bypass it to kotlin-stdlib
EXTERNAL_JAVA_LIBRARY()

VERSION(2.1.20)

LICENSE(Apache-2.0)

ORIGINAL_SOURCE(https://repo1.maven.org/maven2)

PEERDIR(
    contrib/java/org/jetbrains/kotlin/kotlin-stdlib/2.1.20
)

END()
