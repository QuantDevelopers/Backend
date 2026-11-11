package logger

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/QuantDevelopers/Backend/bootstrap"
	domainlogger "github.com/QuantDevelopers/Backend/internal/domain/logger"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	constants *bootstrap.Constants
	zap       *zap.Logger
}

var loggerInstance domainlogger.Logger

func NewLogger(config *bootstrap.Logger, constants *bootstrap.Constants) (*Logger, error) {
	if config.LogLevel == "" {
		config.LogLevel = constants.LogLevel.Info
	}

	var level zapcore.Level
	switch config.LogLevel {
	case constants.LogLevel.Debug:
		level = zapcore.DebugLevel
	case constants.LogLevel.Info:
		level = zapcore.InfoLevel
	case constants.LogLevel.Warn:
		level = zapcore.WarnLevel
	case constants.LogLevel.Error:
		level = zapcore.ErrorLevel
	case constants.LogLevel.Fatal:
		level = zapcore.FatalLevel
	default:
		level = zapcore.InfoLevel
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.CapitalLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	var cores []zapcore.Core
	encoder := zapcore.NewJSONEncoder(encoderConfig)

	consoleOutput, err := strconv.ParseBool(config.ConsoleOutput)
	if err != nil {
		consoleOutput = true
		log.Println("Error during checking console output enable. Set it to true by default")
	}
	if consoleOutput {
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(os.Stdout), level))
	}

	if config.LogFile != "" {
		file, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
		if err != nil {
			return nil, fmt.Errorf("failed to open log file: %w", err)
		}
		cores = append(cores, zapcore.NewCore(encoder, zapcore.AddSync(file), level))
	}

	core := zapcore.NewTee(cores...)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	instance := &Logger{
		constants: constants,
		zap:       zapLogger,
	}
	loggerInstance = instance
	return instance, nil
}

func GetLogger() domainlogger.Logger {
	if loggerInstance == nil {
		loggerInstance = &Logger{
			zap: zap.NewNop(),
		}
	}
	return loggerInstance
}

func toZapFields(fields []domainlogger.Field) []zap.Field {
	zapFields := make([]zap.Field, len(fields))
	for i, field := range fields {
		zapFields[i] = zap.Any(field.Key, field.Value)
	}
	return zapFields
}

func (l *Logger) Debug(msg string, fields ...domainlogger.Field) {
	l.zap.Debug(msg, toZapFields(fields)...)
}

func (l *Logger) Info(msg string, fields ...domainlogger.Field) {
	l.zap.Info(msg, toZapFields(fields)...)
}

func (l *Logger) Warn(msg string, fields ...domainlogger.Field) {
	l.zap.Warn(msg, toZapFields(fields)...)
}

func (l *Logger) Error(msg string, fields ...domainlogger.Field) {
	l.zap.Error(msg, toZapFields(fields)...)
}

func (l *Logger) Fatal(msg string, fields ...domainlogger.Field) {
	l.zap.Fatal(msg, toZapFields(fields)...)
}

func (l *Logger) WithFields(fields map[string]interface{}) domainlogger.Logger {
	zapFields := make([]zap.Field, 0, len(fields))
	for k, v := range fields {
		zapFields = append(zapFields, zap.Any(k, v))
	}
	return &Logger{
		constants: l.constants,
		zap:       l.zap.With(zapFields...),
	}
}

func (l *Logger) Close() {
	_ = l.zap.Sync()
}
