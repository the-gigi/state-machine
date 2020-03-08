package state_machine

import (
	"fmt"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

const (
	INIT StateID = iota
	CREATE
	RUN
	DONE
	FAIL
	NO_SUCH_STATE = 777
)

// getDefaultSpecAndMock() returns a proper state machine spec with valid transitions
func getDefaultSpec(m *mockStateMachineHandler) *StateMachineSpec {
	states := []StateID{INIT, CREATE, RUN, DONE, FAIL}
	spec := &StateMachineSpec{
		InitialState: INIT,
		FinalStates:  StateSet{DONE: true, FAIL: true},
		StateFuncMap: m.GetStateFuncMap(states),
		ValidTransitions: map[StateID]StateSet{
			INIT:   {CREATE: true},
			CREATE: {RUN: true, FAIL: true},
			RUN:    {RUN: true, DONE: true, FAIL: true},
		},
		AllowExternalTransition: true,
	}

	return spec
}

var _ = Describe("StateMachine Tests", func() {
	var (
		m    *mockStateMachineHandler
		spec *StateMachineSpec
	)
	BeforeSuite(func() {

	})
	AfterSuite(func() {

	})

	BeforeEach(func() {
		// default canned transitions from INIT to a DONE final state
		m = newMockStateMachineHandler([]StateID{INIT, CREATE, RUN, RUN, DONE})
		Ω(m).ShouldNot(BeNil())
		spec = getDefaultSpec(m)
		Ω(spec).ShouldNot(BeNil())
	})
	AfterEach(func() {
		m = nil
		spec = nil
	})

	Context("Successful state machine creation", func() {
		It("should create state machine successfully with default spec", func() {
			sm, err := NewStateMachine(spec)
			Ω(err).Should(BeNil())
			Ω(sm).ShouldNot(BeNil())
			Ω(sm.spec).Should(Equal(spec))
			Ω(sm.state).Should(Equal(spec.InitialState))
		})
	})

	Context("Failed state machine creation", func() {
		It("should fail when a handler function is missing for a state", func() {
			spec.StateFuncMap[CREATE] = nil
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("missing function for state %d", CREATE)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when the initial state is not in the state map", func() {
			delete(spec.StateFuncMap, INIT)
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := "the initial state is missing from the state map"
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when a final state is not in the state map", func() {
			delete(spec.StateFuncMap, DONE)
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("the final state %d is missing from the state map", DONE)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when the initial state is a final state", func() {
			spec.InitialState = FAIL
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := "the initial state can't be a final state"
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when there is a transition from a final state", func() {
			spec.ValidTransitions[FAIL] = StateSet{RUN: true}
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("can't transition from a final state %d", FAIL)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when the source state is not in the state map", func() {
			spec.ValidTransitions[NO_SUCH_STATE] = StateSet{CREATE: true}
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("source state %d is missing from state map", NO_SUCH_STATE)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when a target state is not in the state map", func() {
			spec.ValidTransitions[INIT][NO_SUCH_STATE] = true // add invalid transition INIT -> NO_SUCH_STATE
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("target state %d is missing from state map", NO_SUCH_STATE)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when a non-initial state is unreachable", func() {
			spec.ValidTransitions[INIT] = StateSet{} // remove INIT -> CREATE transition
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("state %d is unreachable", CREATE)
			Ω(err.Error()).Should(Equal(errString))
		})

		It("should fail when a non-final state has no transitions", func() {
			spec.ValidTransitions[RUN] = StateSet{}                                     // remove RUN -> DONE (now RUN transitions nowhere)
			spec.ValidTransitions[CREATE] = StateSet{RUN: true, FAIL: true, DONE: true} // ensure DONE is still connected
			_, err := NewStateMachine(spec)
			Ω(err).ShouldNot(BeNil())
			errString := fmt.Sprintf("there are no transitions from state %d", RUN)
			Ω(err.Error()).Should(Equal(errString))
		})
	})

	Context("State machine transitions (using both transition() and public Transition()", func() {
		type transitionFunc func(StateID) (StateID, error)
		var sm *StateMachine
		var err error
		var transitionFuncs = []transitionFunc{}
		BeforeEach(func() {
			sm, err = NewStateMachine(spec)
			Ω(err).Should(BeNil())
			transitionFuncs = []transitionFunc{sm.transition, sm.Transition}

		})

		It("should perform successfully every valid transition", func() {
			// Replace state funcs with a function that returns its own state (prevent extra transitions)
			for s := range sm.spec.StateFuncMap {
				var currState = s // closure state is necessary here
				sm.spec.StateFuncMap[s] = func() StateID {
					return currState
				}
			}

			// Perform the tests for both transition functions
			for i := range transitionFuncs {
				transition := transitionFuncs[i]
				// Iterate over all valid transition
				for s, targets := range sm.spec.ValidTransitions {
					// For each source state iterate over all the target states
					for t := range targets {
						// Set the current state to the source state
						sm.state = s

						// Transition to the current target state using the current transition func
						newState, err := transition(t)

						// Verify everything is as it should be :-)
						Ω(err).Should(BeNil())
						Ω(newState).Should(Equal(t))
						if sm.state != t {
							Ω(sm.state).Should(Equal(t))
						}
						Ω(sm.state).Should(Equal(t))
					}
				}
			}
		})

		It("should fail all invalid transitions", func() {
			// Replace state functions with a function that returns its own state (prevent extra transitions)
			for s := range sm.spec.StateFuncMap {
				var currState = s // closure state is necessary here
				sm.spec.StateFuncMap[s] = func() StateID {
					return currState
				}
			}

			type StateTransition struct {
				source StateID
				target StateID
			}
			// Prepare a bunch of  invalid transitions
			invalidTransitions := []StateTransition{
				{CREATE, INIT},
				{NO_SUCH_STATE, INIT},
				{FAIL, FAIL},
				{FAIL, DONE},
				{RUN, NO_SUCH_STATE},
			}

			// Perform the tests for both transition functions
			for i := range transitionFuncs {
				transition := transitionFuncs[i]
				// Try to transition from
				for i := range invalidTransitions {
					s := invalidTransitions[i].source
					t := invalidTransitions[i].target
					// Set the current state to the source state
					sm.state = s
					// Transition to the current target state using the current transition func
					_, err := transition(t)

					// Verify it failed
					Ω(err).ShouldNot(BeNil())
					errorMsg := fmt.Sprintf("can't transition from state %d to state %d", s, t)
					Ω(err.Error()).Should(Equal(errorMsg))

					// Verify the state machine is still in the source state
					Ω(sm.state).Should(Equal(invalidTransitions[i].source))
				}
			}
		})

		It("should fail valid transitions using Transition() when external transitions are disallowed ", func() {
			// Disallow external transitions
			sm.spec.AllowExternalTransition = false

			sm.state = RUN
			_, err := sm.Transition(DONE)
			Ω(err).ShouldNot(BeNil())
			Ω(err.Error()).Should(Equal("external transition is forbidden"))
			// Verify the state machine is still in the source state
			Ω(sm.state).Should(Equal(RUN))
		})

		It("should perform a transition to the same state as a no-op (state func should NOT be called)", func() {
			for i := range transitionFuncs {
				transition := transitionFuncs[i]
				stateFuncCalled := false
				sm.state = RUN

				// Replace the state func for RUN with a function that records being called and returns invalid state
				sm.spec.StateFuncMap[RUN] = func() StateID {
					stateFuncCalled = true // when called sets the stateFuncCalled variable defined above (closure)
					return NO_SUCH_STATE
				}

				// Transition from RUN -> RUN
				newState, err := transition(RUN)
				Ω(err).Should(BeNil())
				// The state should still be RUN
				Ω(sm.state).Should(Equal(RUN))
				Ω(newState).Should(Equal(RUN))

				// The RUN state func should NOT have been called
				Ω(stateFuncCalled).Should(BeFalse())
			}
		})

		It("should call the new state func when transitioning to a new state", func() {
			for i := range transitionFuncs {
				transition := transitionFuncs[i]

				stateFuncCalled := false
				sm.state = CREATE

				// Replace the state func for RUN with a function that records being called and returns DONE
				sm.spec.StateFuncMap[RUN] = func() StateID {
					stateFuncCalled = true // when called sets the stateFuncCalled variable defined above (closure)
					return DONE
				}

				// Transition from CREATE -> RUN
				newState, err := transition(RUN)
				Ω(err).Should(BeNil())
				// The state should be DONE now
				Ω(sm.state).Should(Equal(DONE))
				Ω(newState).Should(Equal(DONE))

				// The RUN state func should have been called
				Ω(stateFuncCalled).Should(BeTrue())
			}
		})
	})

	Context("State machine execution (using the Execute() method)", func() {

	})
})
