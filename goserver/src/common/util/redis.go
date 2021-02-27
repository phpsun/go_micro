package util

import (
	"common/tlog"
	"errors"
	"net"
	"strconv"
	"time"

	"github.com/go-redis/redis"
)

type RedisConfig struct {
	Addrs    []string `toml:"addrs"`
	Pwd      string   `toml:"pwd"`
	PoolSize int      `toml:"pool_size"`
}

func NewRedisClient(c *RedisConfig) (*redis.Client, error) {
	return NewRedisClientWithTimeout(c, time.Second, 0)
}

func NewRedisClientWithDb(c *RedisConfig, db int) (*redis.Client, error) {
	return NewRedisClientWithTimeout(c, time.Second, db)
}

func NewRedisClientWithTimeout(c *RedisConfig, timeout time.Duration, db int) (*redis.Client, error) {
	redisNum := len(c.Addrs)
	if redisNum == 0 {
		return nil, errors.New("redis addrs is empty")
	}
	ch := make(chan []string, redisNum)
	for i := 0; i < redisNum; i++ {
		list := make([]string, redisNum)
		for j := 0; j < redisNum; j++ {
			list[j] = c.Addrs[(i+j)%redisNum]
		}
		ch <- list
	}
	options := &redis.Options{
		Password:     c.Pwd,
		PoolSize:     c.PoolSize,
		DB:           db,
		ReadTimeout:  timeout,
		WriteTimeout: timeout,
		PoolTimeout:  timeout,
		IdleTimeout:  60 * time.Second,
		Dialer: func() (net.Conn, error) {
			list := <-ch
			ch <- list
			for _, addr := range list {
				c, err := net.DialTimeout("tcp", addr, 1000*time.Millisecond)
				if err == nil {
					return c, nil
				}
			}
			return nil, errors.New("all redis down")
		},
	}
	return redis.NewClient(options), nil
}

func RedisSet(client *redis.Client, key string, value interface{}, expiration time.Duration) {
	_, err := client.Set(key, value, expiration).Result()
	if err != nil {
		tlog.Error(err, key)
	}
}

func RedisSetNX(client *redis.Client, key string, value interface{}, expiration time.Duration) bool {
	flag, err := client.SetNX(key, value, expiration).Result()
	if err != nil {
		tlog.Error(err, key)
	}
	return flag
}

func RedisDelKey(client *redis.Client, key string) bool {
	flag, err := client.Del(key).Result()
	if err != nil {
		tlog.Error(err, key)
	}
	if flag == 1 {
		return true
	}
	return false
}

func RedisGet(client *redis.Client, key string) string {
	str, err := client.Get(key).Result()
	if err != nil {
		if err != redis.Nil {
			tlog.Error(err, key)
		}
		return ""
	}
	return str
}

func RedisGetInt64(client *redis.Client, key string, defaultVal int64) int64 {
	str, err := client.Get(key).Result()
	if err != nil {
		if err != redis.Nil {
			tlog.Error(err, key)
		}
		return defaultVal
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

func RedisHGet(client *redis.Client, key string, field string) string {
	str, err := client.HGet(key, field).Result()
	if err != nil {
		if err != redis.Nil {
			tlog.Error(err, key)
		}
		return ""
	}
	return str
}

func RedisHGetInt64(client *redis.Client, key string, field string, defaultVal int64) int64 {
	str, err := client.HGet(key, field).Result()
	if err != nil {
		if err != redis.Nil {
			tlog.Error(err, key)
		}
		return defaultVal
	}
	n, err := strconv.ParseInt(str, 10, 64)
	if err != nil {
		return defaultVal
	}
	return n
}

func RedisHIncrBy(client *redis.Client, key string, field string, incr int64) int64 {
	n, err := client.HIncrBy(key, field, incr).Result()
	if err != nil {
		tlog.Error(err, key)
		return 0
	}
	return n
}

func RedisZCard(client *redis.Client, key string) int64 {
	n, err := client.ZCard(key).Result()
	if err != nil {
		if err != redis.Nil {
			tlog.Error(err, key)
		}
		return 0
	}
	return n
}


func RedisZRangeWithScores(client *redis.Client, key string, start, stop int64) ([]redis.Z, error) {

	n, err := client.ZRangeWithScores(key, start, stop).Result()
	if err != nil {
		tlog.Error(err, key)
		return nil, err
	}
	return n, err
}

func RedisZRevRangeWithScores(client *redis.Client, key string, start, stop int64) ([]redis.Z, error) {

	n, err := client.ZRevRangeWithScores(key, start, stop).Result()
	if err != nil {
		tlog.Error(err, key)
		return nil, err
	}
	return n, err
}

func RedisZRevRange(client *redis.Client, key string, start, stop int64) ([]string, error) {
	n, err := client.ZRevRange(key, start, stop).Result()
	if err != nil {
		tlog.Error(err, key)
		return nil, err
	}
	return n, err
}

func RedisHMGet(client *redis.Client, key string, fields ...string) (list []interface{}, err error) {
	list, err = client.HMGet(key, fields...).Result()
	if len(fields) != len(list) {
		return
	}
	return
}

func RedisZRangeByScoreWithScores(client *redis.Client, key string, start, stop int64) ([]redis.Z, error) {
	zRange := redis.ZRangeBy{Offset: start, Count: stop}
	n, err := client.ZRangeByScoreWithScores(key, zRange).Result()
	if err != nil {
		tlog.Error(err, key)
		return nil, err
	}
	return n, err
}

func RedisZAdd(client *redis.Client, key string, val interface{}, score float64) (int64, error) {
	member := redis.Z{Member: val, Score: score}
	n, err := client.ZAdd(key, member).Result()
	if err != nil {
		tlog.Error(err, key)
		return 0, err
	}
	return n, err
}

func RedisHSet(client *redis.Client, key, field string, val interface{}) (bool, error) {
	return client.HSet(key, field, val).Result()
}

func RedisHGetAll(client *redis.Client, key string) (map[string]string, error) {
	return client.HGetAll(key).Result()
}