#!/usr/bin/env bash

set -eux

echo "Linting code..."
golangci-lint run ./...

echo "Running tests with reports..."
gotestsum --junitfile unit-tests.xml -- -coverprofile=cover.out -covermode count ./...
go tool cover -html=cover.out -o coverage.html
gocover-cobertura < cover.out > coverage.xml

echo "Checking for vulnerabilities..."
govulncheck ./...

export AWS_REGION="eu-west-1"
export AWS_ACCOUNT="123456789123"

echo "Syntheizing app..."
cdk synth >> /dev/null

echo "Generating infrastructure diagram..."
npx cdk-dia

echo "Done"