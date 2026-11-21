package service

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/Uakks/sis-project/internal/analyzer"
	"github.com/Uakks/sis-project/internal/task"
)

var (
	ErrNotFound       = errors.New("task not found")
	ErrInvalidState   = errors.New("invalid task state")
	ErrQueueFull      = errors.New("task queue is full")
	ErrNotStarted     = errors.New("worker pool has not started")
	ErrStopped        = errors.New("service has stopped")
	ErrAlreadyStarted = errors.New("worker pool already started")
	ErrAsyncOnly      = errors.New("use SubmitTask after starting workers")
	ErrNoAnalyzer     = errors.New("analyzer is not configured")
)

// Service owns the task map, the bounded queue, and all worker goroutines.
// mu protects the map, every stored Task's mutable fields, nextID, and
// started/stopped/queue state. Results are returned as independent snapshots.
type Service struct {
	mu       sync.RWMutex
	tasks    map[int]*task.Task
	nextID   int
	analyzer analyzer.Analyzer

	queue   chan int
	started bool
	stopped bool
	wg      sync.WaitGroup
}

func NewService(a analyzer.Analyzer) *Service {
	return &Service{
		tasks:    make(map[int]*task.Task),
		nextID:   1,
		analyzer: a,
	}
}

// CreateTask validates and stores a task as Pending.
func (s *Service) CreateTask(password string) (*task.Task, error) {
	if strings.TrimSpace(password) == "" {
		return nil, errors.New("password cannot be empty or whitespace-only")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return nil, ErrStopped
	}

	created := &task.Task{
		ID:       s.nextID,
		Password: password,
		Status:   task.Pending,
	}
	s.tasks[created.ID] = created
	s.nextID++
	return copyTask(created), nil
}

// GetTask returns a snapshot that cannot mutate the stored task or its result.
func (s *Service) GetTask(id int) (*task.Task, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stored, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("task %d: %w", id, ErrNotFound)
	}
	return copyTask(stored), nil
}

// ProcessTask preserves Checkpoint 1's synchronous method for legacy callers.
// Once the worker pool is started, callers must use SubmitTask instead.
func (s *Service) ProcessTask(id int) error {
	return s.process(id, task.Pending, true)
}

// process transitions a task to Processing while holding mu, releases the
// lock during analysis, and locks again to save the correct task's result.
func (s *Service) process(id int, expected task.Status, synchronous bool) error {
	s.mu.Lock()
	if synchronous {
		if s.stopped {
			s.mu.Unlock()
			return ErrStopped
		}
		if s.started {
			s.mu.Unlock()
			return ErrAsyncOnly
		}
	}

	stored, ok := s.tasks[id]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("task %d: %w", id, ErrNotFound)
	}
	if stored.Status != expected {
		s.mu.Unlock()
		return fmt.Errorf("task %d is %s: %w", id, stored.Status, ErrInvalidState)
	}
	if s.analyzer == nil {
		s.mu.Unlock()
		return ErrNoAnalyzer
	}

	stored.Status = task.Processing
	password := stored.Password
	s.mu.Unlock()

	// Important: analysis runs without holding the mutex. This allows actual
	// overlap between workers and GetTask calls during processing.
	result, err := s.analyzer.Analyze(password)

	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		stored.Status = task.Failed
		stored.Error = err.Error()
		return fmt.Errorf("analyze task %d: %w", id, err)
	}
	stored.Result = &result
	stored.Error = ""
	stored.Status = task.Completed
	return nil
}

func copyTask(original *task.Task) *task.Task {
	copied := *original
	if original.Result != nil {
		resultCopy := *original.Result
		copied.Result = &resultCopy
	}
	return &copied
}
