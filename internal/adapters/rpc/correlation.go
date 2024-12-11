package rpc

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/proto"
)

func getCorrelationID(ctx context.Context) (uuid.UUID, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return uuid.Max, errors.New("failed to extract metadata")
	}
	// handle request is from REST and gRPC client
	match := md.Get(correlationHeader)
	match = append(match, md.Get(gCorrelationMD)...)
	if len(match) > 0 {
		id, err := uuid.Parse(match[0])
		if err != nil {
			return uuid.Max, fmt.Errorf("failed to parse uuid form correlation header: %w", err)
		}
		return id, nil
	}
	return uuid.Max, errors.New("correlationID not in request metadata/headers")
}

func correlationIDResponseModifier(
	ctx context.Context,
	w http.ResponseWriter,
	p proto.Message,
) error {
	hv := w.Header().Get(gCorrelationMD)
	if hv != "" {
		w.Header().Set(correlationHeader, hv)
	} else {
		w.Header().Set(correlationHeader, "NO_TRACE")
	}
	// remove unwanted metadata
	delete(w.Header(), gCorrelationMD)
	return nil
}
