package trace

import (
	"errors"
	"sort"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var ErrMissingLevel = errors.New("missing level")

type Config struct {
	RotateConfig `json:",inline" yaml:",inline"`
	// Level is the minimum enabled logging level. Note that this is a dynamic
	// level, so calling Config.Level.SetLevel will atomically change the log
	// level of all loggers descended from this config.
	Level zap.AtomicLevel `json:"level" yaml:"level"`
	// Development puts the logger in development mode, which changes the
	// behavior of DPanicLevel and takes stacktraces more liberally.
	Development bool `json:"development" yaml:"development"`
	// DisableCaller stops annotating logs with the calling function's file
	// name and line number. By default, all logs are annotated.
	DisableCaller bool `json:"disableCaller" yaml:"disableCaller"`
	// DisableStacktrace completely disables automatic stacktrace capturing. By
	// default, stacktraces are captured for WarnLevel and above logs in
	// development and ErrorLevel and above in production.
	DisableStacktrace bool `json:"disableStacktrace" yaml:"disableStacktrace"`
	// Sampling sets a sampling policy. A nil SamplingConfig disables sampling.
	Sampling *zap.SamplingConfig `json:"sampling" yaml:"sampling"`

	// OutputPaths is a list of URLs or file paths to write logging output to.
	// See Open for details.
	OutputPaths map[string][]string `json:"outputPaths" yaml:"outputPaths"`

	// InitialFields is a collection of fields to add to the root logger.
	InitialFields map[string]interface{} `json:"initialFields" yaml:"initialFields"`

	// max lines cached in memory before writing into files
	MaxQueueSize int64 `json:"maxQueueSize" yaml:"maxQueueSize"`

	INode uint64

	LogId int64
}

func (cfg *Config) buildOptions(errWriter zapcore.WriteSyncer) []zap.Option {
	opts := []zap.Option{zap.ErrorOutput(errWriter)}
	if cfg.Development {
		opts = append(opts, zap.Development())
	}
	if !cfg.DisableCaller {
		opts = append(opts, zap.AddCaller())
	}

	stackLevel := zap.ErrorLevel
	if cfg.Development {
		stackLevel = zap.WarnLevel
	}
	if !cfg.DisableStacktrace {
		opts = append(opts, zap.AddStacktrace(stackLevel))
	}

	if cfg.Sampling != nil {
		opts = append(opts, zap.WrapCore(func(core zapcore.Core) zapcore.Core {
			return zapcore.NewSampler(core, time.Second, int(cfg.Sampling.Initial), int(cfg.Sampling.Thereafter))
		}))
	}

	if len(cfg.InitialFields) > 0 {
		fs := make([]zap.Field, 0, len(cfg.InitialFields))
		keys := make([]string, 0, len(cfg.InitialFields))
		for k := range cfg.InitialFields {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fs = append(fs, zap.Any(k, cfg.InitialFields[k]))
		}
		opts = append(opts, zap.Fields(fs...))
	}

	return opts
}
