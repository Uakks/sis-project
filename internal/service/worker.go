package service

import (
	"fmt"

	"github.com/Uakks/sis-project/internal/task"
)

// Start creates a fixed number (at least two) of reusable worker goroutines
// and a channel that holds at most queueCapacity waiting task IDs.
func (s *Service) Start(workers, queueCapacity int) error {
	if workers < 2 {
		return fmt.Errorf("workers must be at least 2 (got %d)", workers)
	}
	if queueCapacity < 1 {
		return fmt.Errorf("queue capacity must be positive (got %d)", queueCapacity)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return ErrStopped
	}
	if s.started {
		return ErrAlreadyStarted
	}
	if s.analyzer == nil {
		return ErrNoAnalyzer
	}

	s.queue = make(chan int, queueCapacity)
	s.started = true
	s.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go s.worker()
	}
	return nil
}

// SubmitTask attempts a non-blocking enqueue. If the queue is full,
// the operation fails and the task stays Pending.
// Locking the state change and channel send together prevents duplicates
// and protects against Stop closing the channel during submission.
func (s *Service) SubmitTask(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.stopped {
		return ErrStopped
	}
	if !s.started {
		return ErrNotStarted
	}

	stored, ok := s.tasks[id]
	if !ok {
		return fmt.Errorf("task %d: %w", id, ErrNotFound)
	}
	if stored.Status != task.Pending {
		return fmt.Errorf("task %d is %s: %w", id, stored.Status, ErrInvalidState)
	}

	select {
	case s.queue <- id:
		stored.Status = task.Queued
		return nil
	default:
		return ErrQueueFull
	}
}

// worker stays alive until the queue is closed and all queued IDs are read.
func (s *Service) worker() {
	defer s.wg.Done()
	for id := range s.queue {
		// A failing analyzer records Failed on its task. An error here does not
		// prevent the worker from handling the next task.
		_ = s.process(id, task.Queued, false)
	}
}

// Stop is safe to call concurrently and more than once. Closing the queue
// prevents new submissions; ranging workers drain already accepted tasks.
// It waits until every worker exits before returning.
func (s *Service) Stop() {
	s.mu.Lock()
	if !s.stopped {
		s.stopped = true
		if s.started {
			close(s.queue)
		}
	}
	s.mu.Unlock()

	s.wg.Wait()
}
