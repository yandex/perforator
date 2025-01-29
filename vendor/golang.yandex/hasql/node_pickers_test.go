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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRandom(t *testing.T) {
	nodes := []Node{
		NewNode("shimba", nil),
		NewNode("boomba", nil),
		NewNode("looken", nil),
	}

	rr := PickNodeRandom()

	pickedNodes := make(map[string]struct{})
	for i := 0; i < 100; i++ {
		pickedNodes[rr(nodes).Addr()] = struct{}{}
	}
	expectedNodes := map[string]struct{}{"boomba": {}, "looken": {}, "shimba": {}}

	assert.Equal(t, expectedNodes, pickedNodes)
}

func TestPickNodeRoundRobin(t *testing.T) {
	nodes := []Node{
		NewNode("shimba", nil),
		NewNode("boomba", nil),
		NewNode("looken", nil),
		NewNode("tooken", nil),
		NewNode("chicken", nil),
		NewNode("cooken", nil),
	}
	iterCount := len(nodes) * 3

	rr := PickNodeRoundRobin()

	var pickedNodes []string
	for i := 0; i < iterCount; i++ {
		pickedNodes = append(pickedNodes, rr(nodes).Addr())
	}

	expectedNodes := []string{
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
		"shimba", "boomba", "looken", "tooken", "chicken", "cooken",
	}
	assert.Equal(t, expectedNodes, pickedNodes)
}

func TestClosest(t *testing.T) {
	nodes := []Node{
		NewNode("shimba", nil),
		NewNode("boomba", nil),
		NewNode("looken", nil),
	}

	rr := PickNodeClosest()
	assert.Equal(t, nodes[0], rr(nodes))
}
