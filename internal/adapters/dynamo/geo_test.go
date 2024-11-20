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
			actual, err := newGeoHash(lat, lon)
			Expect(err).To(Not(HaveOccurred()))
			actualHash := actual.hash()
			actualTrimmed := actual.trimmed(level)

			Expect(actualHash).To(Equal(expectedHash))
			Expect(actualTrimmed).To(Equal(expectedTrimmed))
		})
	})
	When("level is larger than 30", func() {
		level := 31
		lon, lat := 8.78141, 49.64636
		expectedHash := uint64(5158837096744667913)
		expectedTrimmed := uint64(5158837096744667913)

		It("is hashed correctly and level is ignored", func() {
			actual, err := newGeoHash(lat, lon)
			Expect(err).To(Not(HaveOccurred()))
			actualHash := actual.hash()
			actualTrimmed := actual.trimmed(level)

			Expect(actualHash).To(Equal(expectedHash))
			Expect(actualTrimmed).To(Equal(expectedTrimmed))
		})
	})
	When("hash is coordinates are not valid", func() {
		lon, lat := 900.0, 12000.78

		It("error occurs", func() {
			_, err := newGeoHash(lat, lon)
			Expect(err).To(HaveOccurred())
		})
	})
})
