package dynamo_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/segmentio/ksuid"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/domain/poi"
)

var _ = Describe("Given charging location CPoIItem", func() {
	When("is valid item", func() {
		pk := ksuid.New()
		poiItem := dynamo.CPoIItem{
			Pk:                pk.String(),
			GeoIndexPk:        1234,
			GeoIndexSk:        12456789012,
			Id:                pk.String(),
			Street:            "Some Street",
			StreetNumber:      "12b",
			ZipCode:           "123456",
			City:              "New York",
			CountryCode:       "DEU",
			Features:          []string{"1", "2"},
			Longitude:         12.5,
			Latitude:          15.5,
			EntranceLongitude: 12.53,
			EntranceLatitude:  15.52,
		}
		It("does not return an error when parsed to Domain", func() {
			expectedDomain := poi.PoILocation{
				Id: pk,
				Location: poi.Coordinates{
					Latitude:  15.5,
					Longitude: 12.5,
				},
				Address: poi.Address{
					Street:       "Some Street",
					StreetNumber: "12b",
					ZipCode:      "123456",
					City:         "New York",
					CountryCode:  "DEU",
				},
				LocationEntrance: poi.Coordinates{
					Latitude:  15.52,
					Longitude: 12.53,
				},
				Features: []string{"1", "2"},
			}
			actual, err := poiItem.Domain()
			Expect(err).To(Not(HaveOccurred()))
			assertDomainToEqual(actual, expectedDomain)
		})
		It("is parseable to Domain and back", func() {
			domain, err := poiItem.Domain()
			Expect(err).To(Not(HaveOccurred()))
			actual := dynamo.NewItemFromDomain(domain)
			assertItemToEqual(actual, poiItem)
		})
	})

	When("item has invalid id", func() {
		poiItem := dynamo.CPoIItem{
			Pk:                "NOT A KSUID",
			GeoIndexPk:        1234,
			GeoIndexSk:        12456789012,
			Id:                "NOT A KSUID",
			Street:            "Some Street",
			StreetNumber:      "12b",
			ZipCode:           "123456",
			City:              "New York",
			CountryCode:       "DEU",
			Features:          []string{"1", "2"},
			Longitude:         12.5,
			Latitude:          15.5,
			EntranceLongitude: 12.53,
			EntranceLatitude:  15.52,
		}
		It("does return error", func() {
			_, err := poiItem.Domain()
			Expect(err).To(HaveOccurred())
		})
	})
})

func assertDomainToEqual(actual, expected poi.PoILocation) {
	Expect(actual.LocationEntrance.Latitude).Should(
		BeNumerically("~", expected.LocationEntrance.Latitude),
	)
	Expect(actual.LocationEntrance.Longitude).Should(
		BeNumerically("~", expected.LocationEntrance.Longitude),
	)
	Expect(actual.Location.Latitude).Should(
		BeNumerically("~", expected.Location.Latitude),
	)
	Expect(actual.Location.Longitude).Should(
		BeNumerically("~", expected.Location.Longitude),
	)
	Expect(actual.Address).To(Equal(expected.Address))
	Expect(actual.Features).To(Equal(expected.Features))
	Expect(actual.Id).To(Equal(expected.Id))
}

func assertItemToEqual(actual, expected dynamo.CPoIItem) {
	Expect(actual.EntranceLatitude).Should(
		BeNumerically("~", expected.EntranceLatitude),
	)
	Expect(actual.EntranceLongitude).Should(
		BeNumerically("~", expected.EntranceLongitude),
	)
	Expect(actual.Latitude).Should(
		BeNumerically("~", expected.Latitude),
	)
	Expect(actual.Longitude).Should(
		BeNumerically("~", expected.Longitude),
	)
	Expect(actual.Street).To(Equal(expected.Street))
	Expect(actual.City).To(Equal(expected.City))
	Expect(actual.StreetNumber).To(Equal(expected.StreetNumber))
	Expect(actual.ZipCode).To(Equal(expected.ZipCode))
	Expect(actual.CountryCode).To(Equal(expected.CountryCode))
	Expect(actual.Features).To(Equal(expected.Features))
	Expect(actual.Id).To(Equal(expected.Id))
}
