package metrics

import (
	"bytes"
	"common/defs"
	"common/tlog"
	"encoding/json"
	"net/http"
	"os"
	"time"
)

var m *Metrics

type Metrics struct {
	name     string
	m        map[string]*Counter
	ch       chan *Event
	endpoint string
	client   *http.Client
}

type Data struct {
	Metric      string  `json:"metric"`
	Tags        string  `json:"tags"`
	Endpoint    string  `json:"endpoint"`
	Value       float64 `json:"value"`
	CounterType string  `json:"counterType"`
	Timestamp   int64   `json:"timestamp"`
	Step        int     `json:"step"`
}

type Counter struct {
	Total int64 `json:"-"`
	Count int   `json:"count"`
	Avg   int64 `json:"avg"`
	Qps   int   `json:"qps"`
}

type Event struct {
	event string
	t     int64
}

func Init(name string, env string) {
	if env != defs.EnvProd {
		return
	}
	if m == nil {
		m = &Metrics{
			name: name,
			m:    make(map[string]*Counter),
			ch:   make(chan *Event, 10240),
			client: &http.Client{
				Timeout: time.Second,
				Transport: &http.Transport{
					MaxIdleConns:          2,
					IdleConnTimeout:       120 * time.Second,
					DisableCompression:    true,
					ResponseHeaderTimeout: 3 * time.Second,
				},
			},
		}
		m.endpoint, _ = os.Hostname()
		go m.run()
		go m.t()
	}
}

func Add(event string, t time.Time) {
	if m == nil {
		return
	}
	select {
	case m.ch <- &Event{event: event, t: int64(time.Since(t) / 1000)}:
	default:
	}
}

//t 微秒数
func AddT(event string, t int64) {
	if m == nil {
		return
	}
	select {
	case m.ch <- &Event{event: event, t: t}:
	default:
	}
}

func (m *Metrics) run() {
	for {
		e := <-m.ch
		if e == nil {
			m.p()
			continue
		}
		counter, ok := m.m[e.event]
		if ok {
			counter.Total += e.t
			counter.Count++
		} else {
			m.m[e.event] = &Counter{
				Total: e.t,
				Count: 1,
			}
		}
	}
}

func (m *Metrics) t() {
	for range time.NewTicker(60 * time.Second).C {
		m.ch <- nil
	}
}

func (m *Metrics) p() {
	for key, c := range m.m {
		if c.Count == 0 {
			delete(m.m, key)
			continue
		}
		c.Qps = c.Count
		c.Avg = c.Total / int64(c.Count)

	}
	if len(m.m) > 0 {
		ds := []*Data{}
		for key, c := range m.m {
			ds = append(ds, &Data{
				Metric:      m.name,
				Tags:        "type=" + key,
				Endpoint:    m.endpoint,
				Value:       float64(c.Qps),
				CounterType: "GAUGE",
				Timestamp:   time.Now().Unix(),
				Step:        60})
			ds = append(ds, &Data{
				Metric:      m.name,
				Tags:        "type=" + key + ".responsetime",
				Endpoint:    m.endpoint,
				Value:       float64(c.Avg),
				CounterType: "GAUGE",
				Timestamp:   time.Now().Unix(),
				Step:        60})
		}
		bb, _ := json.Marshal(ds)
		go report(bb)
		m.m = make(map[string]*Counter)
	}
}

func report(d []byte) {
	req, _ := http.NewRequest("POST", "http://localhost:1988/v1/push", bytes.NewBuffer(d))
	req.Header.Add("Content-Type", "application/json")
	resp, err := m.client.Do(req)
	if err != nil {
		tlog.Errorf("falconError||error=%s", err.Error())
	} else {
		resp.Body.Close()
	}
}
