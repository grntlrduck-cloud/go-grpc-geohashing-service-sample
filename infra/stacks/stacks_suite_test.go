package stacks_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestStacks(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Stacks Suite")
}
