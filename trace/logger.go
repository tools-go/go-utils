package trace

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/tools-go/go-utils/utils"
)

var (
	config Config

	CommonLogger *Logger
)

type Fields = []zap.Field

type Logger struct {
	*zap.Logger
	*Trace
	LogId int64
}

func SetConfig(c Config, logDir string) {
	if logDir == "stdout" {
		for i, vv := range c.OutputPaths {
			for j := range vv {
				c.OutputPaths[i][j] = logDir
			}
		}
	} else {
		for i, vv := range c.OutputPaths {
			for j, v := range vv {
				c.OutputPaths[i][j] = logDir + v
			}
		}
	}

	config = c
}

func NewLogger(logId int64, module string) *Logger {
	l := &Logger{LogId: logId}
	//if CommonLogger != nil {
	//	l.Logger = CommonLogger.Logger.WithOptions() // 等价于clone
	//	return l
	//}

	encoder := NewFCLogEncoder(zapcore.EncoderConfig{
		MessageKey:     "M",
		LevelKey:       "L",
		TimeKey:        "T",
		NameKey:        "N",
		CallerKey:      "C",
		StacktraceKey:  "",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    LevelEncoder,
		EncodeTime:     TimeEncoder,
		EncodeDuration: DurationEncoder,
		EncodeCaller:   CallerEncoder,
		EncodeName:     nil,
	})
	writer := NewAsyncRotateWriter(&config, module)
	if config.Level == (zap.AtomicLevel{}) {
		panic(ErrMissingLevel)
	}

	l.Logger = zap.New(zapcore.NewCore(encoder, writer, config.Level), config.buildOptions(writer)...)
	return l
}

func (l *Logger) Clone() *Logger {
	return &Logger{
		LogId:  utils.GenerateId(),
		Logger: l.Logger.WithOptions(),
	}
}

func (l *Logger) getBaseFields(ctx context.Context) []zap.Field {
	trace := l.GetTrace(ctx)
	return []zap.Field{
		zap.String("traceid", trace.TraceId),
		zap.String("spanid", trace.SpanId),
		zap.Int64("logid", l.LogId),
	}
}

func (l *Logger) With(fields ...zap.Field) *Logger {
	l.Logger = l.Logger.With(fields...)
	return l
}

func (l *Logger) WithBF(ctx context.Context) *Logger {
	return l.With(l.getBaseFields(ctx)...)
}

func (l *Logger) Debug(msg string, fields ...zap.Field) {
	l.Logger.Debug(msg, fields...)
}

func (l *Logger) Debugf(format string, v ...interface{}) {
	l.Logger.Debug(fmt.Sprintf(format, v...))
}

func (l *Logger) Debugln(v ...interface{}) {
	l.Logger.Debug(fmt.Sprint(v...))
}

func (l *Logger) Info(msg string, fields ...zap.Field) {
	l.Logger.Info(msg, fields...)
}

func (l *Logger) Infof(format string, v ...interface{}) {
	l.Logger.Info(fmt.Sprintf(format, v...))
}

func (l *Logger) Infoln(v ...interface{}) {
	l.Logger.Info(fmt.Sprint(v...))
}

func (l *Logger) Warn(msg string, fields ...zap.Field) {
	l.Logger.Warn(msg, fields...)
}

func (l *Logger) Warnf(format string, v ...interface{}) {
	l.Logger.Warn(fmt.Sprintf(format, v...))
}

func (l *Logger) Warnln(v ...interface{}) {
	l.Logger.Warn(fmt.Sprint(v...))
}

func (l *Logger) Error(msg string, fields ...zap.Field) {
	l.Logger.Error(msg, fields...)
}

func (l *Logger) Errorf(format string, v ...interface{}) {
	l.Logger.Error(fmt.Sprintf(format, v...))
}

func (l *Logger) Errorln(v ...interface{}) {
	l.Logger.Error(fmt.Sprint(v...))
}

func (l *Logger) DPanic(msg string, fields ...zap.Field) {
	l.Logger.DPanic(msg, fields...)
}

func (l *Logger) DPanicf(format string, v ...interface{}) {
	l.Logger.DPanic(fmt.Sprintf(format, v...))
}

func (l *Logger) DPanicln(v ...interface{}) {
	l.Logger.DPanic(fmt.Sprint(v...))
}

func (l *Logger) Panic(msg string, fields ...zap.Field) {
	l.Logger.Panic(msg, fields...)
}

func (l *Logger) Panicf(format string, v ...interface{}) {
	l.Logger.Panic(fmt.Sprintf(format, v...))
}

func (l *Logger) Panicln(v ...interface{}) {
	l.Logger.Panic(fmt.Sprint(v...))
}

func (l *Logger) Fatal(msg string, fields ...zap.Field) {
	l.Logger.Fatal(msg, fields...)
}

func (l *Logger) Fatalf(format string, v ...interface{}) {
	l.Logger.Fatal(fmt.Sprintf(format, v...))
}

func (l *Logger) Fataln(v ...interface{}) {
	l.Logger.Fatal(fmt.Sprint(v...))
}

func (l *Logger) Print(msg string, fields ...zap.Field) {
	l.Info(msg, fields...)
}

func (l *Logger) Printf(format string, v ...interface{}) {
	l.Infof(format, v...)
}

func (l *Logger) Println(v ...interface{}) {
	l.Infoln(v...)
}

func BeginWith(tag string, st ...time.Time) Fields {
	if tag == LOGTAG_REQUEST_IN {
		return []zap.Field{zap.String(LOGKEY_TAG, tag)}
	}

	if len(st) == 0 {
		return []zap.Field{
			zap.String(LOGKEY_TAG, tag),
			zap.Time(LOGKEY_BEGIN, time.Now()),
		}
	}

	return []zap.Field{
		zap.String(LOGKEY_TAG, tag),
		zap.Time(LOGKEY_BEGIN, st[0]),
	}
}

func (logger *Logger) LogWithError(err error, fieldss ...Fields) {
	f := make([]zap.Field, 0)
	for _, fields := range fieldss {
		for _, field := range fields {
			switch field.Key {
			case LOGKEY_TAG:
				if _, ok := TagSuccessFailureRelation[field.String]; ok && err != nil {
					f = append(f, zap.String(LOGKEY_TAG, TagSuccessFailureRelation[field.String]))
				} else {
					f = append(f, field)
				}
			case LOGKEY_BEGIN:
				t := time.Unix(0, field.Integer)
				f = append(f, zap.Duration("proc_time", time.Since(t)))
			default:
				f = append(f, field)
			}
		}
	}
	if err != nil {
		logger.Error(strings.ReplaceAll(err.Error(), "\n", " "), f...)
	} else {
		logger.Info(LOG_OK, f...)
	}
}

func Debug(msg string, fields ...zap.Field) {
	CommonLogger.Debug(msg, fields...)
}

func Debugf(format string, v ...interface{}) {
	CommonLogger.Debug(fmt.Sprintf(format, v...))
}

func Debugln(v ...interface{}) {
	CommonLogger.Debug(fmt.Sprint(v...))
}

func Info(msg string, fields ...zap.Field) {
	CommonLogger.Info(msg, fields...)
}

func Infof(format string, v ...interface{}) {
	CommonLogger.Info(fmt.Sprintf(format, v...))
}

func Infoln(v ...interface{}) {
	CommonLogger.Info(fmt.Sprint(v...))
}

func Warn(msg string, fields ...zap.Field) {
	CommonLogger.Warn(msg, fields...)
}

func Warnf(format string, v ...interface{}) {
	CommonLogger.Warn(fmt.Sprintf(format, v...))
}

func Warnln(v ...interface{}) {
	CommonLogger.Warn(fmt.Sprint(v...))
}

func Error(msg string, fields ...zap.Field) {
	CommonLogger.Error(msg, fields...)
}

func Errorf(format string, v ...interface{}) {
	CommonLogger.Error(fmt.Sprintf(format, v...))
}

func Errorln(v ...interface{}) {
	CommonLogger.Error(fmt.Sprint(v...))
}

func DPanic(msg string, fields ...zap.Field) {
	CommonLogger.DPanic(msg, fields...)
}

func DPanicf(format string, v ...interface{}) {
	CommonLogger.DPanic(fmt.Sprintf(format, v...))
}

func DPanicln(v ...interface{}) {
	CommonLogger.DPanic(fmt.Sprint(v...))
}

func Panic(msg string, fields ...zap.Field) {
	CommonLogger.Panic(msg, fields...)
}

func Panicf(format string, v ...interface{}) {
	CommonLogger.Panic(fmt.Sprintf(format, v...))
}

func Panicln(v ...interface{}) {
	CommonLogger.Panic(fmt.Sprint(v...))
}

func Fatal(msg string, fields ...zap.Field) {
	CommonLogger.Fatal(msg, fields...)
}

func Fatalf(format string, v ...interface{}) {
	CommonLogger.Fatal(fmt.Sprintf(format, v...))
}

func Fataln(v ...interface{}) {
	CommonLogger.Fatal(fmt.Sprint(v...))
}

func Print(msg string, fields ...zap.Field) {
	CommonLogger.Info(msg, fields...)
}

func Printf(format string, v ...interface{}) {
	CommonLogger.Infof(format, v...)
}

func Println(v ...interface{}) {
	CommonLogger.Infoln(v...)
}

func NewTestLogger(logId int64) *Logger {
	l := &Logger{LogId: logId}
	logger, err := zap.NewDevelopmentConfig().Build()
	if err != nil {
		panic(err)
	}
	l.Logger = logger

	return l
}
