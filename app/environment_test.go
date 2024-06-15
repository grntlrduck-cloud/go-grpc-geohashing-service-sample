package app_test

import (
	"os"

	"github.com/grntlduck-cloud/go-grpc-geohasing-service-sample/app"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Test application environment load", func() {
	When("table name is set", func() {
		tableName := "some-table"
		os.Setenv("DYNAMO_TABLE_NAME", tableName)
		
		It("does not panic", func() {
			appEnv := app.NewServiceEnv()
			Expect(appEnv).Should(Not(BeNil()))
			Expect(appEnv.TableName).Should(Equal(tableName))
		})
	
	})
})
