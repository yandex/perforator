package copy

func ZeroTerminatedString(s []byte) string {
	i := 0
	for i < len(s) && s[i] != 0 {
		i++
	}
	return string(s[:i])
}
