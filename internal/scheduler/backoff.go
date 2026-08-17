package scheduler

import (
	"context"
	"time"
)

// RetryWithBackoff 带指数退避的重试
func RetryWithBackoff(ctx context.Context, maxRetries int, initialDelay time.Duration, fn func() error) error {
	var err error
	for i := 0; i <= maxRetries; i++ {
		if i > 0 {
			delay := initialDelay * time.Duration(1<<uint(i-1))
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}
		if err = fn(); err == nil {
			return nil
		}
		// 可终止错误判断，这里简化
	}
	return err
}
