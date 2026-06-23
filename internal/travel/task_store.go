package travel

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var ErrTaskNotFound = errors.New("task not found")

type TaskStore interface {
	Create(ctx context.Context, task Task) error
	Update(ctx context.Context, task Task) error
	Get(ctx context.Context, id string) (Task, error)
	FindByHash(ctx context.Context, requestHash string) (Task, bool, error)
}

type MemoryTaskStore struct {
	mu       sync.RWMutex
	tasks    map[string]Task
	hashToID map[string]string
}

func NewMemoryTaskStore() *MemoryTaskStore {
	return &MemoryTaskStore{
		tasks:    map[string]Task{},
		hashToID: map[string]string{},
	}
}

func (s *MemoryTaskStore) Create(ctx context.Context, task Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	s.hashToID[task.RequestHash] = task.ID
	return nil
}

func (s *MemoryTaskStore) Update(ctx context.Context, task Task) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.tasks[task.ID]; !ok {
		return ErrTaskNotFound
	}
	s.tasks[task.ID] = task
	s.hashToID[task.RequestHash] = task.ID
	return nil
}

func (s *MemoryTaskStore) Get(ctx context.Context, id string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	task, ok := s.tasks[id]
	if !ok {
		return Task{}, ErrTaskNotFound
	}
	return task, nil
}

func (s *MemoryTaskStore) FindByHash(ctx context.Context, requestHash string) (Task, bool, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, false, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	id, ok := s.hashToID[requestHash]
	if !ok {
		return Task{}, false, nil
	}
	task, ok := s.tasks[id]
	return task, ok, nil
}

type RedisTaskStore struct {
	client *goredis.Client
	ttl    time.Duration
}

func NewRedisTaskStore(client *goredis.Client, ttl time.Duration) *RedisTaskStore {
	return &RedisTaskStore{client: client, ttl: ttl}
}

func (s *RedisTaskStore) Create(ctx context.Context, task Task) error {
	return s.save(ctx, task)
}

func (s *RedisTaskStore) Update(ctx context.Context, task Task) error {
	return s.save(ctx, task)
}

func (s *RedisTaskStore) save(ctx context.Context, task Task) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("marshal task: %w", err)
	}
	pipe := s.client.TxPipeline()
	pipe.Set(ctx, taskKey(task.ID), data, s.ttl)
	pipe.Set(ctx, hashKey(task.RequestHash), task.ID, s.ttl)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *RedisTaskStore) Get(ctx context.Context, id string) (Task, error) {
	data, err := s.client.Get(ctx, taskKey(id)).Bytes()
	if errors.Is(err, goredis.Nil) {
		return Task{}, ErrTaskNotFound
	}
	if err != nil {
		return Task{}, err
	}
	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return Task{}, fmt.Errorf("decode task: %w", err)
	}
	return task, nil
}

func (s *RedisTaskStore) FindByHash(ctx context.Context, requestHash string) (Task, bool, error) {
	id, err := s.client.Get(ctx, hashKey(requestHash)).Result()
	if errors.Is(err, goredis.Nil) {
		return Task{}, false, nil
	}
	if err != nil {
		return Task{}, false, err
	}
	task, err := s.Get(ctx, id)
	if errors.Is(err, ErrTaskNotFound) {
		return Task{}, false, nil
	}
	return task, err == nil, err
}

func taskKey(id string) string {
	return "travel:task:" + id
}

func hashKey(hash string) string {
	return "travel:request_hash:" + hash
}
