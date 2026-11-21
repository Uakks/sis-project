package service

import (
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/Uakks/sis-project/internal/analyzer"
	"github.com/Uakks/sis-project/internal/task"
)

// blockingAnalyzer tells tests when a worker entered Analyze, then blocks
// until the test releases it. This proves overlap without arbitrary sleeps.
type blockingAnalyzer struct {
	entered chan struct{}
	release <-chan struct{}
}

func (a blockingAnalyzer) Analyze(password string) (task.Result, error) {
	a.entered <- struct{}{}
	<-a.release
	return task.Result{Length: len(password), Score: len(password), Strength: "test"}, nil
}

func waitForEntry(t *testing.T, entered <-chan struct{}) {
	t.Helper()
	select {
	case <-entered:
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for an analyzer to start")
	}
}

func TestWorkerPoolStartup(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(2, 3); err != nil {
		t.Fatalf("failed to start pool: %v", err)
	}
	defer s.Stop()
	if err := s.Start(2, 3); !errors.Is(err, ErrAlreadyStarted) {
		t.Errorf("expected ErrAlreadyStarted, got %v", err)
	}
}

func TestStartInvalidConfiguration(t *testing.T) {
	tests := []struct {
		workers, capacity int
	}{
		{0, 3}, {1, 3}, {2, 0}, {2, -1},
	}
	for _, tc := range tests {
		s := NewService(fakeAnalyzer{})
		if err := s.Start(tc.workers, tc.capacity); err == nil {
			t.Errorf("expected error for workers=%d capacity=%d", tc.workers, tc.capacity)
		}
	}
}

func TestSubmitTaskBeforeStart(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	item, _ := s.CreateTask("Hello123!")
	if err := s.SubmitTask(item.ID); !errors.Is(err, ErrNotStarted) {
		t.Errorf("expected ErrNotStarted, got %v", err)
	}
}

func TestAsyncProcessingResults(t *testing.T) {
	s := NewService(analyzer.PasswordAnalyzer{})
	if err := s.Start(2, 10); err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		password string
		score    int
		strength string
	}{
		{"hello", 1, "Weak"},
		{"Hello123", 4, "Medium"},
		{"Hello123!", 5, "Strong"},
		{"ABC12345", 3, "Medium"},
	}
	var ids []int
	for _, tc := range cases {
		item, err := s.CreateTask(tc.password)
		if err != nil {
			t.Fatal(err)
		}
		ids = append(ids, item.ID)
		if err := s.SubmitTask(item.ID); err != nil {
			t.Fatalf("submit task %d: %v", item.ID, err)
		}
	}

	// Stop drains every accepted task, so results are ready afterward.
	s.Stop()
	for i, id := range ids {
		got, err := s.GetTask(id)
		if err != nil {
			t.Fatal(err)
		}
		if got.Status != task.Completed || got.Result == nil {
			t.Fatalf("task %d was not completed: %+v", id, got)
		}
		if got.Result.Score != cases[i].score || got.Result.Strength != cases[i].strength {
			t.Errorf("task %d: got %+v, wanted score=%d strength=%s", id,
				got.Result, cases[i].score, cases[i].strength)
		}
	}
}

func TestTwoWorkersOverlapAndFullQueue(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	s := NewService(blockingAnalyzer{entered: entered, release: release})
	if err := s.Start(2, 1); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	defer close(release)

	// Start two concurrent analyses. Wait until both have actually entered.
	first, _ := s.CreateTask("first")
	if err := s.SubmitTask(first.ID); err != nil {
		t.Fatal(err)
	}
	waitForEntry(t, entered)

	second, _ := s.CreateTask("second")
	if err := s.SubmitTask(second.ID); err != nil {
		t.Fatal(err)
	}
	waitForEntry(t, entered) // both are blocked inside Analyze

	third, _ := s.CreateTask("third")
	if err := s.SubmitTask(third.ID); err != nil {
		t.Fatal(err)
	}
	queued, _ := s.GetTask(third.ID)
	if queued.Status != task.Queued {
		t.Errorf("expected Queued, got %s", queued.Status)
	}

	fourth, _ := s.CreateTask("fourth")
	if err := s.SubmitTask(fourth.ID); !errors.Is(err, ErrQueueFull) {
		t.Fatalf("expected ErrQueueFull, got %v", err)
	}
	pending, _ := s.GetTask(fourth.ID)
	if pending.Status != task.Pending {
		t.Errorf("rejected task should remain Pending, got %s", pending.Status)
	}

	if err := s.SubmitTask(third.ID); !errors.Is(err, ErrInvalidState) {
		t.Errorf("expected duplicate submission error, got %v", err)
	}

	// Deferred close(release) lets the workers finish before deferred Stop.
}

func TestRetrievalWhileProcessing(t *testing.T) {
	release := make(chan struct{})
	entered := make(chan struct{}, 2)
	s := NewService(blockingAnalyzer{entered: entered, release: release})
	if err := s.Start(2, 2); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	defer close(release)

	item, _ := s.CreateTask("Example123!")
	if err := s.SubmitTask(item.ID); err != nil {
		t.Fatal(err)
	}
	waitForEntry(t, entered)
	got, err := s.GetTask(item.ID)
	if err != nil || got.Status != task.Processing {
		t.Fatalf("GetTask during analysis: task=%+v error=%v", got, err)
	}
	if got.Result != nil {
		t.Error("a processing task must not have a completed result")
	}
}

func TestConcurrentSubmissionsOfSameTask(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(2, 2); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	item, _ := s.CreateTask("Hello123!")

	const callers = 30
	results := make(chan error, callers)
	var wg sync.WaitGroup
	for i := 0; i < callers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- s.SubmitTask(item.ID)
		}()
	}
	wg.Wait()
	close(results)

	accepted := 0
	for err := range results {
		if err == nil {
			accepted++
		} else if !errors.Is(err, ErrInvalidState) {
			t.Errorf("unexpected submission error: %v", err)
		}
	}
	if accepted != 1 {
		t.Errorf("expected exactly one accepted submission, got %d", accepted)
	}
}

func TestConcurrentTaskAccess(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(3, 100); err != nil {
		t.Fatal(err)
	}

	const count = 100
	ids := make(chan int, count)
	var wg sync.WaitGroup
	for i := 0; i < count; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			item, err := s.CreateTask(fmt.Sprintf("Example%d!", i))
			if err != nil {
				t.Errorf("CreateTask: %v", err)
				return
			}
			ids <- item.ID
		}(i)
	}
	wg.Wait()
	close(ids)

	seen := make(map[int]bool)
	for id := range ids {
		if seen[id] {
			t.Errorf("duplicate task ID %d", id)
		}
		seen[id] = true
	}
	if len(seen) != count {
		t.Errorf("created %d tasks instead of %d", len(seen), count)
	}

	for id := range seen {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := s.SubmitTask(id); err != nil {
				t.Errorf("SubmitTask(%d): %v", id, err)
			}
			if _, err := s.GetTask(id); err != nil {
				t.Errorf("GetTask(%d): %v", id, err)
			}
		}(id)
	}
	wg.Wait()
	s.Stop()
	for id := range seen {
		got, err := s.GetTask(id)
		if err != nil || got.Status != task.Completed || got.Result == nil {
			t.Errorf("task %d not completed correctly: %+v, %v", id, got, err)
		}
	}
}

func TestAsyncAnalyzerFailure(t *testing.T) {
	s := NewService(failingAnalyzer{})
	if err := s.Start(2, 2); err != nil {
		t.Fatal(err)
	}
	item, _ := s.CreateTask("Hello123!")
	if err := s.SubmitTask(item.ID); err != nil {
		t.Fatal(err)
	}
	s.Stop()
	got, _ := s.GetTask(item.ID)
	if got.Status != task.Failed || got.Result != nil || got.Error == "" {
		t.Errorf("expected Failed with error, got %+v", got)
	}
}

func TestStopDrainsAndRejectsNewTasks(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(2, 20); err != nil {
		t.Fatal(err)
	}

	var ids []int
	for i := 0; i < 20; i++ {
		item, _ := s.CreateTask(fmt.Sprintf("Example%d!", i))
		ids = append(ids, item.ID)
		if err := s.SubmitTask(item.ID); err != nil {
			t.Fatal(err)
		}
	}
	s.Stop()
	s.Stop() // idempotent
	for _, id := range ids {
		got, err := s.GetTask(id)
		if err != nil || got.Status != task.Completed || got.Result == nil {
			t.Errorf("accepted task %d lost at shutdown: %+v, %v", id, got, err)
		}
	}
	if _, err := s.CreateTask("newtask"); !errors.Is(err, ErrStopped) {
		t.Errorf("expected ErrStopped from CreateTask, got %v", err)
	}
	if err := s.SubmitTask(ids[0]); !errors.Is(err, ErrStopped) {
		t.Errorf("expected ErrStopped from SubmitTask, got %v", err)
	}
	if err := s.Start(2, 3); !errors.Is(err, ErrStopped) {
		t.Errorf("expected ErrStopped from Start, got %v", err)
	}
}

func TestConcurrentStopAndSubmit(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(2, 30); err != nil {
		t.Fatal(err)
	}

	const count = 30
	ids := make([]int, 0, count)
	for i := 0; i < count; i++ {
		item, _ := s.CreateTask(fmt.Sprintf("Example%d!", i))
		ids = append(ids, item.ID)
	}

	results := make(chan struct {
		id  int
		err error
	}, count)
	var wg sync.WaitGroup
	for _, id := range ids {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			results <- struct {
				id  int
				err error
			}{id, s.SubmitTask(id)}
		}(id)
	}
	stopFinished := make(chan struct{})
	go func() {
		s.Stop()
		close(stopFinished)
	}()

	wg.Wait()
	close(results)
	<-stopFinished

	for r := range results {
		got, err := s.GetTask(r.id)
		if err != nil {
			t.Fatal(err)
		}
		if r.err == nil {
			if got.Status != task.Completed || got.Result == nil {
				t.Errorf("accepted task %d was lost: %+v", r.id, got)
			}
		} else if !errors.Is(r.err, ErrStopped) {
			t.Errorf("unexpected SubmitTask error %v", r.err)
		} else if got.Status != task.Pending {
			t.Errorf("rejected task %d should remain Pending, got %s", r.id, got.Status)
		}
	}
}

func TestSynchronousProcessingDisabledAfterStart(t *testing.T) {
	s := NewService(fakeAnalyzer{})
	if err := s.Start(2, 1); err != nil {
		t.Fatal(err)
	}
	defer s.Stop()
	item, _ := s.CreateTask("Hello123!")
	if err := s.ProcessTask(item.ID); !errors.Is(err, ErrAsyncOnly) {
		t.Errorf("expected ErrAsyncOnly, got %v", err)
	}
}
