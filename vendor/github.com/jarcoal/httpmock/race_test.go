package httpmock_test

import (
	"net/http"
	"sync"
	"testing"

	. "github.com/jarcoal/httpmock"
)

func TestActivateNonDefaultRace(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(10)
	for i := 0; i < 10; i++ {
		go func() {
			defer wg.Done()
			ActivateNonDefault(&http.Client{})
		}()
	}
	wg.Wait()
}
