JAVA_PROGRAM()

WITH_JDK()
JDK_VERSION(24)

PEERDIR(perforator/internal/linguist/jvm/unwind/jni)

JAVA_SRCS(SRCDIR src/main/java **/*.java)

IF(DEFINED JNI_GENERATE)
    JAVAC_FLAGS(-h ${ARCADIA_ROOT}/perforator/internal/linguist/jvm/unwind/jni)
ENDIF()

END()
