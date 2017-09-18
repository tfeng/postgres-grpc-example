package config

import "go.uber.org/zap"

var (
	Logger, _ = zap.NewDevelopment()
	Sugar     = Logger.Sugar()
)
