package zaplog

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger Log component
var Logger Logr

func init() {
	InitLoggers("./logs", NewDefaultRotate())
}

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = Level(zap.DebugLevel)
	// InfoLevel is the default logging priority.
	InfoLevel = Level(zap.InfoLevel)
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = Level(zap.WarnLevel)
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = Level(zap.ErrorLevel)
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = Level(zapcore.FatalLevel)
)

const tracerLogHandlerID = 10204

var levelFileName = map[string]Level{"INFO": InfoLevel, "WARNING": WarnLevel, "DEBUG": DebugLevel, "ERROR": ErrorLevel}

type Level zapcore.Level
type LevelEnableFunc func(level Level) bool
type Option zap.Option

type teeOpt struct {
	Filepath string
	LevelF   LevelEnableFunc
	Rot      *RotateOption
}

type logger struct {
	l     *zap.SugaredLogger
	check func(l *zap.SugaredLogger) bool
}

type Logr interface {
	Infof(template string, args ...interface{})
	Info(args ...interface{})
	Errorf(template string, args ...interface{})
	Error(args ...interface{})
	Fatalf(template string, args ...interface{})
	Fatal(args ...interface{})
	Warnf(template string, args ...interface{})
	Warn(args ...interface{})
	With(args ...interface{}) Logr
	Debugf(template string, args ...interface{})
	Debug(args ...interface{})
	Sync() error
}

type RotateOption struct {
	MaxSize    int
	MaxAge     int
	MaxBackups int
	Compress   bool
}
type OptionFunc func(option *RotateOption)

func NewDefaultRotate(opts ...OptionFunc) *RotateOption {
	ro := &RotateOption{
		MaxSize:    2 * 1024, //2G
		MaxAge:     3,        //保留7天
		MaxBackups: 100,      // 保留文件数
		Compress:   false,    // 不压缩为.gz包
	}
	for _, f := range opts {
		f(ro)
	}
	return ro
}

func WithLogSaveDay(day int) OptionFunc {
	return func(r *RotateOption) {
		if day == 0 {
			day = 7
		}
		r.MaxAge = day
	}
}

func newTee(tops []teeOpt) *logger {
	var cores []zapcore.Core
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(time.Format("2006-01-02 15:04:05"))
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	for _, top := range tops {
		if top.Filepath == "" {
			panic("log filepath is empty")
		}
		lv := zap.LevelEnablerFunc(func(level zapcore.Level) bool {
			return top.LevelF(Level(level))
		})

		w := &lumberjack.Logger{
			Filename:   top.Filepath,
			MaxSize:    top.Rot.MaxSize,
			MaxAge:     top.Rot.MaxAge,
			MaxBackups: top.Rot.MaxBackups,
			LocalTime:  true,
			Compress:   top.Rot.Compress,
		}
		zap.AddCaller()

		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg.EncoderConfig),
			zapcore.AddSync(w),
			lv,
		)
		cores = append(cores, core)
	}
	// 同时日志打印到终端
	core := zapcore.NewCore(zapcore.NewConsoleEncoder(cfg.EncoderConfig), os.Stdout, zap.DebugLevel)

	cores = append(cores, core)
	logger := &logger{
		l:     zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(1)).Sugar(),
		check: check,
	}
	return logger
}

func InitLoggers(logDirPath string, opt *RotateOption) {
	if logDirPath == "" {
		logDirPath = "./logs"
	}
	if opt == nil {
		panic(fmt.Errorf("rotate option is nil"))
	}
	var tops []teeOpt
	for name := range levelFileName {
		tops = append(tops, teeOpt{
			Filepath: fmt.Sprintf("%s/%s.log", logDirPath, name),
			Rot:      opt,
			LevelF: func(fname string) LevelEnableFunc {
				return func(l Level) bool {
					level := InfoLevel
					if lv, ok := levelFileName[fname]; ok {
						level = lv
					}
					return l >= level
				}
			}(name),
		})
	}
	Logger = newTee(tops)
}

func check(l *zap.SugaredLogger) bool {
	return l != nil
}

// fields must be k/v format
func WithFieldsContext(ctx context.Context, fields ...interface{}) context.Context {
	lg := Logger.(*logger)
	return context.WithValue(ctx, tracerLogHandlerID, clone(lg.l.With(fields...)))
}

func GetLogFromContext(ctx context.Context) Logr {
	if l, ok := ctx.Value(tracerLogHandlerID).(*logger); ok {
		return l
	}
	return Logger
}

func WithTraceID() string {
	return uuid.New().String()
}

func (l *logger) Infof(template string, args ...interface{}) {
	if l.check(l.l) {
		l.l.Infof(template, args...)
	}
}

func clone(l *zap.SugaredLogger) *logger {
	return &logger{
		l:     l,
		check: check,
	}
}

func (l *logger) Info(args ...interface{}) {
	if l.check(l.l) {
		l.l.Info(args...)
	}
}

func (l *logger) Errorf(template string, args ...interface{}) {
	if l.check(l.l) {
		l.l.Errorf(template, args...)
	}
}

func (l *logger) Error(args ...interface{}) {
	if l.check(l.l) {
		l.l.Error(args...)
	}
}

func (l *logger) Fatalf(template string, args ...interface{}) {
	if l.check(l.l) {
		l.l.Fatalf(template, args...)
	}
}

func (l *logger) Fatal(args ...interface{}) {
	if l.check(l.l) {
		l.l.Fatal(args...)
	}
}

func (l *logger) Warnf(template string, args ...interface{}) {
	if l.check(l.l) {
		l.l.Warnf(template, args...)
	}
}

func (l *logger) Warn(args ...interface{}) {
	if l.check(l.l) {
		l.l.Warn(args...)
	}
}

func (l *logger) Debugf(template string, args ...interface{}) {
	if l.check(l.l) {
		l.l.Debugf(template, args...)
	}
}

func (l *logger) Debug(args ...interface{}) {
	if l.check(l.l) {
		l.l.Debug(args...)

	}
}

// must bu k/v format
func (l *logger) With(args ...interface{}) Logr {
	if l.check(l.l) {
		l.l = l.l.With(args...)
	}
	return l
}

// Sync flushes any buffered log entries.
func (l *logger) Sync() error {
	if l.check(l.l) {
		return l.l.Sync()
	}
	return nil
}
