package trace

import (
	"context"
	"testing"
	"time"

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
			"ping":  []string{"ping.log"},
			"probe": []string{"probe.log"},
		},
	}, "/Users/baoqingzhang/work/gopath/src/github.com/tools-go/go-utils/trace/logs/")
}

func TestNewLogger(t *testing.T) {
	l := NewLogger(0, "ping")
	l.WithBF(context.Background())
	nll := l.Clone().WithBF(context.TODO())
	nll.Infof("==========info=222")

	go func() {
		for {
			pl := NewLogger(1, "probe").WithBF(context.TODO())
			pl.Info("======probe==1111===")
			time.Sleep(2 * time.Second)
		}
	}()
	go func() {
		for {
			pl := NewLogger(1, "probe").WithBF(context.TODO())
			pl.Info("======probe===222==")
			time.Sleep(1 * time.Second)
		}
	}()

	select {}
}
