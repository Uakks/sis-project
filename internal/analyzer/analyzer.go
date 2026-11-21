package analyzer

import (
	"errors"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Uakks/sis-project/internal/task"
)

// Analyzer allows the service to accept different analyzer implementations,
// including test fakes that can coordinate concurrent worker tests.
type Analyzer interface {
	Analyze(password string) (task.Result, error)
}

// PasswordAnalyzer applies a simple five-rule educational scoring model.
// The score is not an estimate of real-world password security.
type PasswordAnalyzer struct{}

func (PasswordAnalyzer) Analyze(password string) (task.Result, error) {
	if strings.TrimSpace(password) == "" {
		return task.Result{}, errors.New("password cannot be empty or whitespace-only")
	}

	result := task.Result{Length: utf8.RuneCountInString(password)}

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			result.HasUpper = true
		case unicode.IsLower(char):
			result.HasLower = true
		case unicode.IsDigit(char):
			result.HasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			result.HasSpecial = true
		}
	}

	if result.Length >= 8 {
		result.Score++
	}
	if result.HasUpper {
		result.Score++
	}
	if result.HasLower {
		result.Score++
	}
	if result.HasDigit {
		result.Score++
	}
	if result.HasSpecial {
		result.Score++
	}

	switch {
	case result.Score <= 2:
		result.Strength = "Weak"
	case result.Score <= 4:
		result.Strength = "Medium"
	default:
		result.Strength = "Strong"
	}

	return result, nil
}
