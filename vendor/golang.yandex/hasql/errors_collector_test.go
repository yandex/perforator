/*
   Copyright 2020 YANDEX LLC

   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

       http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.
*/

package hasql

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestErrorsCollector(t *testing.T) {
	nodesCount := 10
	errCollector := newErrorsCollector()
	require.NoError(t, errCollector.Err())

	connErr := errors.New("node connection error")
	occurredAt := time.Now()

	var wg sync.WaitGroup
	wg.Add(nodesCount)
	for i := 1; i <= nodesCount; i++ {
		go func(i int) {
			defer wg.Done()
			errCollector.Add(
				fmt.Sprintf("node-%d", i),
				connErr,
				occurredAt,
			)
		}(i)
	}

	errCollectDone := make(chan struct{})
	go func() {
		for {
			select {
			case <-errCollectDone:
				return
			default:
				// there are no assertions here, because that logic expected to run with -race,
				// otherwise it doesn't test anything, just eat CPU.
				_ = errCollector.Err()
			}
		}
	}()

	wg.Wait()
	close(errCollectDone)

	err := errCollector.Err()
	for i := 1; i <= nodesCount; i++ {
		assert.ErrorContains(t, err, fmt.Sprintf("\"node-%d\" node error occurred at", i))
	}
	assert.ErrorContains(t, err, connErr.Error())

}
