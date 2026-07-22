package log

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"time"
)

var logger *zap.SugaredLogger

const (
	DEBUG  = zapcore.DebugLevel
	INFO   = zapcore.InfoLevel
	WARN   = zapcore.WarnLevel
	ERROR  = zapcore.ErrorLevel
	DPANIC = zapcore.DPanicLevel
	PANIC  = zapcore.PanicLevel
	FATAL  = zapcore.FatalLevel
)

func init() {
	Init(GetConsoleZapcore(INFO))
}

// GetConsoleZapcore returns a console log core at the given level.
func GetConsoleZapcore(level zapcore.Level) zapcore.Core {
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	encoder.EncodeLevel = func(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString("[" + level.String() + "]")
	}
	encoder.EncodeCaller = nil
	encoder.ConsoleSeparator = " "
	return zapcore.NewCore(zapcore.NewConsoleEncoder(encoder), os.Stdout, zap.NewAtomicLevelAt(level))
}

// GetFileZapcore returns a file log core.
// absPath: absolute log file path.
// level: log level.
// fileMaxSize: max log file size in MB.
func GetFileZapcore(absPath string, level zapcore.Level, fileMaxSize int) zapcore.Core {
	syncWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:  absPath,
		MaxSize:   (1 << 20) * fileMaxSize, //20M
		LocalTime: true,
		Compress:  true,
	})
	encoder := zap.NewProductionEncoderConfig()
	encoder.EncodeTime = func(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
		enc.AppendString(t.Format("2006-01-02 15:04:05"))
	}
	return zapcore.NewCore(zapcore.NewJSONEncoder(encoder), syncWriter, zap.NewAtomicLevelAt(level))
}

func Init(cores ...zapcore.Core) {
	core := zapcore.NewTee(cores...)
	zapLogger := zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	logger = zapLogger.Sugar()
}

func Debug(args ...interface{}) {
	logger.Debug(args...)
}

func Debugf(template string, args ...interface{}) {
	logger.Debugf(template, args...)
}

func Info(args ...interface{}) {
	logger.Info(args...)
}

func Infof(template string, args ...interface{}) {
	logger.Infof(template, args...)
}

func Warn(args ...interface{}) {
	logger.Warn(args...)
}

func Warnf(template string, args ...interface{}) {
	logger.Warnf(template, args...)
}

func Error(args ...interface{}) {
	logger.Error(args...)
}

func Errorf(template string, args ...interface{}) {
	logger.Errorf(template, args...)
}

func DPanic(args ...interface{}) {
	logger.DPanic(args...)
}

func DPanicf(template string, args ...interface{}) {
	logger.DPanicf(template, args...)
}

func Panic(args ...interface{}) {
	logger.Panic(args...)
}

func Panicf(template string, args ...interface{}) {
	logger.Panicf(template, args...)
}

func Fatal(args ...interface{}) {
	logger.Fatal(args...)
}

func Fatalf(template string, args ...interface{}) {
	logger.Fatalf(template, args...)
}
