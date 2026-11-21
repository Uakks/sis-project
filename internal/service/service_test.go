package service

import (
	"errors"
	"testing"

	"github.com/Uakks/sis-project/internal/task"
)

type fakeAnalyzer struct{}

func (fakeAnalyzer) Analyze(password string) (task.Result, error) {
	return task.Result{Length: len(password), Score: 5, Strength: "Strong"}, nil
}

type failingAnalyzer struct{}

func (failingAnalyzer) Analyze(string) (task.Result, error) {
	return task.Result{}, errors.New("analysis failed")
}

func TestCreateTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}
	if created.ID != 1 || created.Status != task.Pending || created.Result != nil {
		t.Errorf("unexpected new task: %+v", created)
	}
}

func TestCreateTaskInvalidInput(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	for _, password := range []string{"", "   ", "\n"} {
		if _, err := s.CreateTask(password); err == nil {
			t.Errorf("expected validation error for %q", password)
		}
	}
}

func TestGetTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}
	retrieved, err := s.GetTask(created.ID)
	if err != nil || retrieved.ID != created.ID {
		t.Fatalf("got task=%+v error=%v; expected ID=%d", retrieved, err, created.ID)
	}
}

func TestGetNonexistentTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if _, err := s.GetTask(999); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestProcessTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	created, err := s.CreateTask("Hello123!")
	if err != nil {
		t.Fatal(err)
	}
	if err := s.ProcessTask(created.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.GetTask(created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != task.Completed || got.Result == nil {
		t.Fatalf("expected completed task with result; got %+v", got)
	}
	if got.Result.Score != 5 || got.Result.Length != 9 {
		t.Errorf("unexpected analysis result: %+v", got.Result)
	}
}

func TestProcessCompletedTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	created, _ := s.CreateTask("Hello123!")
	if err := s.ProcessTask(created.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ProcessTask(created.ID); !errors.Is(err, ErrInvalidState) {
		t.Errorf("expected ErrInvalidState, got %v", err)
	}
}

func TestProcessNonexistentTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.ProcessTask(999); !errors.Is(err, ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestMultipleTasks(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	first, _ := s.CreateTask("Hello123!")
	second, _ := s.CreateTask("Another123!")
	if first.ID == second.ID {
		t.Fatal("two tasks received the same ID")
	}
	if err := s.ProcessTask(first.ID); err != nil {
		t.Fatal(err)
	}
	gotFirst, _ := s.GetTask(first.ID)
	gotSecond, _ := s.GetTask(second.ID)
	if gotFirst.Status != task.Completed || gotSecond.Status != task.Pending || gotSecond.Result != nil {
		t.Errorf("tasks interfered: first=%+v second=%+v", gotFirst, gotSecond)
	}
}

func TestResultIsDefensiveCopy(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	created, _ := s.CreateTask("Hello123!")
	if err := s.ProcessTask(created.ID); err != nil {
		t.Fatal(err)
	}
	copy1, _ := s.GetTask(created.ID)
	copy1.Status = task.Pending
	copy1.Result.Score = -100
	copy2, _ := s.GetTask(created.ID)
	if copy2.Status != task.Completed || copy2.Result.Score != 5 {
		t.Errorf("external changes corrupted internal task: %+v", copy2)
	}
}

func TestSynchronousAnalysisFailure(t *testing.T) {
	s := NewService(failingAnalyzer{})
	created, _ := s.CreateTask("Example123!")
	if err := s.ProcessTask(created.ID); err == nil {
		t.Fatal("expected analyzer failure")
	}
	got, _ := s.GetTask(created.ID)
	if got.Status != task.Failed || got.Result != nil || got.Error == "" {
		t.Errorf("expected failed task with error and no result, got %+v", got)
	}
}
