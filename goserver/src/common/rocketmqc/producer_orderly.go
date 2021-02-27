package rocketmqc

import (
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/primitive"
	"github.com/apache/rocketmq-client-go/v2/producer"
)

type RocketmqOrderlyProducer struct {
	producer rocketmq.Producer
}

func NewRocketmqOrderlyProducer(nameServers []string) (*RocketmqOrderlyProducer, error) {
	initRocketmq()

	p, err := rocketmq.NewProducer(
		producer.WithNsResovler(primitive.NewPassthroughResolver(nameServers)),
		producer.WithRetry(2),
		producer.WithQueueSelector(producer.NewHashQueueSelector()),
	)
	if err != nil {
		return nil, err
	}

	err = p.Start()
	if err != nil {
		return nil, err
	}

	return &RocketmqOrderlyProducer{producer: p}, nil
}

func (this *RocketmqOrderlyProducer) Close() {
	if this.producer != nil {
		this.producer.Shutdown()
	}
}

func (this *RocketmqOrderlyProducer) SendSync(topic string, data []byte, shardingKey string) (*primitive.SendResult, error) {
	if this.producer != nil {
		msg := primitive.NewMessage(topic, data)
		msg.WithShardingKey(shardingKey)
		return this.producer.SendSync(context.Background(), msg)
	}
	err := errors.New("Rocketmq orderly producer not created")
	return nil, err
}
