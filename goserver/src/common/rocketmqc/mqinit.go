package rocketmqc

import (
	"common/tlog"
	"github.com/apache/rocketmq-client-go/v2/rlog"
	"strings"
	"sync"
)

var gRocketmqOnce sync.Once

func initRocketmq() {
	gRocketmqOnce.Do(func() {
		rlog.SetLogger(&rocketmqLogger{})
		//rlog.SetLogLevel("info")
		rlog.SetLogLevel("warn")
	})
}

type rocketmqLogger struct {
	level tlog.LEVEL
}

func (this *rocketmqLogger) Level(level string) {
	switch strings.ToLower(level) {
	case "debug":
		this.level = tlog.DEBUG
	case "warn":
		this.level = tlog.WARNING
	case "error":
		this.level = tlog.ERROR
	case "fatal":
		this.level = tlog.FATAL
	default:
		this.level = tlog.INFO
	}
}

func (this *rocketmqLogger) Debug(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	if this.level <= tlog.DEBUG {
		tlog.Debug(msg, fields)
	}
}

func (this *rocketmqLogger) Info(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	if this.level <= tlog.INFO {
		tlog.Info(msg, fields)
	}
}

func (this *rocketmqLogger) Warning(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	if this.level <= tlog.WARNING {
		tlog.Warning(msg, fields)
	}
}

func (this *rocketmqLogger) Error(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	if this.level <= tlog.ERROR {
		tlog.Error(msg, fields)
	}
}

func (this *rocketmqLogger) Fatal(msg string, fields map[string]interface{}) {
	if msg == "" && len(fields) == 0 {
		return
	}
	if this.level <= tlog.FATAL {
		tlog.Fatal(msg, fields)
	}
}
