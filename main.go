package main

import (
	"fmt"

	"github.com/Uakks/sis-project/internal/analyzer"
	"github.com/Uakks/sis-project/internal/service"
)

func main() {
	s := service.NewService(analyzer.PasswordAnalyzer{})
	if err := s.Start(2, 10); err != nil {
		fmt.Println("Cannot start:", err)
		return
	}

	// Use invented sample passwords only, never real credentials.
	passwords := []string{"hello", "Hello123", "Hello123!", "ABCDE123!"}
	var ids []int

	for _, password := range passwords {
		created, err := s.CreateTask(password)
		if err != nil {
			fmt.Println("Create failed:", err)
			continue
		}
		if err := s.SubmitTask(created.ID); err != nil {
			fmt.Printf("Submit task %d failed: %v\n", created.ID, err)
			continue
		}
		ids = append(ids, created.ID)
		fmt.Printf("Submitted task %d\n", created.ID)
	}

	// Normal shutdown drains the queue and waits for all workers to finish.
	s.Stop()

	fmt.Println("\nResults:")
	for _, id := range ids {
		item, err := s.GetTask(id)
		if err != nil {
			fmt.Println(err)
			continue
		}
		if item.Result == nil {
			fmt.Printf("Task %d: %s (%s)\n", id, item.Status, item.Error)
			continue
		}
		fmt.Printf("Task %d: %s, score %d/5, %s\n",
			id, item.Status, item.Result.Score, item.Result.Strength)
	}
}
