package custom_profiling_operation

import (
	"testing"

	"github.com/stretchr/testify/require"

	cpo_proto "github.com/yandex/perforator/perforator/proto/custom_profiling_operation"
)

func TestStates(t *testing.T) {
	terminalStates := TerminalStates()
	nonTerminalStates := NonTerminalStates()

	allStates := map[cpo_proto.OperationState]bool{}
	for _, state := range terminalStates {
		require.False(t, allStates[state], "State %s already exists", state.String())
		allStates[state] = true
	}
	for _, state := range nonTerminalStates {
		require.False(t, allStates[state], "State %s already exists", state.String())
		allStates[state] = true
	}

	for state := range stateOrder {
		delete(allStates, state)
	}

	require.Equal(t, 0, len(allStates), "Non terminal states + terminal states should cover all states from stateOrder")
}
