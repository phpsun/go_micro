package rocketmqc

import (
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

// reference delay level definition: 1s 5s 10s 30s 1m 2m 3m 4m 5m 6m 7m 8m 9m 10m 20m 30m 1h 2h
// delay level starts from 1. for example, if we set param level=1, then the delay time is 1s.
const (
	DalayLevel1s  = 1
	DalayLevel5s  = 2
	DalayLevel10s = 3
	DalayLevel30s = 4
	DalayLevel1m  = 5
	DalayLevel2m  = 6
	DalayLevel3m  = 7
	DalayLevel4m  = 8
	DalayLevel5m  = 9
	DalayLevel6m  = 10
	DalayLevel7m  = 11
	DalayLevel8m  = 12
	DalayLevel9m  = 13
	DalayLevel10m = 14
	DalayLevel20m = 15
	DalayLevel30m = 16
	DalayLevel1h  = 17
	DalayLevel2h  = 18
)

type RocketmqProducer struct {
	producer rocketmq.Producer
}

func NewRocketmqProducer(nameServers []string) (*RocketmqProducer, error) {
	initRocketmq()

	p, err := rocketmq.NewProducer(
		producer.WithNsResovler(primitive.NewPassthroughResolver(nameServers)),
		producer.WithRetry(2),
	)
	if err != nil {
		return nil, err
	}

	err = p.Start()
	if err != nil {
		return nil, err
	}

	return &RocketmqProducer{producer: p}, nil
}

func (this *RocketmqProducer) Close() {
	if this.producer != nil {
		this.producer.Shutdown()
	}
}

func (this *RocketmqProducer) SendSync(topic string, data []byte, key string) (*primitive.SendResult, error) {
	if this.producer != nil {
		msg := primitive.NewMessage(topic, data)
		if key != "" {
			msg.WithKeys([]string{key})
		}
		return this.producer.SendSync(context.Background(), msg)
	}
	return nil, errors.New("Rocketmq producer not created")
}

func (this *RocketmqProducer) SendSyncWithDelay(topic string, data []byte, key string, delayLevel int) (*primitive.SendResult, error) {
	if this.producer != nil {
		msg := primitive.NewMessage(topic, data)
		msg.WithDelayTimeLevel(delayLevel)
		if key != "" {
			msg.WithKeys([]string{key})
		}
		return this.producer.SendSync(context.Background(), msg)
	}
	return nil, errors.New("Rocketmq producer not created")
}
