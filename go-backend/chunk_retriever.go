package redisClient

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func Connect() *redis.Client {
	ctx := context.Background()
	// Connecting to the redis database
	rdb := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	pong, err := rdb.Ping(ctx).Result()
	if err != nil {
		panic(err)
	}
	fmt.Println("Redis ping:", pong)
	return rdb
}
