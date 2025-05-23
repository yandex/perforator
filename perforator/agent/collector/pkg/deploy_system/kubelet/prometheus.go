package kubelet

import "regexp"

var invalidLabelCharRE = regexp.MustCompile(`[^a-zA-Z0-9_]`)

// This ensures that Kubernetes pod label key conform to Prometheus label naming requirements and ads "pod_" prefix
// For example app.kubernetes.io/perforator becomes pod_app_kubernetes_io_name
// TODO: import https://github.com/prometheus/prometheus/blob/dbf5d01a62249eddcd202303069f6cf7dd3c4a73/util/strutil/strconv.go#L45
func SanitizeLabelName(name string) string {
	return "pod_" + invalidLabelCharRE.ReplaceAllString(name, "_")
}
