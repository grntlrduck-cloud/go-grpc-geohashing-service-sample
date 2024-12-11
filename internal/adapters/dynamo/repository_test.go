package dynamo_test

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/segmentio/ksuid"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/adapters/dynamo"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/app"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/domain/poi"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/test"
)

const testDatapath = "../../../config/db/local/cpoi_dynamo_items_int_test.csv"

var _ = Describe("given db connected", Ordered, func() {
	ctx := context.Background()
	ctx, cancelFn := context.WithCancel(ctx)
	logger := app.NewDevLogger(
		&app.LoggingConfig{},
	)
	var repository poi.Repository
	var container test.DynamoContainer

	BeforeAll(func() {
		container = *test.NewDynamoContainer(ctx)
		dynamoClient, err := dynamo.NewClientWrapper(
			dynamo.WithContext(ctx),
			dynamo.WithRegion("marvel-universe-1"),
			dynamo.WithEndPointOverride(container.Host(), container.Port()),
		)
		Expect(err).To(Not(HaveOccurred()))
		Expect(dynamoClient).To(Not(BeNil()))
		repository, err = dynamo.NewPoIGeoRepository(
			logger,
			dynamo.WithCreateAndInitTable(true),
			dynamo.WithTableName("poi_table_name"),
			dynamo.WithDynamoClientWrapper(dynamoClient),
			dynamo.WithTestInitDataOverrid(testDatapath),
		)
		Expect(err).To(Not(HaveOccurred()))
		Expect(repository).To(Not(BeNil()))
	})

	When("location query received", func() {
		// GetByID
		It("get poi by id returns poi location as expected", func() {
			id := "2ofD9igSisfEtgC743gf3BnzO7L"
			kID, err := ksuid.Parse(id)
			Expect(err).To(Not(HaveOccurred()))
			poi, err := repository.GetByID(ctx, kID, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(poi.ID).To(Equal(kID))
		})

		It("get poi by id with id not in DB return LocationNotFound", func() {
			id := "2ofD9i2CSiA4Oa0tIYRJlJv399H"
			kID, err := ksuid.Parse(id)
			Expect(err).To(Not(HaveOccurred()))
			_, err = repository.GetByID(ctx, kID, logger)
			Expect(err).To((HaveOccurred()))
			Expect(err).To(Equal(poi.ErrLocationNotFound))
		})

		// Upsert
		It("upsert poi does not error as expected", func() {
			poi := poi.PoILocation{
				ID:               ksuid.New(),
				Location:         poi.Coordinates{Latitude: 49.5, Longitude: 8.0},
				LocationEntrance: poi.Coordinates{Latitude: 49.5, Longitude: 8.0},
				Address: poi.Address{
					Street:       "foo",
					StreetNumber: "bar",
					City:         "foobar",
					ZipCode:      "123456",
					CountryCode:  "DEU",
				},
				Features: []string{"hello", "dynamo"},
			}
			err := repository.Upsert(ctx, &poi, logger)
			Expect(err).To(Not(HaveOccurred()))

			actualSafedPoi, err := repository.GetByID(ctx, poi.ID, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(*actualSafedPoi).To(Equal(poi))
		})

		// GetByProxinity
		It("get location by proximity search returns locations as expected", func() {
			cntr := poi.Coordinates{Longitude: 9.147263, Latitude: 49.333418}
			radiusMeters := 50_000.0 // 50 km
			pois, err := repository.GetByProximity(ctx, cntr, radiusMeters, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(pois)).To(BeNumerically(">", 30))
		})

		It("get location by proximity search returns locations as expected", func() {
			cntr := poi.Coordinates{Longitude: 9.147263, Latitude: 49.333418}
			radiusMeters := 100_000.0 // 100 km
			pois, err := repository.GetByProximity(ctx, cntr, radiusMeters, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(pois)).To(BeNumerically(">", 70))
		})

		It("get location by proximity with invalid coordinates returns error", func() {
			cntr := poi.Coordinates{Longitude: 9000.147263, Latitude: 49000.333418}
			radiusMeters := 50_000.0 // 50 km
			_, err := repository.GetByProximity(ctx, cntr, radiusMeters, logger)
			Expect(err).To((HaveOccurred()))
			Expect(err).To(Equal(poi.ErrInvalidSearchCoordinates))
		})

		// GetByBbox
		It("get location by bbox search returns locations as expected", func() {
			sw := poi.Coordinates{Longitude: 8.494772, Latitude: 49.425026}
			ne := poi.Coordinates{Longitude: 10.040508, Latitude: 50.089540}
			pois, err := repository.GetByBbox(ctx, sw, ne, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(pois)).To(BeNumerically(">", 70))
		})

		It("get location by bbox search with invalid coordinates returns nerror", func() {
			sw := poi.Coordinates{Longitude: 80000.494772, Latitude: 49.425026}
			ne := poi.Coordinates{Longitude: 10000.040508, Latitude: 509999.089540}
			_, err := repository.GetByBbox(ctx, sw, ne, logger)
			Expect(err).To((HaveOccurred()))
			Expect(err).To(Equal(poi.ErrInvalidSearchCoordinates))
		})

		// GetByRoute
		It("get location by route search returns locations as expected", func() {
			route := []poi.Coordinates{
				{Longitude: 9.181946, Latitude: 48.796183},
				{Longitude: 8.611994, Latitude: 49.75371},
				{Longitude: 8.180723, Latitude: 49.558617},
				{Longitude: 8.740714, Latitude: 50.144288},
				{Longitude: 13.100310, Latitude: 52.551214},
			}
			pois, err := repository.GetByRoute(ctx, route, logger)
			Expect(err).To(Not(HaveOccurred()))
			Expect(len(pois)).To(BeNumerically(">", 10))
		})

		It("get location by route search with invalid coordinates returns error", func() {
			route := []poi.Coordinates{
				{Longitude: 9000.181946, Latitude: 48888.796183},
				{Longitude: 8.611994, Latitude: 49.75371},
				{Longitude: 8888.180723, Latitude: 4999.558617},
				{Longitude: 8.740714, Latitude: 50.144288},
				{Longitude: 1333.100310, Latitude: 522222.551214},
			}
			_, err := repository.GetByRoute(ctx, route, logger)
			Expect(err).To((HaveOccurred()))
			Expect(err).To(Equal(poi.ErrInvalidSearchCoordinates))
		})
	})

	AfterAll(func() {
		go func() {
			_ = logger.Sync()
		}()
		container.Stop()
		cancelFn()
	})
})
