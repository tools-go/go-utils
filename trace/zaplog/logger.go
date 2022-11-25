package zaplog

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/natefinch/lumberjack"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	Deflogging logger
	program    = filepath.Base(os.Args[0])
	host       = "unknownhost"

	verbosity = "LOGVERB"
)

func init() {
	InitLogers("", nil)
	h, err := os.Hostname()
	if err == nil {
		host = shortHostname(h)
	}
}

// shortHostname returns its argument, truncating at the first period.
// For instance, given "www.google.com" it returns "www".
func shortHostname(hostname string) string {
	if i := strings.Index(hostname, "."); i >= 0 {
		return hostname[:i]
	}
	return hostname
}

func InitFlags(flagset *flag.FlagSet) {
	if flagset == nil {
		flagset = flag.CommandLine
	}
	flagset.Var(&Deflogging.verbosity, "v", "number for the log level verbosity")
}

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = Level(zap.DebugLevel)
	// InfoLevel is the default Deflogging priority.
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

func (l Level) Enabled(level zapcore.Level) bool {
	return Level(level) >= l
}

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

func getVerbosityFromEnv() Level {
	verb := os.Getenv(verbosity)
	if len(verb) == 0 {
		return 0
	}
	v, err := strconv.Atoi(verb)
	if err != nil {
		return 0
	}
	return Level(v)
}

// Get is part of the flag.Getter interface.
func (l *Level) Get() Level {
	if *l == 0 {
		return getVerbosityFromEnv()
	}
	return *l
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.ParseInt(value, 10, 8)
	if err != nil {
		return err
	}
	*l = Level(v)
	return nil
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
	Enable() bool
}

type logger struct {
	l         *zap.SugaredLogger
	check     func(l *zap.SugaredLogger) bool
	opt       *RotateOption
	logDir    string
	verbosity Level
}

func Infof(template string, args ...interface{}) {
	Deflogging.Infof(template, args...)
}
func Info(args ...interface{}) {
	Deflogging.Info(args...)
}
func Errorf(template string, args ...interface{}) {
	Deflogging.Errorf(template, args...)
}
func Error(args ...interface{}) {
	Deflogging.Error(args...)
}
func Fatalf(template string, args ...interface{}) {
	Deflogging.Fatalf(template, args...)
}
func Fatal(args ...interface{}) {
	Deflogging.Fatal(args...)
}
func Warnf(template string, args ...interface{}) {
	Deflogging.Warnf(template, args...)
}
func Warn(args ...interface{}) {
	Deflogging.Warn(args...)
}
func With(args ...interface{}) Logr {
	return Deflogging.With(args...)
}

func Debugf(template string, args ...interface{}) {
	Deflogging.Debugf(template, args...)
}
func Debug(args ...interface{}) {
	Deflogging.Debug(args...)
}
func Sync() error {
	return Deflogging.Sync()
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
		MaxAge:     7,        //保留7天
		MaxBackups: 500,      // 保留文件数
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

// verbose is a boolean type that implements Infof (like Printf) etc.
// See the documentation of V for more information.
type verbose struct {
	enabled bool
}

// 1. 支持命令行参数指定Level, eg:  -v=3
// 2. 支持环境变量支持Level, export LOGVERB=3
func V(level Level) Logr {
	if Deflogging.verbosity.Get() >= level {
		return newVerbose(true)
	}
	return newVerbose(false)
}

func newVerbose(b bool) *verbose {
	if Deflogging.l == nil {
		Deflogging.l = newLogger(Deflogging.logDir, Deflogging.opt)
	}
	return &verbose{b}
}

func newZapCfg() zap.Config {
	cfg := zap.NewProductionConfig()
	cfg.EncoderConfig.EncodeTime = func(time time.Time, encoder zapcore.PrimitiveArrayEncoder) {
		encoder.AppendString(time.Format("2006-01-02 15:04:05"))
	}
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
	return cfg
}

func newStderrCore(cfg zap.Config) zapcore.Core {
	// 同时日志打印到终端
	return zapcore.NewCore(zapcore.NewConsoleEncoder(cfg.EncoderConfig), os.Stdout, zap.DebugLevel)
}

func newFileCores(logDirPath string, cfg zap.Config, opt *RotateOption) []zapcore.Core {
	if opt == nil {
		opt = NewDefaultRotate()
	}
	if len(logDirPath) <= 0 {
		return nil
	}
	Deflogging.opt = opt
	Deflogging.logDir = logDirPath
	var cores []zapcore.Core
	for name := range levelFileName {
		w := &lumberjack.Logger{
			Filename:   filepath.Join(logDirPath, fmt.Sprintf("%s.%s.%s.log", program, host, name)),
			MaxSize:    opt.MaxSize,
			MaxAge:     opt.MaxAge,
			MaxBackups: opt.MaxBackups,
			LocalTime:  true,
			Compress:   opt.Compress,
		}
		level := InfoLevel
		if lv, ok := levelFileName[name]; ok {
			level = lv
		}
		core := zapcore.NewCore(
			zapcore.NewConsoleEncoder(cfg.EncoderConfig),
			zapcore.AddSync(w),
			level,
		)
		cores = append(cores, core)
	}
	return cores
}

func newLogger(logDirPath string, opt *RotateOption) *zap.SugaredLogger {
	var cores []zapcore.Core
	cfg := newZapCfg()
	// 输出到标准错误
	cores = newFileCores(logDirPath, cfg, opt)
	// 输出到文件
	cores = append(cores, newStderrCore(cfg))
	return zap.New(zapcore.NewTee(cores...), zap.AddCaller(), zap.AddCallerSkip(3)).Sugar()
}

func InitLogers(logDirPath string, opt *RotateOption) {
	if opt == nil {
		opt = NewDefaultRotate()
	}
	Deflogging.l = newLogger(logDirPath, opt)
	Deflogging.check = check
}

func check(l *zap.SugaredLogger) bool {
	return l != nil
}

// fields must be k/v format
func WithFieldsContext(ctx context.Context, fields ...interface{}) context.Context {
	return context.WithValue(ctx, tracerLogHandlerID, clone(Deflogging.l.With(fields...)))
}

func GetLogFromContext(ctx context.Context) *logger {
	if l, ok := ctx.Value(tracerLogHandlerID).(*logger); ok {
		return l
	}
	return &Deflogging
}

func WithTraceID() string {
	return uuid.New().String()
}

func clone(l *zap.SugaredLogger) *logger {
	return &logger{
		l:     l,
		check: check,
	}
}

func (l *logger) Infof(template string, args ...interface{}) {
	l.l.Infof(template, args...)
}
func (l *logger) Enable() bool {
	return true
}

func (l *logger) Info(args ...interface{}) {
	l.l.Info(args...)
}

func (l *logger) Errorf(template string, args ...interface{}) {
	l.l.Errorf(template, args...)
}

func (l *logger) Error(args ...interface{}) {
	l.l.Error(args...)
}

func (l *logger) Fatalf(template string, args ...interface{}) {
	l.l.Fatalf(template, args...)
}

func (l *logger) Fatal(args ...interface{}) {
	l.l.Fatal(args...)
}

func (l *logger) Warnf(template string, args ...interface{}) {
	l.l.Warnf(template, args...)
}

func (l *logger) Warn(args ...interface{}) {
	l.l.Warn(args...)
}

func (l *logger) Debugf(template string, args ...interface{}) {
	l.l.Debugf(template, args...)
}

func (l *logger) Debug(args ...interface{}) {
	l.l.Debug(args...)
}

// must bu k/v format
func (l *logger) With(args ...interface{}) Logr {
	l.l = l.l.With(args...)
	return l
}

// Sync flushes any buffered log entries.
func (l *logger) Sync() error {
	return l.l.Sync()
}

func (l *verbose) Infof(template string, args ...interface{}) {
	if l.enabled {
		Deflogging.l.Infof(template, args...)
	}
}

func (l *verbose) Info(args ...interface{}) {
	if l.enabled {
		Deflogging.l.Info(args...)
	}
}

func (l *verbose) Errorf(template string, args ...interface{}) {
	if l.enabled {
		Deflogging.l.Errorf(template, args...)
	}
}

func (l *verbose) Error(args ...interface{}) {
	if l.enabled {
		Deflogging.l.Error(args...)
	}
}

func (l *verbose) Fatalf(template string, args ...interface{}) {
	if l.enabled {
		Deflogging.l.Fatalf(template, args...)
	}
}

func (l *verbose) Fatal(args ...interface{}) {
	if l.enabled {
		Deflogging.l.Fatal(args...)
	}
}

func (l *verbose) Warnf(template string, args ...interface{}) {
	if l.enabled {
		Deflogging.l.Warnf(template, args...)
	}
}

func (l *verbose) Warn(args ...interface{}) {
	if l.enabled {
		Deflogging.l.Warn(args...)
	}
}
func (l *verbose) With(args ...interface{}) Logr {
	if l.enabled {
		Deflogging.l.With(args...)
	}
	return l
}

func (l *verbose) Debugf(template string, args ...interface{}) {
	if l.enabled {
		Deflogging.l.Debugf(template, args...)
	}
}

func (l *verbose) Debug(args ...interface{}) {
	if l.enabled {
		Deflogging.l.Debug(args...)
	}
}

func (l *verbose) Enable() bool {
	return l.enabled
}
