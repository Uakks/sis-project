package service

import (
	"fmt"
	"strings"

	"github.com/Uakks/sis-project/internal/analyzer"
	"github.com/Uakks/sis-project/internal/task"
)

type Service struct {
	tasks    map[int]*task.Task
	nextID   int
	analyzer analyzer.Analyzer
}

func NewService(a analyzer.Analyzer) *Service {
	return &Service{
		tasks:    make(map[int]*task.Task),
		nextID:   1,
		analyzer: a,
	}
}

func (s *Service) CreateTask(password string) (*task.Task, error) {
	if strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("password cannot be empty")
	}

	newTask := &task.Task{
		ID:       s.nextID,
		Password: password,
		Status:   task.Pending,
		Result:   nil,
	}

	s.tasks[newTask.ID] = newTask
	s.nextID++

	return copyTask(newTask), nil
}

func (s *Service) GetTask(id int) (*task.Task, error) {
	foundTask, exists := s.tasks[id]

	if !exists {
		return nil, fmt.Errorf("task with ID %d not found", id)
	}

	return copyTask(foundTask), nil
}

func (s *Service) ProcessTask(id int) error {
	foundTask, exists := s.tasks[id]

	if !exists {
		return fmt.Errorf("task with ID %d not found", id)
	}

	if foundTask.Status != task.Pending {
		return fmt.Errorf(
			"task %d cannot be processed with status %s",
			id,
			foundTask.Status,
		)
	}

	if s.analyzer == nil {
		return fmt.Errorf("password analyzer is not configured")
	}

	foundTask.Status = task.Processing

	result, err := s.analyzer.Analyze(foundTask.Password)

	if err != nil {
		foundTask.Status = task.Pending
		return fmt.Errorf("failed to process task %d: %w", id, err)
	}

	foundTask.Result = &result
	foundTask.Status = task.Completed

	return nil
}

// Return a copy so callers cannot modify stored task data.
func copyTask(original *task.Task) *task.Task {
	copied := *original

	if original.Result != nil {
		resultCopy := *original.Result
		copied.Result = &resultCopy
	}

	return &copied
}
