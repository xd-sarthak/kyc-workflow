package services

import "fmt"

// State represents a KYC submission state.
type State string

const (
	StateDraft             State = "draft"
	StateSubmitted         State = "submitted"
	StateUnderReview       State = "under_review"
	StateApproved          State = "approved"
	StateRejected          State = "rejected"
	StateMoreInfoRequested State = "more_info_requested"
)

// allowedTransitions is the single source of truth for all valid state transitions.
var allowedTransitions = map[State][]State{
	StateDraft:             {StateSubmitted},
	StateSubmitted:         {StateUnderReview},
	StateUnderReview:       {StateApproved, StateRejected, StateMoreInfoRequested},
	StateMoreInfoRequested: {StateSubmitted},
}

// Transition validates whether a state change from → to is allowed.
// This is the ONLY place a state change is authorized in the entire system.
// Returns nil if the transition is valid, or an error describing why it is not.
func Transition(from, to State) error {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return fmt.Errorf("invalid transition: no transitions defined from state %q", from)
	}
	for _, s := range allowed {
		if s == to {
			return nil
		}
	}
	return fmt.Errorf("invalid transition: %q → %q is not allowed", from, to)
}

// IsTerminalState returns true if the state is a terminal state (approved or rejected).
func IsTerminalState(s State) bool {
	return s == StateApproved || s == StateRejected
}

// IsValidState returns true if the given string is a recognized state.
func IsValidState(s string) bool {
	switch State(s) {
	case StateDraft, StateSubmitted, StateUnderReview, StateApproved, StateRejected, StateMoreInfoRequested:
		return true
	}
	return false
}
