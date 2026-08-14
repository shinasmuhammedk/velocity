package redis

import (
	"context"
	"fmt"
	"velocity/internal/config"

	goredis "github.com/redis/go-redis/v9"
)

type Client struct {
	*goredis.Client
}

func New(cfg config.RedisConfig) *Client {
	addr := fmt.Sprintf(
		"%s:%d",
		cfg.Host,
		cfg.Port,
	)
    
    client := goredis.NewClient(&goredis.Options{
        Addr: addr,
        Password: cfg.Password,
        DB: cfg.Database,
    })
    
    return &Client{
        Client: client,
    }
}

func (c *Client) Ping(ctx context.Context)error{
    return c.Client.Ping(ctx).Err()
}

func (c *Client) Close()error{
    return c.Client.Close()
}
