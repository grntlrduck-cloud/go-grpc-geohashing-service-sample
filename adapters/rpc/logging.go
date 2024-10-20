package rpc

import (
	"context"
	"fmt"

	grpclogging "github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/logging"
	"go.uber.org/zap"
)

// https://github.com/grpc-ecosystem/go-grpc-middleware/blob/main/interceptors/logging/examples/zap/example_test.go
func InterceptorLogger(l *zap.Logger) grpclogging.Logger {
	return grpclogging.LoggerFunc(
		func(ctx context.Context, lvl grpclogging.Level, msg string, fields ...any) {
			f := make([]zap.Field, 0, len(fields)/2)

			for i := 0; i < len(fields); i += 2 {
				key := fields[i]
				value := fields[i+1]

				switch v := value.(type) {
				case string:
					f = append(f, zap.String(key.(string), v))
				case int:
					f = append(f, zap.Int(key.(string), v))
				case bool:
					f = append(f, zap.Bool(key.(string), v))
				default:
					f = append(f, zap.Any(key.(string), v))
				}
			}

			logger := l.WithOptions(zap.AddCallerSkip(1)).With(f...)

			switch lvl {
			case grpclogging.LevelDebug:
				logger.Debug(msg)
			case grpclogging.LevelInfo:
				logger.Debug(msg) // mapped to debug to reduce noise
			case grpclogging.LevelWarn:
				logger.Warn(msg)
			case grpclogging.LevelError:
				logger.Error(msg)
			default:
				panic(fmt.Sprintf("unknown level %v", lvl))
			}
		},
	)
}
