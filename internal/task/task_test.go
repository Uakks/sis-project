package task

import "testing"

func TestTaskCreation(t *testing.T) {
	task := Task{
		ID:       1,
		Password: "Hello123!",
		Status:   Pending,
	}

	if task.ID != 1 {
		t.Errorf("expected ID 1, got %d", task.ID)
	}

	if task.Status != Pending {
		t.Errorf("expected Pending status, got %s", task.Status)
	}
}
