package database

import (
	"context"
	"fmt"

	"github.com/redis/go-redis/v9"
)

func InitRedis(host string, port int, password string, db int) (*redis.Client, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     fmt.Sprintf("%s:%d", host, port),
		Password: password,
		DB:       db,
	})

	ctx := context.Background()
	if _, err := rdb.Ping(ctx).Result(); err != nil {
		return nil, fmt.Errorf("连接Redis失败: %w", err)
	}

	return rdb, nil
}
