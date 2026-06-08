package scheduler

import "github.com/yandex/perforator/perforator/pkg/profilequerylang"

const discoverJobsProfileFiltersWhere = profilequerylang.CPOIDLabel + " = '' AND coalesce(nullIf(pod_id, ''), node_id) != ''"
