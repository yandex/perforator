package querylang

func mapKeys[K comparable](in ...map[K]struct{}) []K {
	size := 0
	for i := range in {
		size += len(in[i])
	}
	result := make([]K, 0, size)

	for i := range in {
		for k := range in[i] {
			result = append(result, k)
		}
	}

	return result
}

func hasCondition(conditions []*Condition, hasF func(item *Condition) bool) bool {
	for i := range conditions {
		if hasF(conditions[i]) {
			return true
		}
	}

	return false
}

func filterConditions(conditions []*Condition, hasF func(item *Condition, index int) bool) []*Condition {
	result := make([]*Condition, 0, len(conditions))

	for i := range conditions {
		if hasF(conditions[i], i) {
			result = append(result, conditions[i])
		}
	}

	return result
}

func setDifference[V comparable](in []V, drop ...V) []V {
	var result []V
	dropMap := make(map[V]struct{}, len(drop))

	for _, d := range drop {
		dropMap[d] = struct{}{}
	}

	for _, v := range in {
		if _, ok := dropMap[v]; !ok {
			result = append(result, v)
		}
	}

	return result
}

func setIntersection[V comparable](left, right []V) []V {
	var result []V
	leftMap := make(map[V]struct{}, len(left))

	for _, l := range left {
		leftMap[l] = struct{}{}
	}

	for _, r := range right {
		if _, ok := leftMap[r]; ok {
			result = append(result, r)
		}
	}

	return result
}

func setUnion[V comparable](left, right []V) []V {
	resultMap := make(map[V]struct{}, len(left)+len(right))
	result := make([]V, 0, len(left)+len(right))

	for _, l := range left {
		resultMap[l] = struct{}{}
		result = append(result, l)
	}

	for _, r := range right {
		if _, ok := resultMap[r]; !ok {
			result = append(result, r)
		}
	}

	return result
}
