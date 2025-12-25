package workflow_session

import (
	"context"
	"fmt"
	"time"
)

const (
	defaultKeyPrefix = "workflow:run"
	mappingSuffix    = "session"
	lockSuffix       = "lock"
)

type MappingStore interface {
	Get(ctx context.Context, workflowRunID string) (sessionID string, found bool, err error)
	Set(ctx context.Context, workflowRunID string, sessionID string, ttl time.Duration) error
	Touch(ctx context.Context, workflowRunID string, ttl time.Duration) error
}

type LockHandle interface {
	Unlock(ctx context.Context) error
}

type Locker interface {
	TryLock(ctx context.Context, workflowRunID string, ttl time.Duration) (LockHandle, bool, error)
}

type Keyer interface {
	MappingKey(workflowRunID string) string
	LockKey(workflowRunID string) string
}

type DefaultKeyer struct {
	Prefix string
}

func (k DefaultKeyer) MappingKey(workflowRunID string) string {
	prefix := k.Prefix
	if prefix == "" {
		prefix = defaultKeyPrefix
	}
	return fmt.Sprintf("%s:%s:%s", prefix, workflowRunID, mappingSuffix)
}

func (k DefaultKeyer) LockKey(workflowRunID string) string {
	prefix := k.Prefix
	if prefix == "" {
		prefix = defaultKeyPrefix
	}
	return fmt.Sprintf("%s:%s:%s", prefix, workflowRunID, lockSuffix)
}

type CreateResult struct {
	SessionID string
	Payload   string
}

type SessionCreator func(ctx context.Context) (CreateResult, error)
