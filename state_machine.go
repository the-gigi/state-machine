package state_machine

import (
	"errors"
	"fmt"
	"reflect"
	"runtime"
)

type State struct {
	Name string
}

// A State id is just an unsigned integer
type StateID int

// A map of state ids to bool. Convenient for membership tests
type StateSet map[StateID]bool

// The function type that runs when the state machine's Execute() method is called
//
// The function that can perform arbitrary processing and return a StateID.
// If the state function returned a different state than the current state
// then a state transition will occur (if valid)
type StateFunc func() StateID

func (f *StateFunc) String() string {
	return runtime.FuncForPC(reflect.ValueOf(f).Pointer()).Name()
}

// Maps a state id to the function that runs when entering that state
type StateFuncMap = map[StateID]StateFunc

// The StateMachine is initialized with the states and valid transitions.
// It starts in the initial state, enforces valid transitions
// until it reaches a final state (if any) and then it stays there.
type StateMachine struct {
	state StateID
	spec  *StateMachineSpec
}

type StateMachineSpec struct {
	InitialState            StateID
	FinalStates             StateSet
	StateFuncMap            StateFuncMap
	ValidTransitions        map[StateID]StateSet
	AllowExternalTransition bool
}

func (sms *StateMachineSpec) IsFinalState(state StateID) bool {
	return sms.FinalStates[state]
}

// NewStateMachine() takes a StateMachineSpec, verifies it
// and creates a new StateMachine using the spec
func NewStateMachine(spec *StateMachineSpec) (*StateMachine, error) {
	if spec == nil {
		return nil, errors.New("the StateMachine spec can't be empty")
	}

	// Make sure there is a handler function for each state
	for s, stateFunc := range spec.StateFuncMap {
		if stateFunc == nil {
			return nil, fmt.Errorf("missing function for state %d", s)
		}
	}

	// Make sure there the initial state is in the state map
	if spec.StateFuncMap[spec.InitialState] == nil {
		return nil, errors.New("the initial state is missing from the state map")
	}

	// Make sure all the final states are in the state map
	for k := range spec.FinalStates {
		if spec.StateFuncMap[k] == nil {
			return nil, fmt.Errorf("the final state %d is missing from the state map", k)
		}
	}

	// Make sure the initial state is not one of the final states
	if spec.IsFinalState(spec.InitialState) {
		return nil, fmt.Errorf("the initial state can't be a final state")
	}

	var reachableStates = StateSet{spec.InitialState: true}
	// Check the valid transitions
	for k, v := range spec.ValidTransitions {
		// Make sure there are no transitions from a final state to any state
		if spec.IsFinalState(k) {
			return nil, fmt.Errorf("can't transition from a final state %d", k)
		}

		// Make sure the source state is in the state map
		if spec.StateFuncMap[k] == nil {
			return nil, fmt.Errorf("source state %d is missing from state map", k)
		}

		// Make sure all the destination states are in the state map + keep track of reachable states
		for s := range v {
			if spec.StateFuncMap[s] == nil {
				return nil, fmt.Errorf("target state %d is missing from state map", s)
			}
			reachableStates[s] = true
		}
	}

	// Make sure all states are reachable
	for i := range spec.StateFuncMap {
		if !reachableStates[StateID(i)] {
			return nil, fmt.Errorf("state %d is unreachable", i)
		}
	}

	// Make sure all non-final states have transitions
	for s := range spec.StateFuncMap {
		// Skip final states
		if spec.FinalStates[s] {
			continue
		}

		targets := spec.ValidTransitions[s]
		if len(targets) == 0 {
			return nil, fmt.Errorf("there are no transitions from state %d", s)
		}
	}

	// Return a StateMachine instance with the spec, and set the `state` field to the initial state
	return &StateMachine{
		spec:  spec,
		state: spec.InitialState,
	}, nil
}

// transition() transitions the state machine to a new state and invoke its function
//
// If the transition is not allowed it will return an error
func (sm *StateMachine) transition(newState StateID) (state StateID, err error) {
	state = sm.state

	// Verify the new state is a valid transition from the current state
	if !sm.isValidTransition(newState) {
		err = fmt.Errorf("can't transition from state %d to state %d", sm.state, newState)
		return
	}

	// If transitioning to the same state just return (no op)
	if state == newState {
		return
	}

	// Execute the new state function and store its result as the state machine's state
	newFunc := sm.spec.StateFuncMap[newState]
	sm.state = newFunc()

	state = sm.state
	return
}

func (sm *StateMachine) isValidTransition(newState StateID) bool {
	return sm.spec.ValidTransitions[sm.state][newState]
}

// Transition() invokes the private transition() method
//
//  It is designed for state machines that need to be controlled externally
//  Normally, transition() is called only by the Execute() function in response
//  to a state function returning a new state.
//
// The state machine must be configured to allow external transition (disabled by default)
func (sm *StateMachine) Transition(newState StateID) (StateID, error) {
	if !sm.spec.AllowExternalTransition {
		return sm.state, errors.New("external transition is forbidden")
	}

	return sm.transition(newState)
}

// Execute() runs the current state function and transitions to the state it returned
//
// The return values are the result of the transition.
func (sm *StateMachine) Execute() (StateID, error) {
	stateFunc := sm.spec.StateFuncMap[sm.state]
	newState := stateFunc()
	return sm.transition(newState)

}
