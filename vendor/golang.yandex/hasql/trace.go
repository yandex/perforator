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

// Tracer is a set of hooks to run at various stages of background nodes status update.
// Any particular hook may be nil. Functions may be called concurrently from different goroutines.
type Tracer struct {
	// UpdateNodes is called when before updating nodes status.
	UpdateNodes func()
	// UpdatedNodes is called after all nodes are updated. The nodes is a list of currently alive nodes.
	UpdatedNodes func(nodes AliveNodes)
	// NodeDead is called when it is determined that specified node is dead.
	NodeDead func(node Node, err error)
	// NodeAlive is called when it is determined that specified node is alive.
	NodeAlive func(node Node)
	// NotifiedWaiters is called when all callers of 'WaitFor*' functions have been notified.
	NotifiedWaiters func()
}
