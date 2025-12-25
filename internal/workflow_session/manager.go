package workflow_session

import (
	"context"
	"fmt"
	"log"
	"time"
)

const (
	defaultMappingTTL        = 24 * time.Hour
	defaultLockTTL           = 2 * time.Minute
	defaultLockWaitTimeout   = 2 * time.Minute
	defaultLockRetryInterval = 200 * time.Millisecond
)

type ManagerConfig struct {
	MappingTTL        time.Duration
	LockTTL           time.Duration
	LockWaitTimeout   time.Duration
	LockRetryInterval time.Duration
}

type Manager struct {
	store             MappingStore
	locker            Locker
	mappingTTL        time.Duration
	lockTTL           time.Duration
	lockWaitTimeout   time.Duration
	lockRetryInterval time.Duration
}

func NewManager(cfg ManagerConfig, store MappingStore, locker Locker) (*Manager, error) {
	if store == nil {
		return nil, fmt.Errorf("mapping store is required")
	}
	if locker == nil {
		return nil, fmt.Errorf("locker is required")
	}
	if cfg.MappingTTL <= 0 {
		cfg.MappingTTL = defaultMappingTTL
	}
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = defaultLockTTL
	}
	if cfg.LockWaitTimeout <= 0 {
		cfg.LockWaitTimeout = defaultLockWaitTimeout
	}
	if cfg.LockRetryInterval <= 0 {
		cfg.LockRetryInterval = defaultLockRetryInterval
	}
	return &Manager{
		store:             store,
		locker:            locker,
		mappingTTL:        cfg.MappingTTL,
		lockTTL:           cfg.LockTTL,
		lockWaitTimeout:   cfg.LockWaitTimeout,
		lockRetryInterval: cfg.LockRetryInterval,
	}, nil
}

func (m *Manager) Get(ctx context.Context, workflowRunID string) (string, bool, error) {
	if workflowRunID == "" {
		return "", false, fmt.Errorf("workflow run id is required")
	}
	sessionID, found, err := m.store.Get(ctx, workflowRunID)
	if err != nil {
		return "", false, err
	}
	if found {
		if err := m.store.Touch(ctx, workflowRunID, m.mappingTTL); err != nil {
			log.Printf("⚠️  Workflow mapping touch failed: %v", err)
		}
	}
	return sessionID, found, nil
}

func (m *Manager) GetOrCreate(ctx context.Context, workflowRunID string, creator SessionCreator) (CreateResult, bool, error) {
	if workflowRunID == "" {
		return CreateResult{}, false, fmt.Errorf("workflow run id is required")
	}
	if creator == nil {
		return CreateResult{}, false, fmt.Errorf("session creator is required")
	}

	sessionID, found, err := m.store.Get(ctx, workflowRunID)
	if err != nil {
		return CreateResult{}, false, err
	}
	if found {
		if err := m.store.Touch(ctx, workflowRunID, m.mappingTTL); err != nil {
			log.Printf("⚠️  Workflow mapping touch failed: %v", err)
		}
		return CreateResult{SessionID: sessionID}, false, nil
	}

	lockHandle, ok, err := m.locker.TryLock(ctx, workflowRunID, m.lockTTL)
	if err != nil {
		return CreateResult{}, false, err
	}
	if ok {
		return m.createWithLock(ctx, workflowRunID, lockHandle, creator)
	}

	if err := m.waitForMapping(ctx, workflowRunID); err != nil {
		return CreateResult{}, false, err
	}

	sessionID, found, err = m.store.Get(ctx, workflowRunID)
	if err != nil {
		return CreateResult{}, false, err
	}
	if found {
		if err := m.store.Touch(ctx, workflowRunID, m.mappingTTL); err != nil {
			log.Printf("⚠️  Workflow mapping touch failed: %v", err)
		}
		return CreateResult{SessionID: sessionID}, false, nil
	}

	log.Printf("⚠️  Workflow mapping not found after wait, retrying lock")
	lockHandle, ok, err = m.locker.TryLock(ctx, workflowRunID, m.lockTTL)
	if err != nil {
		return CreateResult{}, false, err
	}
	if !ok {
		return CreateResult{}, false, ErrLockTimeout
	}
	return m.createWithLock(ctx, workflowRunID, lockHandle, creator)
}

func (m *Manager) waitForMapping(ctx context.Context, workflowRunID string) error {
	if m.lockWaitTimeout <= 0 {
		return nil
	}
	deadline := time.Now().Add(m.lockWaitTimeout)
	for time.Now().Before(deadline) {
		if err := sleepWithContext(ctx, m.lockRetryInterval); err != nil {
			return err
		}
		_, found, err := m.store.Get(ctx, workflowRunID)
		if err != nil {
			return err
		}
		if found {
			if err := m.store.Touch(ctx, workflowRunID, m.mappingTTL); err != nil {
				log.Printf("⚠️  Workflow mapping touch failed: %v", err)
			}
			return nil
		}
	}
	return nil
}

func (m *Manager) createWithLock(ctx context.Context, workflowRunID string, lockHandle LockHandle, creator SessionCreator) (CreateResult, bool, error) {
	defer func() {
		if err := lockHandle.Unlock(ctx); err != nil {
			log.Printf("⚠️  Workflow lock release failed: %v", err)
		}
	}()

	sessionID, found, err := m.store.Get(ctx, workflowRunID)
	if err != nil {
		return CreateResult{}, false, err
	}
	if found {
		if err := m.store.Touch(ctx, workflowRunID, m.mappingTTL); err != nil {
			log.Printf("⚠️  Workflow mapping touch failed: %v", err)
		}
		return CreateResult{SessionID: sessionID}, false, nil
	}

	result, err := creator(ctx)
	if err != nil {
		return CreateResult{}, false, err
	}
	if result.SessionID == "" {
		return CreateResult{}, false, fmt.Errorf("session id is required from creator")
	}
	if err := m.store.Set(ctx, workflowRunID, result.SessionID, m.mappingTTL); err != nil {
		return CreateResult{}, false, err
	}
	return result, true, nil
}

func sleepWithContext(ctx context.Context, duration time.Duration) error {
	if duration <= 0 {
		return nil
	}
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
