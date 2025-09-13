GO_TEST_FOR(perforator/pkg/storage/custom_profiles/meta/clickhouse)

IF (NOT OPENSOURCE)
    SIZE(MEDIUM)

    DATA(
        arcadia/perforator/cmd/migrate/migrations/clickhouse
    )

    INCLUDE(${ARCADIA_ROOT}/library/recipes/zookeeper/recipe.inc)
    INCLUDE(${ARCADIA_ROOT}/library/recipes/clickhouse/recipe.inc)
ENDIF()

END()
