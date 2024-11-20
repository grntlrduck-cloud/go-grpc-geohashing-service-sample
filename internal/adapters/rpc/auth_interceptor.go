package rpc

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

const healthServiceMethodName = "/api.v1.health.HealthService/HealthCheck"

type KeyAuthInterceptor struct {
	secretValue string
}

func NewKeyAuthInterceptor(secret string) (*KeyAuthInterceptor, error) {
	hashFunc := sha256.New()
	_, err := hashFunc.Write([]byte(secret))
	if err != nil {
		return nil, errors.New("failed to hash secret, can not initialize KeyAuthInterceptor")
	}
	hash := hashFunc.Sum(nil)

	return &KeyAuthInterceptor{secretValue: hex.EncodeToString(hash)}, nil
}

func (k *KeyAuthInterceptor) UnaryKeyAuthorizer() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
		if strings.Contains(info.FullMethod, healthServiceMethodName) {
			return handler(ctx, req)
		}
		requestKey, err := getApiKeyFromContext(ctx)
		if err != nil {
			return nil, status.Error(codes.Unauthenticated, "unauthenticated, key missing")
		}

		// now we hash the key so we have a constant length to compare to prevent timing attacks for length determination
		hashFunc := sha256.New()
		_, err = hashFunc.Write([]byte(requestKey))
		if err != nil {
			return nil, status.Error(codes.Internal, "failed to verify auth")
		}

		requestKeyHash := hashFunc.Sum(nil)
		hexEnc := hex.EncodeToString(requestKeyHash)
		if hexEnc == k.secretValue {
			return handler(ctx, req)
		}
		return nil, status.Error(codes.PermissionDenied, "invalid key")
	}
}

func getApiKeyFromContext(ctx context.Context) (string, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "", errors.New("failed to extract metadata")
	}
	// extract grpc metadata and header from REST request to handle both cases
	match := md.Get(apiKeyHeader)
	match = append(match, md.Get(gApiKeyMetadata)...)
	if len(match) > 0 {
		return match[0], nil
	}
	return "", errors.New("api key not in request metadata/headers")
}
