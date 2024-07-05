# go-grpc-geohashing-service-sample - poi-info-service

## The Plan

Prerequisites: Public HostedZone + VPC (in my case a mini VPC due to costs)

- Go
- gRPC
- Dynamo
- GitHub Actions
- AWS Infrastructure:
  - Cognito User Group for Auth
  - Public ALB + WAF
  - ECS Fargate Cluster
  - ECR Container Registry

## Data Set for PoIs

The dataset processed in this service was downloaded from
[kaggle](https://www.kaggle.com/datasets/mexwell/electric-vehicle-charging-in-germany)
Collected from https://opendata.rhein-kreis-neuss.de/ by the Federal Network
Agency of Germany The dataset is licensed under the
[ATTRIBUTION 4.0 INTERNATIONAL](https://creativecommons.org/licenses/by/4.0/)
license

The data is modified and processed as part of this sample application just for
demo purposes. The modification is minimal and adjust it to the simple model
defined in the API and adds gehoashing to enable querying the data efficiently.

## Setup

Before setting app the required protoc plugins and tools by running
`make configure` ensure to set up

- [protoc](https://grpc.io/docs/protoc-installation/)
- [make](https://www.gnu.org/software/make/)
- [buf](https://buf.build/docs/installation)
- [docker](https://docs.docker.com/engine/install/) &
  [docker-buildx](https://github.com/docker/buildx)
- [colima](https://github.com/abiosoft/colima) or
  [docker desktop](https://www.docker.com/products/docker-desktop/)
- [configure GOBIN or GOPATH](https://go.dev/wiki/SettingGOPATH)
- [aws cdk](https://docs.aws.amazon.com/cdk/v2/guide/getting_started.html) &
  [aws cli](https://docs.aws.amazon.com/cli/latest/userguide/getting-started-install.html)

### Colima and Testcontainers

If you have a Mac you might be using colima since docker desktop requires a
license especially in corporate ograniyations. Make sure to correclty configure
colima:

```bash
export TESTCONTAINERS_DOCKER_SOCKET_OVERRIDE=/var/run/docker.sock
export DOCKER_HOST="unix://${HOME}/.colima/docker.sock"
```

## Install Dependencies

```bash
make ci
```

## Testing

The tests include unit and integrations in a BDD manner. For the integration
tests testcontainers is used to easily automate the container lifetime during
test suites.

Run tests and generate reports

```bash
make test_report
```

Run linter, vulnerability scan, tests & reports, synthesize cdk stacks, and
build container on arm machines

```bash
make test_full_local_arm
```

on amd/x86_64 machines

```bash
make test_full_local_amd
```

### Use ginkgo to bootstrap test suites

to bootstrap a new test suite in a module run

```bash
cd path/to/dir
ginkgo bootstrap
```

Checkout ginkgo [documentation](https://onsi.github.io/ginkgo/) for more
details.

## Vulnerability Checks

run vulnerability check

```bash
make vuln_scan
```

## Other Useful commands

- `cdk deploy` deploy this stack to your default AWS account/region
- `cdk diff` compare deployed stack with current state
- `cdk synth` emits the synthesized CloudFormation template
- `go mod tidy` remove unused go modules
- `go mod download` install go modules
- `go get -u ./...` update all dependencies recursive
- `ginkgo bootstrap` boostrap ginkgo test suit into current dir

## Helpful Resources

- Google and gRPC gateway documentation
  - https://buf.build/grpc-ecosystem/grpc-gateway/docs/main:grpc.gateway.protoc_gen_openapiv2.options#grpc.gateway.protoc_gen_openapiv2.options
  - https://grpc-ecosystem.github.io/grpc-gateway/docs/mapping/customizing_openapi_output/
  - https://github.com/grpc-ecosystem/grpc-gateway
  - https://github.com/googleapis/googleapis/blob/master/google/api/http.proto
