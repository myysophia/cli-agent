package workflow_session

import (
	"context"
	"time"
)

type FallbackLocker struct {
	Primary   Locker
	Secondary Locker
}

func NewFallbackLocker(primary Locker, secondary Locker) *FallbackLocker {
	return &FallbackLocker{
		Primary:   primary,
		Secondary: secondary,
	}
}

func (l *FallbackLocker) TryLock(ctx context.Context, workflowRunID string, ttl time.Duration) (LockHandle, bool, error) {
	if l.Primary != nil {
		handle, ok, err := l.Primary.TryLock(ctx, workflowRunID, ttl)
		if err == nil {
			return handle, ok, nil
		}
		if l.Secondary == nil {
			return nil, false, err
		}
	}
	if l.Secondary != nil {
		return l.Secondary.TryLock(ctx, workflowRunID, ttl)
	}
	return nil, false, ErrLockUnavailable
}
