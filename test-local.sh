#!/usr/bin/env bash
# TODO move to make!
set -eux

echo "Linting code..."
golangci-lint run ./...

echo "Running tests with reports..."
go run --mod=mod gotest.tools/gotestsum --junitfile unit-tests.xml -- -coverprofile=cover.out -covermode count ./...
go tool cover -html=cover.out -o coverage.html
go run --mod=mod github.com/boumenot/gocover-cobertura <cover.out > coverage.xml

echo "Checking for vulnerabilities..."
go run --mod=mod golang.org/x/vuln/cmd/govulncheck ./...

echo "Generating protobuf ..."
buf generate

echo "Syntheizing app..."
export AWS_REGION="eu-west-1"
export AWS_ACCOUNT="123456789123"
cdk synth >>/dev/null

echo "Generating infrastructure diagram..."
npx cdk-dia

echo "Done"
