# installs required binaries for linting and protobuf generation for local depvelopment as well as a pre commit hook to lint all files before committing
configure:
	@echo "Ensure GOBIN is added to path, buf, aws-cdk, docker & docker-buildx, and protoc is installed as docuemented in README. The configure script does not set up the aforementioned tools."
	@echo "Installing protobuf dependencies to GOBIN..."
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway@latest
	@go install github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2@latest
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
	@go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest	
	@echo "Installing golangci-lint to GOBIN..."
	@go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing ginkgo to GOBIN..."
	@go install github.com/onsi/ginkgo/v2/ginkgo@latest
	@echo "Installing dependencies from go.mod..."
	@go mod download
	@echo "Installing pre-commit hook ..."
	@cp pre-commit.sh .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Done."

ci:
	go mod download && go mod verify

lint:
	golangci-lint run ./...

synth_local:
	@echo "Syntheizing app with region=eu-west-1 and mock account..."
	AWS_ACCOUNT=123456789012 AWS_REGION=eu-west-1 cdk synth

synth_ci:
	cdk synth >>/dev/null

dia:
	npx cdk-dia

test_report:
	go run --mod=mod gotest.tools/gotestsum --junitfile unit-tests.xml  -- -coverprofile=cover.out -covermode count -p 1 ./...
	grep -v -E -f .covignore cover.out > coverage.filtered.out
	mv coverage.filtered.out cover.out
	go tool cover -html=cover.out -o coverage.html
	go run --mod=mod github.com/boumenot/gocover-cobertura <cover.out > coverage.xml

update_deps:
	go get -u ./...
	go mod tidy

vuln_scan:
	go run --mod=mod golang.org/x/vuln/cmd/govulncheck ./...

build_amd:
	docker buildx build --platform linux/amd64 -t grntlerduck/poi-info-service:$(shell git rev-parse --short HEAD) .

build_arm:
	docker buildx build --platform linux/arm64 -t grntlerduck/poi-info-service:$(shell git rev-parse --short HEAD) .

test_full_local_amd: lint vuln_scan test_report synth_local build_amd

test_full_local_arm: lint vuln_scan test_report synth_local build_arm

run_build_container:
	docker build -t go-grpc-geo:local .
	docker run -p 443:443 -p 8443:8443 go-grpc-geo:local

compose_local:
	docker compose up --build --remove-orphans
