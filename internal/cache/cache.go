package cache

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/vmihailenco/msgpack"
)
import rediscache "github.com/go-redis/cache"
import "github.com/go-redis/redis"

var cacheGets = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "starcharts",
		Subsystem: "cache",
		Name:      "gets_total",
		Help:      "Total number of successful cache puts",
	},
)

var cachePuts = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "starcharts",
		Subsystem: "cache",
		Name:      "puts_total",
		Help:      "Total number of successful cache puts",
	},
)

var cacheDeletes = prometheus.NewCounter(
	prometheus.CounterOpts{
		Namespace: "starcharts",
		Subsystem: "cache",
		Name:      "deletes_total",
		Help:      "Total number of successful cache deletes",
	},
)

func init() {
	prometheus.MustRegister(cacheGets, cachePuts, cacheDeletes)
}

type Redis struct {
	redis *redis.Client
	codec *rediscache.Codec
}

func New(redis *redis.Client) *Redis {
	//rediscache.Codec is used for encoding and decoding Redis cached data
	codec := &rediscache.Codec{Redis: redis, //redis client instance
		Marshal: func(v interface{}) ([]byte, error) {
			return msgpack.Marshal(v) //serialization function
		},
		Unmarshal: func(b []byte, v interface{}) error {
			return msgpack.Unmarshal(b, v)
		}, //deserialization function
	}
	return &Redis{
		redis: redis,
		codec: codec,
	}
}

func (c *Redis) Close() error {
	return c.redis.Close()
}

func (c *Redis) Get(key string, result interface{}) error {
	if err := c.codec.Get(key, result); err != nil {
		return nil
	}
	cacheGets.Inc()
	return nil
}

func (c *Redis) Put(key string, obj interface{}) error {
	if err := c.codec.Set(&rediscache.Item{
		Key:    key,
		Object: obj,
	}); err != nil {
		return err
	}
	cachePuts.Inc()
	return nil
}

func (c *Redis) Delete(key string) error {
	if err := c.codec.Delete(key); err != nil {
		return err
	}
	cacheDeletes.Inc()
	return nil
}
