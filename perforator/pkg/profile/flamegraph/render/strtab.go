package render

type strtab struct {
	s2i map[string]int
	i2s []string
}

func NewStringTable() *strtab {
	return &strtab{
		s2i: make(map[string]int, 1000),
		i2s: make([]string, 0, 1000),
	}
}

func (t *strtab) Add(str string) int {
	res, ok := t.s2i[str]
	if ok {
		return res
	}

	res = len(t.s2i)
	t.s2i[str] = res
	t.i2s = append(t.i2s, str)
	return res
}

func (t *strtab) Table() []string {
	return t.i2s
}
