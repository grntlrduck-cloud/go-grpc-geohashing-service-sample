package rpc

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

func CorrelationIdMatcher(key string) (string, bool) {
	switch key {
	case "X-Correlation-Id":
		return key, true
	default:
		return key, false
	}
}

func GetCorrelationId(ctx context.Context) (*uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, errors.New("failed to extract metadata in service")
	}
	match := md.Get("X-Correlation-Id")
	if len(match) > 0 {
		id, err := uuid.Parse(match[0])
		if err != nil {
			return nil, fmt.Errorf("failed to parse uuid form correlation header: %w", err)
		}
		return &id, nil
	}
	return nil, errors.New("correlationId not in headers")
}
