package rocketmqc

import (
	"common/tlog"
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type RocketmqConsumer struct {
	consumer rocketmq.PushConsumer
}

type FuncConsumerCallback func(ctx *primitive.ConsumeConcurrentlyContext, reconsumeTimes int32, msg *primitive.MessageExt) (delayLevel int)

func NewRocketmqConsumer(nameServers []string, group string, topic string, callback FuncConsumerCallback) (*RocketmqConsumer, error) {
	initRocketmq()

	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName(group+"-"+topic),
		consumer.WithNsResovler(primitive.NewPassthroughResolver(nameServers)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
	)
	if err != nil {
		return nil, err
	}
	err = c.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		delayLevel := 0
		concurrentCtx, _ := primitive.GetConcurrentlyCtx(ctx)
		for _, msg := range msgs {
			reconsumeTimes := msg.ReconsumeTimes
			if reconsumeTimes >= 3 {
				tlog.Errorf("msg reconsumer many times: %d, %v", reconsumeTimes, msg)
			}
			level := callback(concurrentCtx, msg.ReconsumeTimes, msg)
			if delayLevel < level {
				delayLevel = level
			}
		}
		if delayLevel == 0 {
			return consumer.ConsumeSuccess, nil
		} else {
			concurrentCtx.DelayLevelWhenNextConsume = delayLevel
			return consumer.ConsumeRetryLater, nil
		}
	})
	if err != nil {
		return nil, err
	}

	return &RocketmqConsumer{consumer: c}, nil
}

func (this *RocketmqConsumer) Start() error {
	if this.consumer != nil {
		return this.consumer.Start()
	}
	return errors.New("consumer not created")
}

func (this *RocketmqConsumer) Close() {
	if this.consumer != nil {
		this.consumer.Shutdown()
	}
}
