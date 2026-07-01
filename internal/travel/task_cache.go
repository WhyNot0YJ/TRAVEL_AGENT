package travel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

const taskResultCacheVersion = 1

type TaskCache interface {
	SetRequestHash(ctx context.Context, requestHash, taskID string) error
	GetTaskIDByHash(ctx context.Context, requestHash string) (string, bool, error)
	SetTask(ctx context.Context, task Task) error
	GetTask(ctx context.Context, taskID string) (Task, bool, error)
}

type CachedTaskStore struct {
	authority TaskStore
	cache     TaskCache
}

func NewCachedTaskStore(authority TaskStore, cache TaskCache) *CachedTaskStore {
	return &CachedTaskStore{authority: authority, cache: cache}
}

func (s *CachedTaskStore) Create(ctx context.Context, task Task) error {
	if err := s.authority.Create(ctx, task); err != nil {
		return err
	}
	s.cacheTask(ctx, task)
	return nil
}

func (s *CachedTaskStore) Update(ctx context.Context, task Task) error {
	if err := s.authority.Update(ctx, task); err != nil {
		return err
	}
	s.cacheTask(ctx, task)
	return nil
}

func (s *CachedTaskStore) Get(ctx context.Context, id string) (Task, error) {
	if s.cache != nil {
		task, ok, err := s.cache.GetTask(ctx, id)
		if err == nil && ok && isTerminalTaskStatus(task.Status) {
			return task, nil
		}
		if err != nil {
			log.Printf("task cache get failed task_id=%s: %v", id, err)
		}
	}
	task, err := s.authority.Get(ctx, id)
	if err != nil {
		return Task{}, err
	}
	s.cacheTask(ctx, task)
	return task, nil
}

func (s *CachedTaskStore) FindByHash(ctx context.Context, requestHash string) (Task, bool, error) {
	if s.cache != nil {
		if taskID, ok, err := s.cache.GetTaskIDByHash(ctx, requestHash); err == nil && ok {
			task, err := s.authority.Get(ctx, taskID)
			if err == nil {
				s.cacheTask(ctx, task)
				return task, true, nil
			}
			if !errors.Is(err, ErrTaskNotFound) {
				return Task{}, false, err
			}
		} else if err != nil {
			log.Printf("request hash cache lookup failed hash=%s: %v", requestHash, err)
		}
	}
	task, ok, err := s.authority.FindByHash(ctx, requestHash)
	if err != nil || !ok {
		return task, ok, err
	}
	s.cacheTask(ctx, task)
	return task, true, nil
}

func (s *CachedTaskStore) cacheTask(ctx context.Context, task Task) {
	if s == nil || s.cache == nil {
		return
	}
	if task.RequestHash != "" {
		if err := s.cache.SetRequestHash(ctx, task.RequestHash, task.ID); err != nil {
			log.Printf("request hash cache set failed task_id=%s: %v", task.ID, err)
		}
	}
	if isTerminalTaskStatus(task.Status) {
		if err := s.cache.SetTask(ctx, task); err != nil {
			log.Printf("task result cache set failed task_id=%s: %v", task.ID, err)
		}
	}
}

func isTerminalTaskStatus(status TaskStatus) bool {
	switch status {
	case TaskSucceeded, TaskFailed, TaskCanceled, TaskDeadLetter:
		return true
	default:
		return false
	}
}

type RedisTaskCache struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisTaskCache(client *goredis.Client, ttl time.Duration) *RedisTaskCache {
	return &RedisTaskCache{client: client, ttl: ttl}
}

func (c *RedisTaskCache) SetRequestHash(ctx context.Context, requestHash, taskID string) error {
	if c == nil || c.client == nil || requestHash == "" || taskID == "" {
		return nil
	}
	return c.client.Set(ctx, hashKey(requestHash), taskID, c.ttl).Err()
}

func (c *RedisTaskCache) GetTaskIDByHash(ctx context.Context, requestHash string) (string, bool, error) {
	if c == nil || c.client == nil || requestHash == "" {
		return "", false, nil
	}
	taskID, err := c.client.Get(ctx, hashKey(requestHash)).Result()
	if errors.Is(err, goredis.Nil) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return taskID, true, nil
}

func (c *RedisTaskCache) SetTask(ctx context.Context, task Task) error {
	if c == nil || c.client == nil || task.ID == "" {
		return nil
	}
	payload := cachedTaskPayload{
		Version: taskResultCacheVersion,
		Task:    task,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal task cache: %w", err)
	}
	return c.client.Set(ctx, taskKey(task.ID), data, c.ttl).Err()
}

func (c *RedisTaskCache) GetTask(ctx context.Context, taskID string) (Task, bool, error) {
	if c == nil || c.client == nil || taskID == "" {
		return Task{}, false, nil
	}
	data, err := c.client.Get(ctx, taskKey(taskID)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}
	var payload cachedTaskPayload
	if err := json.Unmarshal(data, &payload); err == nil && payload.Version == taskResultCacheVersion {
		return payload.Task, true, nil
	}
	var legacy Task
	if err := json.Unmarshal(data, &legacy); err != nil {
		return Task{}, false, fmt.Errorf("decode task cache: %w", err)
	}
	return legacy, true, nil
}

type cachedTaskPayload struct {
	Version int  `json:"version"`
	Task    Task `json:"task"`
}

type MemoryTaskCache struct {
	mu       sync.RWMutex
	ttl      time.Duration
	tasks    map[string]memoryTaskCacheEntry
	hashToID map[string]memoryStringCacheEntry
}

type memoryTaskCacheEntry struct {
	Task      Task
	ExpiresAt time.Time
}

type memoryStringCacheEntry struct {
	Value     string
	ExpiresAt time.Time
}

func NewMemoryTaskCache(ttl time.Duration) *MemoryTaskCache {
	return &MemoryTaskCache{
		ttl:      ttl,
		tasks:    map[string]memoryTaskCacheEntry{},
		hashToID: map[string]memoryStringCacheEntry{},
	}
}

func (c *MemoryTaskCache) SetRequestHash(ctx context.Context, requestHash, taskID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if requestHash == "" || taskID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.hashToID[requestHash] = memoryStringCacheEntry{Value: taskID, ExpiresAt: c.expiresAt()}
	return nil
}

func (c *MemoryTaskCache) GetTaskIDByHash(ctx context.Context, requestHash string) (string, bool, error) {
	if err := ctx.Err(); err != nil {
		return "", false, err
	}
	c.mu.RLock()
	entry, ok := c.hashToID[requestHash]
	c.mu.RUnlock()
	if !ok || entry.expired(time.Now()) {
		return "", false, nil
	}
	return entry.Value, true, nil
}

func (c *MemoryTaskCache) SetTask(ctx context.Context, task Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if task.ID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tasks[task.ID] = memoryTaskCacheEntry{Task: task, ExpiresAt: c.expiresAt()}
	return nil
}

func (c *MemoryTaskCache) GetTask(ctx context.Context, taskID string) (Task, bool, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, false, err
	}
	c.mu.RLock()
	entry, ok := c.tasks[taskID]
	c.mu.RUnlock()
	if !ok || entry.expired(time.Now()) {
		return Task{}, false, nil
	}
	return entry.Task, true, nil
}

func (c *MemoryTaskCache) expiresAt() time.Time {
	if c.ttl <= 0 {
		return time.Time{}
	}
	return time.Now().Add(c.ttl)
}

func (entry memoryTaskCacheEntry) expired(now time.Time) bool {
	return !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt)
}

func (entry memoryStringCacheEntry) expired(now time.Time) bool {
	return !entry.ExpiresAt.IsZero() && now.After(entry.ExpiresAt)
}
