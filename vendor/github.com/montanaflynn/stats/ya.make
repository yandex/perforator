GO_LIBRARY()

LICENSE(MIT)

VERSION(v0.7.1)

SRCS(
    correlation.go
    cumulative_sum.go
    data.go
    describe.go
    deviation.go
    distances.go
    doc.go
    entropy.go
    errors.go
    geometric_distribution.go
    legacy.go
    load.go
    max.go
    mean.go
    median.go
    min.go
    mode.go
    norm.go
    outlier.go
    percentile.go
    quartile.go
    ranksum.go
    regression.go
    round.go
    sample.go
    sigmoid.go
    softmax.go
    sum.go
    util.go
    variance.go
)

GO_TEST_SRCS(
    errors_test.go
    util_test.go
)

GO_XTEST_SRCS(
    correlation_test.go
    # cumulative_sum_test.go
    # data_test.go
    describe_test.go
    deviation_test.go
    distances_test.go
    # entropy_test.go
    examples_test.go
    geometric_distribution_test.go
    legacy_test.go
    load_test.go
    # max_test.go
    # mean_test.go
    # median_test.go
    # min_test.go
    # mode_test.go
    nist_test.go
    norm_test.go
    outlier_test.go
    # percentile_test.go
    quartile_test.go
    ranksum_test.go
    regression_test.go
    round_test.go
    sample_test.go
    sigmoid_test.go
    softmax_test.go
    # sum_test.go
    test_utils_test.go
    variance_test.go
)

END()

RECURSE(
    # examples
    gotest
)
