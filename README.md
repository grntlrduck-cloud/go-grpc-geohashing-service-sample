# go-grpc-geohashing-service-sample

## The Plan

Prerequisites: Public HostedZone + VPC (in my case a mini VPC due to costs)

* Go
* gRPC
* PostGis or Dynamo
* GitHub Actions
* AWS Infrastructure:
    * Cognito User Group for Auth
    * Public ALB + WAF
    * ECS Fargate Cluster
    * ECR Container Registry

## Setup
* Install go modules required for local development
* Install gotestsum `go install gotest.tools/gotestsum@latest` (for test reports in CI tools)
* Install gocover-cobertura `go install github.com/boumenot/gocover-cobertura@latest` (for test reports conversion in CI tools)
* Install golangci-lint `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` 
* Install or update ginkgo `go install github.com/onsi/ginkgo/v2/ginkgo`
* Install or update vuln checker `go install golang.org/x/vuln/cmd/govulncheck@latest`
* Install proto compiler `go install google.golang.org/protobuf/cmd/protoc-gen-go@latest`

## Generate code from protobuf

The booking event lambda receives a json message with a base64 encoded bytestring, the payload, which is parsed to the bookingpb.Event.
The code is generated with protoc and the go compiler plugin as follows:
```bash 
 protoc --go_out=paths=source_relative:. lambda/proto/*proto
```

## Testing

The tests include unit and integrations in a BDD manner.
For the integration tests testcontainers is used to easily automate the container lifetime during test suites.

To execute all tests and reports simply run the test script to execute linting and test execution with reportsTo run
```bash
./test-local.sh
```

run unit tests with junit test report and coverage.html report

```bash
gotestsum --junitfile unit-tests.xml -- -coverprofile=cover.out ./... go tool cover -html=cover.out -o coverage.html
```

run linter
````bash
golangci-lint run ./...
````

### Use ginkgo to bootstrap test suites

make sure you have ginkgo installed, if not install as follows: 
```bash
go install github.com/onsi/ginkgo/v2/ginkgo
```

to bootstrap a new test suite in a module run 
```bash
cd path/to/dir
ginkgo bootstrap
```

Checkout ginkgo [documentation](https://onsi.github.io/ginkgo/) for more details.

## Colima and Testcontainers

If you have a Mac you might be using colima since docker desktop requires a license.
Make sure to correclty configure colima:

```
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST="unix://${HOME}/.colima/docker.sock"
```


## Vulnerability Checks

install vulnerability checker 
```bash 
go install golang.org/x/vuln/cmd/govulncheck@latest
```

run vulnerability check

````bash 
govulncheck ./...
````

## Useful commands

* `cdk deploy`          deploy this stack to your default AWS account/region
* `cdk diff`            compare deployed stack with current state
* `cdk synth`           emits the synthesized CloudFormation template
* `go mod tidy`         remove unused go modules
* `go mod download`     install go modules
* `go get -u ./...`     update all dependencies recursive 
* `ginkgo bootstrap`    boostrap ginkgo test suit into current dir