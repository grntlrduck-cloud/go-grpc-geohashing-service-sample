package dynamo_test

import (
	"github.com/amazon-ion/ion-go/ion"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/segmentio/ksuid"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
)

var _ = Describe("Given charging location CPoIItem", func() {
	When("is valid item", func() {
		pk := ksuid.New()
		poiItem := dynamo.CPoIItem{
			Pk:                pk.String(),
			GeoIndexPk:        1234,
			GeoIndexSk:        12456789012,
			ID:                pk.String(),
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
				ID: pk,
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
			assertDomainToEqual(*actual, expectedDomain)
		})
	})

	When("item has invalid id", func() {
		poiItem := dynamo.CPoIItem{
			Pk:                "NOT A KSUID",
			GeoIndexPk:        1234,
			GeoIndexSk:        12456789012,
			ID:                "NOT A KSUID",
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

	When("domain is paresed to CPoIItem", func() {
		domain := poi.PoILocation{
			ID: ksuid.New(),
			Address: poi.Address{
				Street:       "IDK",
				StreetNumber: "13b",
				ZipCode:      "123456",
				City:         "No City",
				CountryCode:  "USA",
			},
			Location: poi.Coordinates{
				Latitude:  14.5,
				Longitude: 15.4,
			},
			LocationEntrance: poi.Coordinates{
				Latitude:  14.5,
				Longitude: 15.4,
			},
			Features: []string{"blub"},
		}
		It("is valid location", func() {
			expected := dynamo.CPoIItem{
				ID:                domain.ID.String(),
				Pk:                domain.ID.String(),
				Longitude:         domain.Location.Longitude,
				Latitude:          domain.Location.Latitude,
				EntranceLongitude: domain.LocationEntrance.Longitude,
				EntranceLatitude:  domain.LocationEntrance.Latitude,
				GeoIndexPk:        1231351868039364608,
				GeoIndexSk:        1231347589921125375,
				Street:            domain.Address.Street,
				StreetNumber:      domain.Address.StreetNumber,
				ZipCode:           domain.Address.ZipCode,
				City:              domain.Address.City,
				CountryCode:       domain.Address.CountryCode,
				Features:          domain.Features,
			}
			actual, err := dynamo.NewItemFromDomain(&domain)
			Expect(err).To(Not(HaveOccurred()))
			assertItemToEqualWithoutId(*actual, expected)
			Expect(actual.ID).To(Equal(expected.ID))
			Expect(actual.Pk).To(Equal(expected.Pk))
		})
	})

	When("csv entries are paresed to CPoIItem", func() {
		csvEntries := []*dynamo.ChargingCSVEntry{
			{
				ChargingType:         "Schnellladeeinreichtung",
				Power:                250.0,
				NumberOfChargePoints: 3,
				PlugType1:            "AC CCS COMBO1",
				PlugType2:            "DC CSS COMBO2",
				PlugType3:            "DC CSS COMBO2",
				City:                 "Munich",
				ZipCode:              "123456",
				Street:               "Strasse",
				StreetNumber:         "12a",
				Longitude:            15.4,
				Latitude:             14.5,
			},
		}
		It("mapped correctly", func() {
			expected := []*dynamo.CPoIItem{
				{
					ID:                ksuid.Nil.String(),
					Pk:                ksuid.Nil.String(),
					Longitude:         15.4,
					Latitude:          14.5,
					EntranceLongitude: 15.4,
					EntranceLatitude:  14.5,
					GeoIndexPk:        1231351868039364608,
					GeoIndexSk:        1231347589921125375,
					Street:            "Strasse",
					StreetNumber:      "12a",
					ZipCode:           "123456",
					City:              "Munich",
					CountryCode:       "DEU",
					Features: []string{
						"3_CHARGEPOINTS",
						"250_KW_CHARGING",
						"AC_CHARGING",
						"DC_CHARGING",
					},
				},
			}

			actual := dynamo.EntriesToDynamo(csvEntries)

			Expect(len(actual)).To(Equal(len(expected)))
			assertItemToEqualWithoutId(*expected[0], *actual[0])
		})
	})

	When("mapped to DynamoDB Ion item", func() {
		pk := ksuid.New()
		poiItem := dynamo.CPoIItem{
			Pk:                pk.String(),
			GeoIndexPk:        1234,
			GeoIndexSk:        123456789012,
			ID:                pk.String(),
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
		expectedIonCp := dynamo.CPoIIonItem{
			Pk:                pk.String(),
			GeoIndexPk:        *ion.NewDecimalInt(1234),
			GeoIndexSk:        *ion.NewDecimalInt(123456789012),
			ID:                pk.String(),
			Street:            "Some Street",
			StreetNumber:      "12b",
			ZipCode:           "123456",
			City:              "New York",
			CountryCode:       "DEU",
			Features:          []string{"1", "2"},
			Longitude:         *ion.MustParseDecimal("12.5"),
			Latitude:          *ion.MustParseDecimal("15.5"),
			EntranceLongitude: *ion.MustParseDecimal("12.53"),
			EntranceLatitude:  *ion.MustParseDecimal("15.52"),
		}
		expectedIonItem := dynamo.IonItem{expectedIonCp}

		It("is correct", func() {
			actual := poiItem.IonItem()
			Expect(*actual).To(Equal(expectedIonItem))
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
	Expect(actual.ID).To(Equal(expected.ID))
}

func assertItemToEqualWithoutId(actual, expected dynamo.CPoIItem) {
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
	Expect(actual.StreetNumber).To(Equal(expected.StreetNumber))
	Expect(actual.City).To(Equal(expected.City))
	Expect(actual.ZipCode).To(Equal(expected.ZipCode))
	Expect(actual.GeoIndexPk).To(Equal(expected.GeoIndexPk))
	Expect(actual.GeoIndexSk).To(Equal(expected.GeoIndexSk))
	Expect(actual.CountryCode).To(Equal(expected.CountryCode))
	Expect(actual.Features).To(Equal(expected.Features))
}
