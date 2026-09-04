GO_TEST_FOR(perforator/pkg/storage/cluster_top/aggregated)

# This test requires library/recipes, which is not supported in the oss repo
IF (NOT OPENSOURCE)
    SIZE(MEDIUM)

    # Exercise async deduplication with dependent materialized views on 26.3.
    IF (NOT CLICKHOUSE_VERSION)
        SET(CLICKHOUSE_VERSION 26.3.16.16)
    ENDIF()

    DATA(
        arcadia/perforator/cmd/migrate/migrations/clickhouse
    )

    INCLUDE(${ARCADIA_ROOT}/library/recipes/zookeeper/recipe.inc)
    INCLUDE(${ARCADIA_ROOT}/library/recipes/clickhouse/recipe.inc)
ENDIF()

END()
