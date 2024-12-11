package rpc

const (
	apiKeyHeader      = "X-Api-Key"               //nolint:gosec
	gAPIKeyMetadata   = "Grpc-Metadata-X-Api-Key" //nolint:gosec
	correlationHeader = "X-Correlation-Id"
	gCorrelationMD    = "Grpc-Metadata-X-Correlation-Id"
)
