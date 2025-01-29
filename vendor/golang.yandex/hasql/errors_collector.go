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
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"
)

// CollectedErrors are errors collected when checking node statuses
type CollectedErrors struct {
	Errors []NodeError
}

func (e *CollectedErrors) Error() string {
	if len(e.Errors) == 1 {
		return e.Errors[0].Error()
	}

	errs := make([]string, len(e.Errors))
	for i, ne := range e.Errors {
		errs[i] = ne.Error()
	}
	/*
		I don't believe there exist 'best join separator' that fit all cases (cli output, JSON, .. etc),
		so we use newline as error.Join did it.
		In difficult cases (as suggested in https://github.com/yandex/go-hasql/pull/14),
		the user should be able to receive "raw" errors and format them as it suits him.
	*/
	return strings.Join(errs, "\n")
}

// NodeError is error that background goroutine got while check given node
type NodeError struct {
	Addr       string
	Err        error
	OccurredAt time.Time
}

func (e *NodeError) Error() string {
	// 'foo.db' node error occurred at '2009-11-10..': FATAL: terminating connection due to ...
	return fmt.Sprintf("%q node error occurred at %q: %s", e.Addr, e.OccurredAt, e.Err)
}

type errorsCollector struct {
	store map[string]NodeError
	mu    sync.Mutex
}

func newErrorsCollector() errorsCollector {
	return errorsCollector{store: make(map[string]NodeError)}
}

func (e *errorsCollector) Add(addr string, err error, occurredAt time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.store[addr] = NodeError{
		Addr:       addr,
		Err:        err,
		OccurredAt: occurredAt,
	}
}

func (e *errorsCollector) Remove(addr string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.store, addr)
}

func (e *errorsCollector) Err() error {
	e.mu.Lock()
	errList := make([]NodeError, 0, len(e.store))
	for _, nErr := range e.store {
		errList = append(errList, nErr)
	}
	e.mu.Unlock()

	if len(errList) == 0 {
		return nil
	}

	sort.Slice(errList, func(i, j int) bool {
		return errList[i].OccurredAt.Before(errList[j].OccurredAt)
	})
	return &CollectedErrors{Errors: errList}
}
