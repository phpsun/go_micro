package datalogger

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"time"
)

const (
	PartitionDay  = 0
	PartitionHour = 1
)

type DataConfig struct {
	Dir       string         `toml:"dir"`
	Partition int            `toml:"partition"`
	Timezone  string         `toml:"timezone"`
	Location  *time.Location `toml:"-"`
}

type DataLogger struct {
	c        DataConfig
	file     string
	f        *os.File
	w        *bufio.Writer
	byteBuff bytes.Buffer
	bytePool *sync.Pool
	ch       chan *data
	timer    *time.Ticker
	end      chan bool
}

type data struct {
	msg []byte
}

func NewDataLogger(c *DataConfig) (*DataLogger, error) {
	if c.Timezone == "" {
		c.Location = time.Now().Location()
	} else {
		location, err := time.LoadLocation(c.Timezone)
		if err != nil {
			return nil, err
		}
		c.Location = location
	}

	l := &DataLogger{
		c:        *c,
		bytePool: &sync.Pool{New: func() interface{} { return new(bytes.Buffer) }},
		ch:       make(chan *data, 8192),
		timer:    time.NewTicker(time.Second),
		end:      make(chan bool, 1),
	}

	err := os.MkdirAll(l.c.Dir, 0755)

	if err != nil {
		return nil, err
	}

	l.refresh()
	l.f, err = os.OpenFile(l.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

	if err != nil {
		return nil, err
	}

	l.w = bufio.NewWriter(l.f)
	l.run()

	return l, nil
}

func (l *DataLogger) Log(args ...interface{}) {
	l.log("", args...)
}

func (l *DataLogger) Logf(format string, args ...interface{}) {
	l.log(format, args...)
}

func (l *DataLogger) Close() {
	l.timer.Stop()
	l.ch <- nil
	close(l.ch)
	<-l.end
}

func (l *DataLogger) refresh() bool {
	nowTime := time.Now().In(l.c.Location)
	year, month, day := nowTime.Date()

	var newfile string
	switch l.c.Partition {
	case PartitionHour:
		hour := nowTime.Hour()
		newfile = path.Join(l.c.Dir, fmt.Sprintf("%04d%02d%02d.%02d.log", year, month, day, hour))
	default:
		newfile = path.Join(l.c.Dir, fmt.Sprintf("%04d%02d%02d.log", year, month, day))
	}

	if l.file != newfile {
		l.file = newfile
		return true
	}
	return false
}

func (l *DataLogger) run() {
	go l.flush()
	go l.start()
}

func (l *DataLogger) start() {
	for m := range l.ch {
		if m == nil {
			l.w.Flush()

			if l.refresh() {
				l.f.Close()
				l.f, _ = os.OpenFile(l.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				l.w.Reset(l.f)
			}
		} else {
			w := &l.byteBuff
			w.Write(m.msg)
			w.WriteByte(10)
			b := w.Bytes()
			l.w.Write(b)
			l.byteBuff.Reset()
		}
	}

	l.end <- true
}

func (l *DataLogger) flush() {
	for range l.timer.C {
		l.ch <- nil
	}
}

func (l *DataLogger) log(format string, args ...interface{}) {
	w := l.bytePool.Get().(*bytes.Buffer)
	if len(format) == 0 {
		for i := 0; i < len(args); i++ {
			if i > 0 {
				w.Write([]byte{' '})
			}

			fmt.Fprint(w, args[i])
		}
	} else {
		fmt.Fprintf(w, format, args...)
	}
	b := make([]byte, w.Len())
	copy(b, w.Bytes())
	w.Reset()
	l.bytePool.Put(w)

	l.ch <- &data{msg: b}
}
