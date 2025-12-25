package workflow_session

import (
	"context"
	"time"
)

type FallbackMappingStore struct {
	Primary   MappingStore
	Secondary MappingStore
}

func NewFallbackMappingStore(primary MappingStore, secondary MappingStore) *FallbackMappingStore {
	return &FallbackMappingStore{
		Primary:   primary,
		Secondary: secondary,
	}
}

func (s *FallbackMappingStore) Get(ctx context.Context, workflowRunID string) (string, bool, error) {
	if s.Primary != nil {
		sessionID, found, err := s.Primary.Get(ctx, workflowRunID)
		if err == nil {
			return sessionID, found, nil
		}
		if s.Secondary == nil {
			return "", false, err
		}
	}
	if s.Secondary != nil {
		return s.Secondary.Get(ctx, workflowRunID)
	}
	return "", false, ErrMappingUnavailable
}

func (s *FallbackMappingStore) Set(ctx context.Context, workflowRunID string, sessionID string, ttl time.Duration) error {
	if s.Primary == nil && s.Secondary == nil {
		return ErrMappingUnavailable
	}
	if s.Primary != nil {
		if err := s.Primary.Set(ctx, workflowRunID, sessionID, ttl); err != nil {
			if s.Secondary == nil {
				return err
			}
			if err := s.Secondary.Set(ctx, workflowRunID, sessionID, ttl); err != nil {
				return err
			}
			return nil
		}
		if s.Secondary != nil {
			_ = s.Secondary.Set(ctx, workflowRunID, sessionID, ttl)
		}
		return nil
	}
	return s.Secondary.Set(ctx, workflowRunID, sessionID, ttl)
}

func (s *FallbackMappingStore) Touch(ctx context.Context, workflowRunID string, ttl time.Duration) error {
	if s.Primary == nil && s.Secondary == nil {
		return ErrMappingUnavailable
	}
	if s.Primary != nil {
		if err := s.Primary.Touch(ctx, workflowRunID, ttl); err != nil {
			if s.Secondary == nil {
				return err
			}
			if err := s.Secondary.Touch(ctx, workflowRunID, ttl); err != nil {
				return err
			}
			return nil
		}
		if s.Secondary != nil {
			_ = s.Secondary.Touch(ctx, workflowRunID, ttl)
		}
		return nil
	}
	return s.Secondary.Touch(ctx, workflowRunID, ttl)
}
