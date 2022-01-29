package dao

// 缓存的实现demo

import (
	redisutil "github.com/cclehui/redis-util"
	"github.com/gomodule/redigo/redis"
)

// 基于 redigo 的redis 缓存

type RedisConfig struct {
	Server   string `yaml:"server"`   // "xxxxx:6379"
	Password string `yaml:"password"` // "wxxxxxxx"
}

type CacheUtilDemo struct {
	*redisutil.RedisUtil
}

func NewCacheUtilDemo(redisConfig *RedisConfig) *CacheUtilDemo {
	redisPool := &redis.Pool{
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", redisConfig.Server)
			if err != nil {
				return nil, err
			}

			if _, err := c.Do("AUTH", redisConfig.Password); err != nil {
				c.Close()
				return nil, err
			}

			return c, nil
		},
	}

	return &CacheUtilDemo{RedisUtil: redisutil.NewRedisUtil(redisPool)}
}
