package kubelet

import (
	"fmt"
	"net/http"
	"os"
)

const (
	tokenPath = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

// kubeletTokenTransport is a custom RoundTripper that adds the Bearer token to each request to access kubelet api.
type kubeletTokenTransport struct {
	rt http.RoundTripper
}

// RoundTrip implements the RoundTripper interface.
func (t *kubeletTokenTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Need to read it everytime, because it might change.
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read service account token, %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+string(token))
	return t.rt.RoundTrip(req)
}
