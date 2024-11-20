package app_test

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/internal/app"
)

var _ = Describe("init logger", func() {
	When("logging config is provided", func() {
		loggingConfig := &app.LoggingConfig{
			AppName:  "test",
			Level:    "test",
			Env:      "local",
			Host:     "localhost",
			Region:   "not-region",
			Account:  "123456",
			TeamName: "lonew-wanderer",
		}

		It("dev logger is initialized as expected and does not panic", func() {
			dev := app.NewDevLogger(loggingConfig)
			Expect(dev).To(Not(BeNil()))
		})

		It("prod logger is initialized as expected and does not panic", func() {
			prod := app.NewLogger(loggingConfig)
			Expect(prod).To(Not(BeNil()))
		})
	})

	When("logging config is nil", func() {
		It("dev logger is initialized as expected and does not panic", func() {
			dev := app.NewDevLogger(nil)
			Expect(dev).To(Not(BeNil()))
		})

		It("prod logger is initialized as expected and does not panic", func() {
			prod := app.NewLogger(nil)
			Expect(prod).To(Not(BeNil()))
		})
	})
})
