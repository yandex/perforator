GO_TEST_FOR(perforator/pkg/storage/profile/meta/clickhouse)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    SIZE(MEDIUM)

    DATA(
        arcadia/perforator/cmd/migrate/migrations/clickhouse
    )

    INCLUDE(${ARCADIA_ROOT}/library/recipes/zookeeper/recipe.inc)
    INCLUDE(${ARCADIA_ROOT}/library/recipes/clickhouse/recipe.inc)
ENDIF()

END()
