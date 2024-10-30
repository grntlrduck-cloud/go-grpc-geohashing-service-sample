package app_test

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/grntlrduck-cloud/go-grpc-geohasing-service-sample/app"
)

// run in serial to avoid flakieness right away
var _ = Describe("Given application run as in container", Serial, func() {
	// application expects the configs available under ./config
	// but the test suit sets current working dir to PROJECT_ROOT/app, hence we need to walk up once
	err := os.Chdir("..")
	Expect(err).To(Not(HaveOccurred()))

	When("boot profile is prod", func() {
		It("boot config is merged with default", func() {
			os.Setenv("BOOT_PROFILE_ACTIVE", "prod")
			expectedSslEnabled := true
			bootConfig, err := app.LoadBootConfig()
			Expect(err).To(Not(HaveOccurred()))
			Expect(bootConfig.Grpc.Ssl.Enabled).To(Equal(expectedSslEnabled))
		})
	})

	When("boot profile is not set", func() {
		It("boot config is loaded without error", func() {
			os.Setenv("BOOT_PROFILE_ACTIVE", "")
			expectedSslEnabled := false
			bootConfig, err := app.LoadBootConfig()
			Expect(err).To(Not(HaveOccurred()))
			Expect(bootConfig.Grpc.Ssl.Enabled).To(Equal(expectedSslEnabled))
		})
		It("environment variables are expanded", func() {
			os.Setenv("BOOT_PROFILE_ACTIVE", "")

			expectedTableName := "PoITable"
			os.Setenv("POI_TABLE_NAME", expectedTableName)

			expectedAppName := "grpc-chagring-location-service"
			os.Setenv("APP_NAME", expectedAppName)

			expectedAccountId := "123456789012"
			os.Setenv("ACCOUNT_ID", expectedAccountId)

			bootConfig, err := app.LoadBootConfig()

			Expect(err).To(Not(HaveOccurred()))
			Expect(bootConfig.Aws.DynamoDb.PoiTableName).To(Equal(expectedTableName))
			Expect(bootConfig.Aws.Config.Account).To(Equal(expectedAccountId))
			Expect(bootConfig.App.Name).To(Equal(expectedAppName))
		})
	})

	When("boot profile is foo", func() {
		It("fails to load config and error is returned", func() {
			os.Setenv("BOOT_PROFILE_ACTIVE", "foo")
			_, err := app.LoadBootConfig()
			Expect(err).To(HaveOccurred())
		})
	})
})
