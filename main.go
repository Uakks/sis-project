package main

import (
	"fmt"

	"github.com/Uakks/sis-project/internal/service"
	"github.com/Uakks/sis-project/internal/analyzer"
)

func main() {
	// Create analyzer and service
	a := analyzer.PasswordAnalyzer{}
	s := service.NewService(a)

	// Sample passwords
	passwords := []string{
		"hello",
		"Hello123",
		"Hello123!",
	}

	// Create and process tasks
	for _, password := range passwords {
		created, err := s.CreateTask(password)
		if err != nil {
			fmt.Println("Creation error:", err)
			continue
		}

		fmt.Printf("Task %d created\n", created.ID)
		fmt.Println("Status:", created.Status)

		// Process task
		if err := s.ProcessTask(created.ID); err != nil {
			fmt.Println("Processing error:", err)
			continue
		}

		// Retrieve processed task
		completed, err := s.GetTask(created.ID)
		if err != nil {
			fmt.Println("Retrieval error:", err)
			continue
		}

		// Display result
		fmt.Println("Task ID:", completed.ID)
		fmt.Println("Status:", completed.Status)
		fmt.Println("Length:", completed.Result.Length)
		fmt.Println("Score:", completed.Result.Score)
		fmt.Println("Strength:", completed.Result.Strength)
		fmt.Println("-------------------------")
	}
}