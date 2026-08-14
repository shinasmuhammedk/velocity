package redis

import (
	"context"
	"fmt"
	"time"
)

type HealthChecker struct {
	client *Client
}

func NewHealthChecker(client *Client) *HealthChecker {
	return &HealthChecker{
		client: client,
	}
}

func (h *HealthChecker) Check(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	if err := h.client.Ping(ctx); err != nil {
		return fmt.Errorf("redis health check failed: %w", err)
	}
	return nil
}
