package main

import (
	"sync"
	"time"

	log "git.xiaojukeji.com/golang/dlog"
)

var wg sync.WaitGroup

// MUST
func logDefault() {
	var conf log.LogConfig
	conf.Type = "file"
	conf.Level = "DEBUG"
	conf.FileName = "default"
	conf.FileRotateSize = 1024 * 1024 * 1024
	conf.FileRotateCount = 20
	conf.RotateByHour = true
	conf.KeepHours = 24

	log.Init(conf)
	defer wg.Done()
	for {
		log.Info("in logDefault function")
		time.Sleep(time.Millisecond * 2)
		log.Warning("in logDefault function")
	}
}

func logNonDefault() {
	var conf log.LogConfig
	conf.Type = "file"
	conf.Level = "DEBUG"
	conf.FileName = "nondefault"
	conf.FileRotateSize = 1024 * 1024 * 8
	conf.FileRotateCount = 20
	conf.RotateByHour = true
	conf.KeepHours = 12

	defer wg.Done()
	logger, err := log.NewLoggerFromConfig(conf)
	if err != nil {
		return
	}

	//log.SetRotateByHour(true)
	//log.SetKeepHours(12)

	for {
		logger.Info("in logNonDefault function")
		logger.Warning("in logNonDefault function")
		logger.Error("in logNonDefault function")
		logger.PrintfSimple("PrintfSimple: %s", "only a test")
		time.Sleep(time.Millisecond * 1)
	}
}

func seperateFileBackend() {

	b, err := log.NewFileBackend("./seperateFileBackend") //log文件目录
	if err != nil {
		panic(err)
	}

	log.SetLogging("INFO", b)   //只输出大于等于INFO的log
	b.Rotate(10, 1024*1024*500) //自动切分日志，保留10个文件（INFO.log.000-INFO.log.009，循环覆盖），每个文件大小为500M, 因为dlog支持多个文件后端， 所以需要为每个file backend指定具体切分数值
	b.SetRotateByHour(true)
	for {
		log.Warning("in seperateFileBackend function")
		log.Info("in seperateFileBackend function")
		time.Sleep(time.Second * 1)
	}

	log.Close()
}

func logPrintf() {
	var conf log.LogConfig
	conf.Type = "file"
	conf.Level = "DEBUG"
	conf.FileName = "printf"
	conf.FileRotateSize = 1024 * 1024 * 1024
	conf.FileRotateCount = 20
	conf.RotateByHour = true

	log.Init(conf)
	defer wg.Done()
	for {
		log.Printf("in logDefault function")
		time.Sleep(time.Millisecond * 2)
	}
}

func main() {
	//wg.Add(1)
	//go logDefault()
	wg.Add(1)
	go logNonDefault()
	//wg.Add(1)
	//go seperateFileBackend()
	//wg.Add(1)
	//go logPrintf()
	wg.Wait()
}
