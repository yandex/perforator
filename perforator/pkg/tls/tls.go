package tls

import "strings"

const (
	tlsLabelPrefix        = "tls:"
	tlsMatcherFieldPrefix = "tls."
	tlsVariablePrefix     = "perforator_tls_"
)

func BuildTLSLabelKeyFromVariable(tlsVariable string) string {
	return tlsLabelPrefix + tlsVariable
}

func BuildTLSLabelKey(tlsKey string) string {
	return tlsLabelPrefix + tlsVariablePrefix + tlsKey
}

func BuildTLSKeyFromLabelKey(tlsLabelKey string) (string, bool) {
	return strings.CutPrefix(tlsLabelKey, tlsLabelPrefix+tlsVariablePrefix)
}

func IsTLSMatcherField(matcherField string) bool {
	return strings.HasPrefix(matcherField, tlsMatcherFieldPrefix)
}

func BuildTLSKeyFromMatcherField(matcherField string) (string, bool) {
	result, ok := strings.CutPrefix(matcherField, tlsMatcherFieldPrefix)
	if !ok {
		return result, ok
	}
	result, _ = strings.CutPrefix(result, tlsVariablePrefix)
	// Always return true because perforator_tls_ is an optional prefix for tls variable
	// and is left for backward compatibility.
	return result, true
}
