package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransition(t *testing.T) {
	tests := []struct {
		name    string
		from    State
		to      State
		wantErr bool
	}{
		// Legal transitions
		{"draft→submitted", StateDraft, StateSubmitted, false},
		{"submitted→under_review", StateSubmitted, StateUnderReview, false},
		{"under_review→approved", StateUnderReview, StateApproved, false},
		{"under_review→rejected", StateUnderReview, StateRejected, false},
		{"under_review→more_info_requested", StateUnderReview, StateMoreInfoRequested, false},
		{"more_info_requested→submitted", StateMoreInfoRequested, StateSubmitted, false},

		// Illegal transitions
		{"approved→draft", StateApproved, StateDraft, true},
		{"approved→submitted", StateApproved, StateSubmitted, true},
		{"approved→under_review", StateApproved, StateUnderReview, true},
		{"rejected→submitted", StateRejected, StateSubmitted, true},
		{"rejected→approved", StateRejected, StateApproved, true},
		{"submitted→approved_skip_review", StateSubmitted, StateApproved, true},
		{"draft→approved", StateDraft, StateApproved, true},
		{"draft→under_review", StateDraft, StateUnderReview, true},
		{"under_review→draft", StateUnderReview, StateDraft, true},
		{"under_review→submitted", StateUnderReview, StateSubmitted, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := Transition(tt.from, tt.to)
			if tt.wantErr {
				require.Error(t, err)
				// All illegal transition errors contain the from state.
				assert.Contains(t, err.Error(), string(tt.from))
				// For non-terminal from states, the error also contains the to state
				// (message: "X" → "Y" is not allowed).
				// For terminal states (approved, rejected), the error says
				// "no transitions defined from state X" — only the from state.
				if tt.from != StateApproved && tt.from != StateRejected {
					assert.Contains(t, err.Error(), string(tt.to))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTransition_UnknownSourceState(t *testing.T) {
	err := Transition(State("nonexistent"), StateSubmitted)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "nonexistent")
}

func TestIsTerminalState(t *testing.T) {
	assert.True(t, IsTerminalState(StateApproved))
	assert.True(t, IsTerminalState(StateRejected))
	assert.False(t, IsTerminalState(StateDraft))
	assert.False(t, IsTerminalState(StateSubmitted))
	assert.False(t, IsTerminalState(StateUnderReview))
	assert.False(t, IsTerminalState(StateMoreInfoRequested))
}

func TestIsValidState(t *testing.T) {
	valids := []string{"draft", "submitted", "under_review", "approved", "rejected", "more_info_requested"}
	for _, s := range valids {
		assert.True(t, IsValidState(s), "expected %q to be valid", s)
	}
	assert.False(t, IsValidState("garbage"))
	assert.False(t, IsValidState(""))
	assert.False(t, IsValidState("APPROVED")) // case sensitive
}
