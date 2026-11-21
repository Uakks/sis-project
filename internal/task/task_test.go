package task

import "testing"

func TestTaskCreation(t *testing.T) {
	item := Task{ID: 1, Password: "Example123!", Status: Pending}
	if item.ID != 1 || item.Status != Pending || item.Result != nil {
		t.Fatalf("unexpected new task: %+v", item)
	}
}
