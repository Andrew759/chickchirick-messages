package service

import (
	"chickchirick-messages/cmd/config"
	"fmt"
	"strconv"

	"github.com/redis/go-redis/v9"
)

type RedisDecorator struct {
	Client *redis.Client
}

func InitRedis(config config.RedisConfig) *RedisDecorator {
	client := redis.NewClient(&redis.Options{
		Addr:     config.Host + ":" + strconv.Itoa(config.Port),
		Password: config.Password,
		DB:       config.Db,
	})

	redisClient := RedisDecorator{
		Client: client,
	}

	return &redisClient
}

func (rd RedisDecorator) RedisClose() {
	err := rd.Client.Close()
	if err != nil {
		panic(fmt.Errorf("redis close error: %w", err))
	}
}
