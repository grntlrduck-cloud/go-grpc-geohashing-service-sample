package dynamo_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestDynamo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Dynamo Suite")
}
