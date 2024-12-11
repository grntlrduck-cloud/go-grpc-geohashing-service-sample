package rpc_test

import (
	"context"
	"net/http"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	poiv1 "github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/poi"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/core"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/test"
)

var _ = Describe("given location search request", Ordered, func() {
	ctx := context.Background()
	appCtx := context.Background()
	appCtxCancel, cancel := context.WithCancel(appCtx)
	var runner *core.ApplicationRunner
	var container *test.DynamoContainer
	var rpcTestClient *test.PoIRPCClient
	var restTestClient *test.PoIHTTPProxyClient
	BeforeAll(func() {
		err := os.Chdir("../../../")
		Expect(err).To(Not(HaveOccurred()))
		container = test.NewDynamoContainer(ctx)
		os.Setenv("DYNAMOLOCAL_HOST", container.Host())
		os.Setenv("DYNAMOLOCAL_PORT", container.Port())
		os.Setenv("BOOT_PROFILE_ACTIVE", "test")
		runner = core.NewApplicationRunner(core.WithApplicationContext(appCtxCancel))
		Expect(runner).To(Not(BeNil()))
		go func() {
			runner.Run()
		}()
	Outer:
		for {
			select {
			case <-time.After(10 * time.Second):
				break Outer
			default:
				if runner.Running() {
					break Outer
				}
				continue
			}
		}
		rpcTestClient = test.NewPoIRPCClient(runner.BootConfig().Grpc.Server.Port)
		restTestClient = test.NewPoIHTTPProxyClient(
			"localhost",
			runner.BootConfig().Grpc.Proxy.Port,
		)
	})

	When("service is serving", func() {
		// PoI RPC
		It("poi info search using grpc returns poi", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			actual, err := rpcTestClient.PoI(id, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			Expect(actual).To(Not(BeNil()))
			actualId := actual.Poi.Id
			Expect(actualId).To(Equal(id))
		})

		// testing auth interceptor works as expected is enough with one service endpoint
		It("poi info search without key results in unauthenticated", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			_, err := rpcTestClient.PoI(id, true, false, "")
			Expect(err).To(HaveOccurred())
			statusErr, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Code()).To(Equal(codes.Unauthenticated))
		})

		It("poi info search with invalid key results in unauthenticated", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			_, err := rpcTestClient.PoI(id, true, true, "NOT THE SECRET")
			Expect(err).To(HaveOccurred())
			statusErr, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(statusErr.Code()).To(Equal(codes.PermissionDenied))
		})

		It("poi info search using rest returns poi", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			resp := restTestClient.Info(id, true, true, "")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			actualId := resp.Ok.Poi.ID
			Expect(actualId).To(Equal(id))
		})

		// just check interceptor and header matcher work as expected, unautenticated case is obsolete
		It("poi htt info search with invalid key results in unauthenticated", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			resp := restTestClient.Info(id, true, true, "NOT THE SECRET")
			Expect(resp.StatusCode).To(Equal(http.StatusForbidden))
		})

		It("poi info search without correlationID return invalid arguments", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			_, err := rpcTestClient.PoI(id, false, true, "")
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search without id return invalid arguments", func() {
			id := "" // empty string will result in invalid arguments
			_, err := rpcTestClient.PoI(id, true, true, "")
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search with invalid id format return invalid arguments", func() {
			id := "ivnalid@~/n|1c6540a7-d184-4e03-bae8-1cfd11b0c69c"
			_, err := rpcTestClient.PoI(id, true, true, "")
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search with none existing id return not found", func() {
			id := "2ofD9feghS7QFTzMcUvEsaxGT9B" // id not in test data
			_, err := rpcTestClient.PoI(id, true, true, "")
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.NotFound))
		})

		// Bbox RPC
		It("poi rpc bbox search returns result correctly", func() {
			// large geographic area
			sw := &poiv1.Coordinate{Lon: 8.494772, Lat: 49.425026}
			ne := &poiv1.Coordinate{Lon: 10.040508, Lat: 50.089540}
			resp, err := rpcTestClient.Bbox(ne, sw, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 70))
		})

		It("poi rpc bbox search with smaller area returns result correctly", func() {
			// bit smaller area
			sw := &poiv1.Coordinate{Lon: 9.58462, Lat: 50.64389}
			ne := &poiv1.Coordinate{Lon: 8.441913, Lat: 49.884059}
			resp, err := rpcTestClient.Bbox(ne, sw, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 50))
		})

		It("poi http bbox search returns result correctly", func() {
			// large geographic area
			sw := test.CoordinatesHTTP{Lon: 8.494772, Lat: 49.425026}
			ne := test.CoordinatesHTTP{Lon: 10.040508, Lat: 50.089540}
			resp := restTestClient.Bbox(ne, sw, true, true, "")

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			items := resp.Ok.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 70))
		})

		It("poi rpc bbox search without correlationID results in invalid arguments", func() {
			// large geographic area
			sw := &poiv1.Coordinate{Lon: 8.494772, Lat: 49.425026}
			ne := &poiv1.Coordinate{Lon: 10.040508, Lat: 50.089540}
			_, err := rpcTestClient.Bbox(ne, sw, false, true, "") // send no correlationID
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc bbox search with invalid sw coordinate results in invalid arguments", func() {
			// invalid sw coordinate
			sw := &poiv1.Coordinate{Lon: 900.0, Lat: 1200.5}
			ne := &poiv1.Coordinate{Lon: 10.040508, Lat: 50.089540}
			_, err := rpcTestClient.Bbox(ne, sw, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc bbox search with nil sw coordinate results in invalid arguments", func() {
			// invalid sw coordinate
			var sw *poiv1.Coordinate = nil
			ne := &poiv1.Coordinate{Lon: 10.040508, Lat: 50.089540}
			_, err := rpcTestClient.Bbox(ne, sw, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc bbox search with nil ne coordinate results in invalid arguments", func() {
			// invalid ne coordinate
			sw := &poiv1.Coordinate{Lon: 8.494772, Lat: 49.425026}
			var ne *poiv1.Coordinate = nil
			_, err := rpcTestClient.Bbox(ne, sw, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		// Proximity RPC
		It("poi rpc proximity search in large are returns result correctly", func() {
			// large geographic area
			cntr := &poiv1.Coordinate{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 100_000.0 // 100km
			resp, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 70))
		})

		It("poi rpc proximity search in medium sized area returns result correctly", func() {
			// medium sized geographic area
			cntr := &poiv1.Coordinate{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 50_000.0 // 50km
			resp, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 30))
		})

		It("poi http proximity search in medium sized area returns result correctly", func() {
			// medium sized geographic area
			cntr := test.CoordinatesHTTP{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 50_000.0 // 50km
			resp := restTestClient.Prxoimity(cntr, radiusmeter, true, true, "")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			numPois := len(resp.Ok.Items)
			Expect(numPois).To(BeNumerically(">", 30))
		})

		It("poi rpc proximity search in medium sized area returns result correctly", func() {
			// smaller sized geographic area
			cntr := &poiv1.Coordinate{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 30_000.0 // 30km
			resp, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			numPois := len(items)
			Expect(numPois).To(BeNumerically(">", 3))
		})

		It("poi rpc proximity search with too large radius returns invalid arguments", func() {
			// gigantic search radius
			cntr := &poiv1.Coordinate{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 2_000_000.0 // 2000km
			_, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc proximity search without correlationID returns invalid arguments", func() {
			cntr := &poiv1.Coordinate{Lon: 9.147263, Lat: 49.333418}
			var radiusmeter float64 = 10_000.0 // 10km
			_, err := rpcTestClient.Proximity(cntr, radiusmeter, false, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc proximity search without center coordinate returns invalid arguments", func() {
			var cntr *poiv1.Coordinate = nil
			var radiusmeter float64 = 10_000.0 // 10km
			_, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi rpc proximity search with invalid coordinate returns invalid arguments", func() {
			cntr := &poiv1.Coordinate{Lon: 90000.147263, Lat: 49000.333418}
			var radiusmeter float64 = 10_000.0 // 10km
			_, err := rpcTestClient.Proximity(cntr, radiusmeter, true, true, "")
			Expect(err).To((HaveOccurred()))
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		// Route RPC
		It("poi rpc route search return result correctly", func() {
			// random route from Frankfurt area to Berlin area
			route := []*poiv1.Coordinate{
				{Lon: 9.181946, Lat: 48.796183},
				{Lon: 8.611994, Lat: 49.75371},
				{Lon: 8.180723, Lat: 49.558617},
				{Lon: 8.740714, Lat: 50.144288},
				{Lon: 13.100310, Lat: 52.551214},
			}
			resp, err := rpcTestClient.Route(route, true, true, "")
			Expect(err).To(Not(HaveOccurred()))
			items := resp.Items
			Expect(len(items)).To(BeNumerically(">", 10))
		})

		It("poi http route search return result correctly", func() {
			// random route from Frankfurt area to Berlin area
			route := []test.CoordinatesHTTP{
				{Lon: 9.181946, Lat: 48.796183},
				{Lon: 8.611994, Lat: 49.75371},
				{Lon: 8.180723, Lat: 49.558617},
				{Lon: 8.740714, Lat: 50.144288},
				{Lon: 13.100310, Lat: 52.551214},
			}
			resp := restTestClient.Route(route, true, true, "")
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			items := resp.Ok.Items
			Expect(len(items)).To(BeNumerically(">", 10))
		})

		It("poi rpc route search without correlationID returns invalid arguments", func() {
			// random route from Frankfurt area to Berlin area
			route := []*poiv1.Coordinate{
				{Lon: 9.181946, Lat: 48.796183},
				{Lon: 8.611994, Lat: 49.75371},
				{Lon: 8.180723, Lat: 49.558617},
				{Lon: 8.740714, Lat: 50.144288},
				{Lon: 13.100310, Lat: 52.551214},
			}
			_, err := rpcTestClient.Route(route, false, true, "") // dont send correlationID
			Expect(err).To((HaveOccurred()))
			errStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(errStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It(
			"poi rpc route search with only one coordinate in route returns invalid arguments",
			func() {
				route := []*poiv1.Coordinate{
					{Lon: 9.181946, Lat: 48.796183},
				}
				_, err := rpcTestClient.Route(route, true, true, "")
				Expect(err).To((HaveOccurred()))
				errStatus, ok := status.FromError(err)
				Expect(ok).To(BeTrue())
				Expect(errStatus.Code()).To(Equal(codes.InvalidArgument))
			},
		)

		It(
			"poi rpc route search with no route returns invalid arguments",
			func() {
				var route []*poiv1.Coordinate = nil
				_, err := rpcTestClient.Route(route, true, true, "")
				Expect(err).To((HaveOccurred()))
				errStatus, ok := status.FromError(err)
				Expect(ok).To(BeTrue())
				Expect(errStatus.Code()).To(Equal(codes.InvalidArgument))
			},
		)

		It("poi rpc route search invalid coordinates returns invalid arguments", func() {
			// random route from Frankfurt area to Berlin area
			route := []*poiv1.Coordinate{
				{Lon: 9.181946, Lat: 48.796183},
				{Lon: 8000.611994, Lat: 49000.75371},
				{Lon: 8.180723, Lat: 123449.558617},
				{Lon: 8.740714, Lat: 50.144288},
				{Lon: 13000.100310, Lat: 52.551214},
			}
			_, err := rpcTestClient.Route(route, true, true, "") // dont send correlationID
			Expect(err).To((HaveOccurred()))
			errStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(errStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		AfterAll(func() {
			cancel()
			container.Stop()
		})
	})
})
