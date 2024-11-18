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

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/core"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/test"
)

var _ = Describe("given location search request", Ordered, func() {
	ctx := context.Background()
	appCtx := context.Background()
	appCtxCancel, cancel := context.WithCancel(appCtx)
	var runner *core.ApplicationRunner
	var container *test.DynamoContainer
	var rpcTestClient *test.PoIRpcClient
	var restTestClient *test.PoIHttpProxyClient
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
		rpcTestClient = test.NewPoIRpcClient(runner.BootConfig().Grpc.Server.Port)
		restTestClient = test.NewPoIHttpProxyClient(
			"localhost",
			runner.BootConfig().Grpc.Proxy.Port,
		)
	})

	When("service is serving", func() {
		It("poi info search using grpc returns poi", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			actual, err := rpcTestClient.PoI(id, true)
			actualId := actual.Poi.Id
			Expect(err).To(Not(HaveOccurred()))
			Expect(actual).To(Not(BeNil()))
			Expect(actualId).To(Equal(id))
		})

		It("poi info search using rest returns poi", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			resp := restTestClient.Info(id, true)
			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			actualId := resp.Ok.Poi.Id
			Expect(actualId).To(Equal(id))
		})

		It("poi info search without correlationId return invalid arguments", func() {
			id := "2ofD9hciu5kGIGdGXjPuJy3tUvH" // id from test data csv
			_, err := rpcTestClient.PoI(id, false)
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search without id return invalid arguments", func() {
			id := "" // empty string will result in invalid arguments
			_, err := rpcTestClient.PoI(id, true)
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search with invalid id format return invalid arguments", func() {
			id := "ivnalid@~/n|1c6540a7-d184-4e03-bae8-1cfd11b0c69c"
			_, err := rpcTestClient.PoI(id, true)
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.InvalidArgument))
		})

		It("poi info search with none existing id return not found", func() {
			id := "2ofD9feghS7QFTzMcUvEsaxGT9B" // id not in test data
			_, err := rpcTestClient.PoI(id, true)
			Expect(err).To(HaveOccurred())
			actualStatus, ok := status.FromError(err)
			Expect(ok).To(BeTrue())
			Expect(actualStatus.Code()).To(Equal(codes.NotFound))
		})
	})

	AfterAll(func() {
		cancel()
		container.Stop()
	})
})
