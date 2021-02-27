package discovery

import (
	"context"
	"strconv"
	"time"
)

const GRPC_TIMEOUT = 8 * time.Second
const GRPC_LONG_TIMEOUT = 8 * time.Second
const GRPC_SHORT_TIMEOUT = 2 * time.Second

type (
	CtxKey string
	CtxMap map[string]interface{}
)

func Context(timeout time.Duration, kv CtxMap) (context.Context, context.CancelFunc) {
	ctx := context.Background()
	for k, v := range kv {
		ctx = context.WithValue(ctx, CtxKey(k), v)
	}
	return context.WithTimeout(ctx, timeout)
}

func ContextWithIdRouting(uid int64) (context.Context, context.CancelFunc) {
	return Context(
		GRPC_TIMEOUT,
		CtxMap{"routing": strconv.FormatInt(uid, 10)},
	)
}

func ContextWithTarget(i int) (context.Context, context.CancelFunc) {
	return Context(GRPC_TIMEOUT, CtxMap{"target": i})
}

func ContextWithStandard() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), GRPC_TIMEOUT)
}

func ContextWithLongTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), GRPC_LONG_TIMEOUT)
}

func ContextWithShortTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), GRPC_SHORT_TIMEOUT)
}

//Broadcast 广播, 广播上限目前为 10 台机器
func Broadcast(f func(context.Context) error) {
	for i := 1; i <= 10; i++ {
		ctx, cancel := ContextWithTarget(i)
		err := f(ctx)
		cancel()
		if err != nil {
			break
		}
	}
}
