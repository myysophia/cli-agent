package workflow_session

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RedisMappingStore struct {
	client *redis.Client
	keyer  Keyer
}

func NewRedisMappingStore(client *redis.Client, keyer Keyer) (*RedisMappingStore, error) {
	if client == nil {
		return nil, fmt.Errorf("redis client is nil")
	}
	if keyer == nil {
		keyer = DefaultKeyer{}
	}
	return &RedisMappingStore{
		client: client,
		keyer:  keyer,
	}, nil
}

func (s *RedisMappingStore) Get(ctx context.Context, workflowRunID string) (string, bool, error) {
	if workflowRunID == "" {
		return "", false, fmt.Errorf("workflow run id is required")
	}
	key := s.keyer.MappingKey(workflowRunID)
	value, err := s.client.Get(ctx, key).Result()
	if err == redis.Nil {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return value, true, nil
}

func (s *RedisMappingStore) Set(ctx context.Context, workflowRunID string, sessionID string, ttl time.Duration) error {
	if workflowRunID == "" {
		return fmt.Errorf("workflow run id is required")
	}
	if sessionID == "" {
		return fmt.Errorf("session id is required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	key := s.keyer.MappingKey(workflowRunID)
	return s.client.Set(ctx, key, sessionID, ttl).Err()
}

func (s *RedisMappingStore) Touch(ctx context.Context, workflowRunID string, ttl time.Duration) error {
	if workflowRunID == "" {
		return fmt.Errorf("workflow run id is required")
	}
	if ttl <= 0 {
		return fmt.Errorf("ttl must be positive")
	}
	key := s.keyer.MappingKey(workflowRunID)
	ok, err := s.client.Expire(ctx, key, ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	return nil
}
