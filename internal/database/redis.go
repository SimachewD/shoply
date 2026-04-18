
// internal/database/redis.go
package database

import (
    "github.com/redis/go-redis/v9"
    "context"
)

func ConnectRedis(url string) (*redis.Client, error) {
    opt, err := redis.ParseURL(url)
    if err != nil {
        return nil, err
    }

    client := redis.NewClient(opt)
    if err := client.Ping(context.Background()).Err(); err != nil {
        return nil, err
    }

    return client, nil
}