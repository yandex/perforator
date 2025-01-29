package env

import (
	"fmt"
	"strings"
)

const (
	envLabelPrefix        = "env:"
	envMatcherFieldPrefix = "env."
)

func BuildEnvLabelKey(envKey string) string {
	return envLabelPrefix + envKey
}

func BuildEnvKeyFromLabelKey(envLabelKey string) (string, bool) {
	return strings.CutPrefix(envLabelKey, envLabelPrefix)
}

func BuildConcatenatedEnv(key string, value string) string {
	return fmt.Sprintf("%v=%v", key, value)
}

func IsEnvMatcherField(matcherField string) bool {
	return strings.HasPrefix(matcherField, envMatcherFieldPrefix)
}

func BuildEnvKeyFromMatcherField(matcherField string) (string, bool) {
	return strings.CutPrefix(matcherField, envMatcherFieldPrefix)
}
