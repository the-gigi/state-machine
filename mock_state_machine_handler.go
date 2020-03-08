package state_machine

type mockStateMachineHandler struct {
	cannedTransitions []StateID
	current           int
	stateFuncMap      StateFuncMap
}

// Advances the canned transition index and returns the next state
func (m *mockStateMachineHandler) cannedTransition() StateID {
	if m.current < len(m.cannedTransitions) {
		m.current += 1
	}
	return m.cannedTransitions[m.current]
}

func newMockStateMachineHandler(cannedTransitions []StateID) *mockStateMachineHandler {
	m := &mockStateMachineHandler{
		cannedTransitions: cannedTransitions,
	}

	return m
}

func (m *mockStateMachineHandler) GetStateFuncMap(states []StateID) StateFuncMap {
	result := StateFuncMap{}
	for i := range states {
		result[states[i]] = m.cannedTransition
	}
	return result
}
