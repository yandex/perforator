package kubeletclient

import (
	"fmt"
	"net/http"
	"os"
)

const (
	KubeletPort = "10250"
	tokenPath   = "/var/run/secrets/kubernetes.io/serviceaccount/token"
)

type InnerClient interface {
	Do(req *http.Request) (*http.Response, error)
}

type Client struct {
	inner InnerClient
}

func New(ic InnerClient) *Client {
	return &Client{inner: ic}
}

func (c *Client) Do(req *http.Request) (*http.Response, error) {
	// Need to read it everytime, because it might change.
	// TODO: May be add some retry policy
	token, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, fmt.Errorf("couldn't read service account token, %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+string(token))

	return c.inner.Do(req)
}
