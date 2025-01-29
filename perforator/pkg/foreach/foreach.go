package foreach

func Filter[T any](slice []T, f func(T) bool) []T {
	res := make([]T, 0, len(slice))
	for _, e := range slice {
		if f(e) {
			res = append(res, e)
		}
	}
	return res
}

func Map[T any, X any](slice []T, f func(T) X) []X {
	res := make([]X, 0, len(slice))
	for _, e := range slice {
		res = append(res, f(e))
	}
	return res
}
