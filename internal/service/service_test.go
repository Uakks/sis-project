package service

import (
	"testing"

	"github.com/Uakks/sis-project/internal/task"
)

// Fake analyzer for testing the service.
type fakeAnalyzer struct{}

func (f fakeAnalyzer) Analyze(password string) (task.Result, error) {
	return task.Result{
		Length:   len(password),
		Score:    5,
		Strength: "Strong",
	}, nil
}

func TestCreateTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	created, err := s.CreateTask("Hello123!")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if created.ID <= 0 {
		t.Error("expected a positive task ID")
	}

	if created.Status != task.Pending {
		t.Errorf("expected Pending, got %s", created.Status)
	}

	if created.Result != nil {
		t.Error("new task should not have a result")
	}
}

func TestCreateTaskInvalidInput(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	_, err := s.CreateTask("")

	if err == nil {
		t.Error("expected error for empty password")
	}
}

func TestGetTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}

	retrieved, err := s.GetTask(created.ID)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if retrieved.ID != created.ID {
		t.Errorf(
			"expected ID %d, got %d",
			created.ID,
			retrieved.ID,
		)
	}
}

func TestGetNonexistentTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	_, err := s.GetTask(999)

	if err == nil {
		t.Error("expected error for nonexistent task")
	}
}

func TestProcessTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}

	err = s.ProcessTask(created.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	processed, err := s.GetTask(created.ID)
	if err != nil {
		t.Fatal(err)
	}

	if processed.Status != task.Completed {
		t.Errorf(
			"expected Completed, got %s",
			processed.Status,
		)
	}

	if processed.Result == nil {
		t.Fatal("expected processing result, got nil")
	}

	if processed.Result.Score != 5 {
		t.Errorf(
			"expected score 5, got %d",
			processed.Result.Score,
		)
	}

	if processed.Result.Length != 9 {
		t.Errorf(
			"expected length 9, got %d",
			processed.Result.Length,
		)
	}
}

func TestProcessCompletedTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}

	if err := s.ProcessTask(created.ID); err != nil {
		t.Fatal(err)
	}

	err = s.ProcessTask(created.ID)

	if err == nil {
		t.Error("expected error when processing completed task")
	}
}

func TestMultipleTasks(t *testing.T) {
	s := NewService(fakeAnalyzer{})

	first, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}

	second, err := s.CreateTask("Another123!")
	if err != nil {
		t.Fatal(err)
	}

	if first.ID == second.ID {
		t.Error("tasks must have unique IDs")
	}

	if err := s.ProcessTask(first.ID); err != nil {
		t.Fatal(err)
	}

	firstTask, err := s.GetTask(first.ID)
	if err != nil {
		t.Fatal(err)
	}

	secondTask, err := s.GetTask(second.ID)
	if err != nil {
		t.Fatal(err)
	}

	if firstTask.Status != task.Completed {
		t.Error("first task should be completed")
	}

	if secondTask.Status != task.Pending {
		t.Error("second task should still be pending")
	}

	if secondTask.Result != nil {
		t.Error("second task should not have a result yet")
	}
}
