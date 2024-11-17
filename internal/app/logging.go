package app

import (
	"log"

	"go.uber.org/zap"
)

func NewLogger(config *LoggingConfig) *zap.Logger {
	logger, err := zap.NewProduction(
		zap.Fields(
			baseLoggerFields(config)...,
		),
	)
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
		panic(err)
	}
	return logger
}

func NewDevLogger(config *LoggingConfig) *zap.Logger {
	logger, err := zap.NewDevelopment(
		zap.Fields(
			baseLoggerFields(config)...,
		),
	)
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
		panic(err)
	}
	return logger
}

func baseLoggerFields(config *LoggingConfig) []zap.Field {
	if config != nil {
		return []zap.Field{
			zap.String("level", config.Level),
			zap.String("env", config.Env),
			zap.String("host", config.Host),
			zap.String("app_name", config.AppName),
			zap.String("region", config.Region),
			zap.String("account", config.Account),
			zap.String("team_name", config.TeamName),
		}
	}
	return []zap.Field{}
}
