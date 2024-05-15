package trace

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Writer interface {
	zapcore.WriteSyncer
	io.Closer
}

type asyncWriter struct {
	quit   chan bool
	queue  chan []byte
	writer Writer
}

type closerWriter struct {
	zapcore.WriteSyncer
	closeChan chan struct{}
	closeFunc func() error
}

func (cw *closerWriter) Close() error {
	if cw.closeFunc != nil {
		return cw.closeFunc()
	}
	if cw.closeChan != nil {
		close(cw.closeChan)
	}
	return nil
}

type nopCloserWriter struct {
	zapcore.WriteSyncer
}

func (nopCloserWriter) Close() error {
	return nil
}

func NewAsyncRotateWriter(cfg *Config, module string) Writer {
	return NewAsyncWriter(NewRotateWriter(cfg, module))
}

func NewAsyncWriter(writer Writer) Writer {
	if config.MaxQueueSize == 0 {
		config.MaxQueueSize = 100000
	}
	asyncWriter := &asyncWriter{
		quit:   make(chan bool),
		queue:  make(chan []byte, config.MaxQueueSize),
		writer: writer,
	}
	go asyncWriter.watcher()
	return asyncWriter
}

func NewRotateWriter(cfg *Config, module string) Writer {
	var rotateWriters []Writer
	for mod, outputPaths := range cfg.OutputPaths {
		if mod != module {
			continue
		}

		for _, outputPath := range outputPaths {
			switch outputPath {
			case "stdout":
				rotateWriters = append(rotateWriters, &nopCloserWriter{os.Stdout})
			case "stderr":
				rotateWriters = append(rotateWriters, &nopCloserWriter{os.Stderr})
			default:
				rotateConfig := cfg.RotateConfig
				if rotateConfig.MaxSize == 0 {
					rotateConfig.MaxSize = 2048 // 单文件最大2GB
				}
				if rotateConfig.MaxAge == 0 {
					rotateConfig.MaxAge = 14 // 最多保留14天的日志
				}
				if rotateConfig.MaxBackups == 0 {
					rotateConfig.MaxBackups = 1000 // 最多保留近1000分备份
				}
				if rotateConfig.MaxBackupSize == 0 {
					rotateConfig.MaxBackupSize = 102400 // 最多保留100GB的备份
				}

				rotateWriter := &RotateWriter{
					RotateConfig: rotateConfig,
					LocalTime:    true,
				}
				rotateWriter.Filename, _ = filepath.Abs(outputPath)
				stat, err := os.Stat(rotateWriter.Filename)
				if err == nil || os.IsExist(err) {
					rotateWriter.INode = stat.Sys().(*syscall.Stat_t).Ino
				}

				quitChan := make(chan struct{})
				go fileScanner(rotateWriter, quitChan)

				rotateWriters = append(rotateWriters,
					&closerWriter{WriteSyncer: zapcore.AddSync(rotateWriter), closeChan: quitChan})
			}
		}
	}
	return combineWrites(rotateWriters...)
}

func combineWrites(writers ...Writer) Writer {
	syncers := make([]zapcore.WriteSyncer, len(writers))
	closers := make([]io.Closer, len(writers))
	for idx, writer := range writers {
		syncers[idx] = writer
		closers[idx] = writer
	}
	syncer := zap.CombineWriteSyncers(syncers...)
	closer := func() error {
		for idx := range closers {
			closers[idx].Close()
		}
		return nil
	}
	return &closerWriter{
		WriteSyncer: syncer,
		closeFunc:   closer,
	}
}

func fileScanner(rotateWriter *RotateWriter, quit chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			now := time.Now()
			stat, err := os.Stat(rotateWriter.Filename)
			if stat != nil && rotateWriter.INode == 0 {
				rotateWriter.INode = stat.Sys().(*syscall.Stat_t).Ino
			}
			if err != nil && os.IsNotExist(err) {
				fmt.Println("========fileScanner isnotexist")
				rotateWriter.Rotate()
			} else if now.Minute() == 0 && now.Second() == 0 {
				fmt.Println("========fileScanner min=0 sec=0")
				rotateWriter.Rotate()
			} else if stat != nil && stat.Sys().(*syscall.Stat_t).Ino != rotateWriter.INode {
				fmt.Println("========fileScanner openexist or new")
				rotateWriter.OpenExistingOrNew()
			}
		case <-quit:
			return

		}
	}
}

func (w *asyncWriter) Write(b []byte) (n int, err error) {
	data := make([]byte, len(b))
	copy(data, b)
	select {
	case w.queue <- data:
		return len(b), nil
	default:
		return 0, nil
	}
}

func (w *asyncWriter) Sync() error {
	timer := time.NewTimer(time.Second)
	defer timer.Stop()
	for {
		timer.Reset(time.Second)
		select {
		case msg := <-w.queue:
			w.writer.Write(msg)
		case <-timer.C:
			return nil
		}
	}
}

func (w *asyncWriter) Close() error {
	close(w.quit)
	return w.writer.Close()
}

func (w *asyncWriter) watcher() {
	for {
		select {
		case msg := <-w.queue:
			w.writer.Write(msg)
		case <-w.quit:
			return
		}
	}
}
