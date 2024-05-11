package log

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func init() {
	SetConfig(Config{
		Level: zap.NewAtomicLevelAt(zap.DebugLevel),
		RotateConfig: RotateConfig{
			MaxAge:     1,
			MaxSize:    10,
			MaxBackups: 10,
		},
		OutputPaths: map[string][]string{
			"unkongwn": []string{""},
		},
	}, "stdout")
}

func TestNewLogger(t *testing.T) {
	l := NewLogger(0, "unkongwn")
	l.WithBF(context.TODO())
	l.Info("========info==")
}
