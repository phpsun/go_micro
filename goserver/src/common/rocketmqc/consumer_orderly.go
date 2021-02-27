package rocketmqc

import (
	"common/tlog"
	"context"
	"errors"
	"github.com/apache/rocketmq-client-go/v2"
	"github.com/apache/rocketmq-client-go/v2/consumer"
	"github.com/apache/rocketmq-client-go/v2/primitive"
)

type RocketmqOrderlyConsumer struct {
	consumer rocketmq.PushConsumer
}

type FuncOrderlyConsumerCallback func(ctx *primitive.ConsumeOrderlyContext, reconsumeTimes int32, msg *primitive.MessageExt) bool

func NewRocketmqOrderlyConsumer(nameServers []string, group string, topic string, callback FuncOrderlyConsumerCallback) (*RocketmqOrderlyConsumer, error) {
	initRocketmq()

	c, err := rocketmq.NewPushConsumer(
		consumer.WithGroupName(group+"-"+topic),
		consumer.WithNsResovler(primitive.NewPassthroughResolver(nameServers)),
		consumer.WithConsumerModel(consumer.Clustering),
		consumer.WithConsumeFromWhere(consumer.ConsumeFromFirstOffset),
		consumer.WithConsumerOrder(true),
	)
	if err != nil {
		return nil, err
	}
	err = c.Subscribe(topic, consumer.MessageSelector{}, func(ctx context.Context,
		msgs ...*primitive.MessageExt) (consumer.ConsumeResult, error) {
		bRet := true
		orderlyCtx, _ := primitive.GetOrderlyCtx(ctx)
		for _, msg := range msgs {
			reconsumeTimes := msg.ReconsumeTimes
			if reconsumeTimes >= 3 {
				tlog.Errorf("msg reconsumer many times: %d, %v", reconsumeTimes, msg)
			}
			if !callback(orderlyCtx, msg.ReconsumeTimes, msg) {
				bRet = false
			}
		}
		if bRet {
			return consumer.ConsumeSuccess, nil
		} else {
			return consumer.SuspendCurrentQueueAMoment, nil
		}
	})
	if err != nil {
		return nil, err
	}

	return &RocketmqOrderlyConsumer{consumer: c}, nil
}

func (this *RocketmqOrderlyConsumer) Start() error {
	if this.consumer != nil {
		return this.consumer.Start()
	}
	return errors.New("consumer not created")
}

func (this *RocketmqOrderlyConsumer) Close() {
	if this.consumer != nil {
		this.consumer.Shutdown()
	}
}
