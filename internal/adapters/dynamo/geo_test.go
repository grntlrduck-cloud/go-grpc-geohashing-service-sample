package dynamo

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("given coordinates", func() {
	When("hash is calculated", func() {
		level := 8
		lon, lat := 8.78141, 49.64636
		expectedHash := uint64(5158837096744667913)
		expectedTrimmed := uint64(5158820596594769920)

		It("is hashed correctly", func() {
			actual := newGeoHash(lat, lon)
			actualHash := actual.hash()
			actualTrimmed := actual.trimmed(level)

			Expect(actualHash).To(Equal(expectedHash))
			Expect(actualTrimmed).To(Equal(expectedTrimmed))
		})
	})
})
