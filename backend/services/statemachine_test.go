package services

import (
	"testing"
)

func TestTransition_ValidTransitions(t *testing.T) {
	tests := []struct {
		from State
		to   State
	}{
		{StateDraft, StateSubmitted},
		{StateSubmitted, StateUnderReview},
		{StateUnderReview, StateApproved},
		{StateUnderReview, StateRejected},
		{StateUnderReview, StateMoreInfoRequested},
		{StateMoreInfoRequested, StateSubmitted},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			if err := Transition(tt.from, tt.to); err != nil {
				t.Errorf("expected valid transition %s → %s, got error: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestTransition_IllegalTransitions(t *testing.T) {
	tests := []struct {
		from State
		to   State
	}{
		{StateApproved, StateDraft},
		{StateRejected, StateDraft},
		{StateDraft, StateApproved},
		{StateDraft, StateUnderReview},
		{StateSubmitted, StateApproved},
		{StateSubmitted, StateRejected},
		{StateSubmitted, StateDraft},
		{StateApproved, StateSubmitted},
		{StateRejected, StateSubmitted},
		{StateMoreInfoRequested, StateApproved},
		{StateMoreInfoRequested, StateDraft},
	}

	for _, tt := range tests {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			if err := Transition(tt.from, tt.to); err == nil {
				t.Errorf("expected error for illegal transition %s → %s, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestTransition_UnknownSourceState(t *testing.T) {
	err := Transition(State("nonexistent"), StateSubmitted)
	if err == nil {
		t.Error("expected error for unknown source state, got nil")
	}
}

func TestIsTerminalState(t *testing.T) {
	if !IsTerminalState(StateApproved) {
		t.Error("approved should be terminal")
	}
	if !IsTerminalState(StateRejected) {
		t.Error("rejected should be terminal")
	}
	if IsTerminalState(StateDraft) {
		t.Error("draft should not be terminal")
	}
	if IsTerminalState(StateSubmitted) {
		t.Error("submitted should not be terminal")
	}
}

func TestIsValidState(t *testing.T) {
	valids := []string{"draft", "submitted", "under_review", "approved", "rejected", "more_info_requested"}
	for _, s := range valids {
		if !IsValidState(s) {
			t.Errorf("expected %q to be valid", s)
		}
	}
	if IsValidState("garbage") {
		t.Error("expected 'garbage' to be invalid")
	}
}
