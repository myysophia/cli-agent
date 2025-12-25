package handler

import (
	"context"
	"log"
	"time"

	"dify-cli-gateway/internal/workflow_session"
)

var workflowSessionManager *workflow_session.Manager

func InitWorkflowSessionManager() {
	cfg := GetWorkflowSessionConfig()

	managerCfg := workflow_session.ManagerConfig{
		MappingTTL:        time.Duration(cfg.MappingTTLMinutes) * time.Minute,
		LockTTL:           time.Duration(cfg.LockTTLMS) * time.Millisecond,
		LockWaitTimeout:   time.Duration(cfg.LockWaitTimeoutMS) * time.Millisecond,
		LockRetryInterval: time.Duration(cfg.LockRetryIntervalMS) * time.Millisecond,
	}
	if cfg.LockTTLMS > 0 && cfg.LockWaitTimeoutMS > 0 && cfg.LockTTLMS < cfg.LockWaitTimeoutMS {
		log.Printf("⚠️  workflow_session.lock_ttl_ms should be >= lock_wait_timeout_ms to avoid premature lock expiry")
	}

	memoryStore := workflow_session.NewMemoryMappingStore()
	memoryLocker := workflow_session.NewMemoryLocker()

	var store workflow_session.MappingStore
	var locker workflow_session.Locker

	if cfg.Redis != nil && cfg.Redis.Addr != "" {
		redisCfg := workflow_session.RedisConfig{
			Addr:         cfg.Redis.Addr,
			Username:     cfg.Redis.Username,
			Password:     cfg.Redis.Password,
			DB:           cfg.Redis.DB,
			DialTimeout:  time.Duration(cfg.Redis.DialTimeoutMS) * time.Millisecond,
			ReadTimeout:  time.Duration(cfg.Redis.ReadTimeoutMS) * time.Millisecond,
			WriteTimeout: time.Duration(cfg.Redis.WriteTimeoutMS) * time.Millisecond,
			PoolSize:     cfg.Redis.PoolSize,
		}

		timeout := time.Duration(cfg.Redis.DialTimeoutMS) * time.Millisecond
		if timeout <= 0 {
			timeout = 5 * time.Second
		}
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		client, err := workflow_session.NewRedisClient(ctx, redisCfg)
		if err != nil {
			log.Printf("⚠️  Redis unavailable for workflow sessions, fallback to memory store: %v", err)
		} else {
			keyer := workflow_session.DefaultKeyer{}
			redisStore, err := workflow_session.NewRedisMappingStore(client, keyer)
			if err != nil {
				log.Printf("⚠️  Redis mapping store init failed, fallback to memory store: %v", err)
			} else {
				store = workflow_session.NewFallbackMappingStore(redisStore, memoryStore)
			}
			redisLocker, err := workflow_session.NewRedisLocker(client, keyer)
			if err != nil {
				log.Printf("⚠️  Redis locker init failed, fallback to memory lock: %v", err)
			} else {
				locker = workflow_session.NewFallbackLocker(redisLocker, memoryLocker)
			}
		}
	}

	if store == nil {
		store = memoryStore
	}
	if locker == nil {
		locker = memoryLocker
	}

	manager, err := workflow_session.NewManager(managerCfg, store, locker)
	if err != nil {
		log.Printf("⚠️  Workflow session manager init failed, fallback to memory: %v", err)
		manager, err = workflow_session.NewManager(managerCfg, memoryStore, memoryLocker)
		if err != nil {
			log.Printf("❌ Workflow session manager unavailable: %v", err)
			return
		}
	}

	workflowSessionManager = manager
	log.Printf("✅ Workflow session manager initialized")
}

func getWorkflowSessionManager() *workflow_session.Manager {
	if workflowSessionManager == nil {
		InitWorkflowSessionManager()
	}
	return workflowSessionManager
}
