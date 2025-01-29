package slices

// MapP applies given function to every pointer to element of slice
func MapP[S ~[]T, T, M any](s S, fn func(*T) M) []M {
	if s == nil {
		return []M(nil)
	}
	if len(s) == 0 {
		return make([]M, 0)
	}
	res := make([]M, len(s))
	for i := range s {
		res[i] = fn(&s[i])
	}
	return res
}

// Map applies given function to every value of slice
func Map[S ~[]T, T, M any](s S, fn func(T) M) []M {
	if s == nil {
		return []M(nil)
	}
	if len(s) == 0 {
		return make([]M, 0)
	}
	res := make([]M, len(s))
	for i, v := range s {
		res[i] = fn(v)
	}
	return res
}

// MapE applies given function to every value of slice and return slice or first error
func MapE[S ~[]T, T, M any](s S, fn func(T) (M, error)) ([]M, error) {
	if s == nil {
		return []M(nil), nil
	}
	if len(s) == 0 {
		return make([]M, 0), nil
	}
	res := make([]M, len(s))
	for i, v := range s {
		transformed, err := fn(v)
		if err != nil {
			return nil, err
		}
		res[i] = transformed
	}
	return res, nil
}

// Mutate is like Map, but it prohibits type changes and modifies original slice.
func Mutate[S ~[]T, T any](s S, fn func(T) T) S {
	if len(s) == 0 {
		return s
	}
	for i, v := range s {
		s[i] = fn(v)
	}
	return s
}
