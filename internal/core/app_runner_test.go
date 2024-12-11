package core_test

import (
	"context"
	"os"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/api/gen/v1/health"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/core"
	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/test"
)

// basic smoke/integration test
// can not be run parallel to other suits which start the full application
var _ = Describe("given application", Ordered, func() {
	ctx := context.Background()
	appCtx := context.Background()
	appCtxCancel, cancel := context.WithCancel(appCtx)
	var runner *core.ApplicationRunner
	var container *test.DynamoContainer

	BeforeAll(func() {
		err := os.Chdir("../..")
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
	})

	When("started", func() {
		It("is running as expected", func() {
			// run appin background
			isRunning := false
		Outer:
			for {
				select {
				case <-time.After(10 * time.Second):
					break Outer
				default:
					if runner.Running() {
						isRunning = true
						break Outer
					}
					continue
				}
			}
			Expect(isRunning).To(BeTrue())
		})

		It("application is healthy", func() {
			healthClient := test.NewHealthRPCClient(runner.BootConfig().Grpc.Server.Port)
			resp := healthClient.CheckHealth()
			Expect(resp.GetStatus()).To(Equal(health.HealthCheckResponse_SERVING_STATUS_SERVING))
		})
		It("application is is terminated on context cancel", func() {
			isRunning := true
			cancel()
		Outer:
			for {
				select {
				case <-time.After(10 * time.Second):
					break Outer
				default:
					if !runner.Running() {
						isRunning = false
						break Outer
					}
					continue
				}
			}
			Expect(isRunning).To(BeFalse())
		})
	})

	AfterAll(func() {
		cancel()
		container.Stop()
	})
})
