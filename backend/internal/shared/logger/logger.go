package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

func Init(env string) {
	var cfg zap.Config

	if env == "production" {
		cfg = zap.NewProductionConfig()
		cfg.EncoderConfig.TimeKey = "timestamp"
		cfg.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	} else {
		cfg = zap.NewDevelopmentConfig()
		cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
	}

	var err error
	log, err = cfg.Build(zap.AddCallerSkip(1))
	if err != nil {
		// If logger fails to init, write to stderr and exit.
		os.Stderr.WriteString("failed to initialize logger: " + err.Error())
		os.Exit(1)
	}
}

func get() *zap.Logger {
	if log == nil {
		Init("development")
	}
	return log
}

func Info(msg string, fields ...zap.Field) {
	get().Info(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	get().Error(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	get().Warn(msg, fields...)
}

func Debug(msg string, fields ...zap.Field) {
	get().Debug(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	get().Fatal(msg, fields...)
}

func With(fields ...zap.Field) *zap.Logger {
	return get().With(fields...)
}

func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}
