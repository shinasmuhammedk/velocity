package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var log *zap.Logger

// Init initializes the global logger.
// Call this once during application startup.
func Init(environment string) error {
	var cfg zap.Config

	if environment == "production" {
		cfg = zap.NewProductionConfig()
	} else {
		cfg = zap.NewDevelopmentConfig()
	}

	cfg.Encoding = "json"

	cfg.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeDuration: zapcore.StringDurationEncoder,
	}

	cfg.OutputPaths = []string{"stdout"}
	cfg.ErrorOutputPaths = []string{"stderr"}

	var err error

	log, err = cfg.Build(
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)

	if err != nil {
		return err
	}

	return nil
}

// Sync flushes buffered log entries.
func Sync() {
	if log != nil {
		_ = log.Sync()
	}
}

// Logger returns the underlying zap logger.
func Logger() *zap.Logger {
	return log
}

// ---------- Helper Methods ----------

func Debug(msg string, fields ...zap.Field) {
	log.Debug(msg, fields...)
}

func Info(msg string, fields ...zap.Field) {
	log.Info(msg, fields...)
}

func Warn(msg string, fields ...zap.Field) {
	log.Warn(msg, fields...)
}

func Error(msg string, fields ...zap.Field) {
	log.Error(msg, fields...)
}

func Fatal(msg string, fields ...zap.Field) {
	log.Fatal(msg, fields...)
}

func Panic(msg string, fields ...zap.Field) {
	log.Panic(msg, fields...)
}

// ---------- Field Helpers ----------

func String(key, value string) zap.Field {
	return zap.String(key, value)
}

func Int(key string, value int) zap.Field {
	return zap.Int(key, value)
}

func Int64(key string, value int64) zap.Field {
	return zap.Int64(key, value)
}

func Float64(key string, value float64) zap.Field {
	return zap.Float64(key, value)
}

func Bool(key string, value bool) zap.Field {
	return zap.Bool(key, value)
}

func Duration(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

func Any(key string, value interface{}) zap.Field {
	return zap.Any(key, value)
}

func ErrorField(err error) zap.Field {
	return zap.Error(err)
}

// ---------- Development Fallback ----------

func init() {
	log = zap.New(
		zapcore.NewCore(
			zapcore.NewJSONEncoder(zap.NewDevelopmentEncoderConfig()),
			zapcore.AddSync(os.Stdout),
			zap.DebugLevel,
		),
	)
}