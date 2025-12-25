package workflow_session

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

var redisUnlockScript = redis.NewScript(`
if redis.call("get", KEYS[1]) == ARGV[1] then
	return redis.call("del", KEYS[1])
else
	return 0
end`)

type RedisLocker struct {
	client *redis.Client
	keyer  Keyer
}

func NewRedisLocker(client *redis.Client, keyer Keyer) (*RedisLocker, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if keyer == nil {
		keyer = DefaultKeyer{}
	}
	return &RedisLocker{
		client: client,
		keyer:  keyer,
	}, nil
}

func (l *RedisLocker) TryLock(ctx context.Context, workflowRunID string, ttl time.Duration) (LockHandle, bool, error) {
	if workflowRunID == "" {
		return nil, false, fmt.Errorf("workflow run id is required")
	}
	if ttl <= 0 {
		return nil, false, fmt.Errorf("ttl must be positive")
	}
	token, err := newLockToken()
	if err != nil {
		return nil, false, err
	}
	key := l.keyer.LockKey(workflowRunID)
	ok, err := l.client.SetNX(ctx, key, token, ttl).Result()
	if err != nil {
		return nil, false, err
	}
	if !ok {
		return nil, false, nil
	}
	return &redisLockHandle{
		client: l.client,
		key:    key,
		token:  token,
	}, true, nil
}

type redisLockHandle struct {
	client *redis.Client
	key    string
	token  string
}

func (h *redisLockHandle) Unlock(ctx context.Context) error {
	if h == nil {
		return nil
	}
	if h.client == nil {
		return fmt.Errorf("redis client is nil")
	}
	result, err := redisUnlockScript.Run(ctx, h.client, []string{h.key}, h.token).Result()
	if err != nil {
		return err
	}
	if deleted, ok := result.(int64); ok && deleted == 0 {
		return fmt.Errorf("lock not owned or already released")
	}
	return nil
}
