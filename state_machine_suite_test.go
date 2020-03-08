package state_machine

import (
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestStateMachine(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "StateMachine Suite")
}
