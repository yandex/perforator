GO_LIBRARY()

PEERDIR(
    # stdlib imports used in interface signatures must be declared explicitly
    ${GOSTD}/math/big
    perforator/pkg/storage/cluster_top/aggregated
    perforator/pkg/storage/util
    perforator/proto/perforator
)

GO_MOCKGEN_FROM(perforator/pkg/storage/cluster_top)

GO_MOCKGEN_SOURCE(models.go)

END()
