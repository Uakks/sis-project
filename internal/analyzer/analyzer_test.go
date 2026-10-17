package analyzer

import (
	"testing"
)

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name             string
		password         string
		expectedScore    int
		expectedStrength string
	}{
		{
			name:             "Strong password",
			password:         "Hello123!",
			expectedScore:    5,
			expectedStrength: "Strong",
		},
		{
			name:             "Medium password",
			password:         "Hello123",
			expectedScore:    4,
			expectedStrength: "Medium",
		},
		{
			name:             "Weak password",
			password:         "hello",
			expectedScore:    1,
			expectedStrength: "Weak",
		},
	}

	analyzer := PasswordAnalyzer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.Analyze(tt.password)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result.Score != tt.expectedScore {
				t.Errorf(
					"expected score %d, got %d",
					tt.expectedScore,
					result.Score,
				)
			}

			if result.Strength != tt.expectedStrength {
				t.Errorf(
					"expected strength %s, got %s",
					tt.expectedStrength,
					result.Strength,
				)
			}
		})
	}
}

func TestAnalyzeInvalidInput(t *testing.T) {
	tests := []struct {
		name     string
		password string
	}{
		{"Empty password", ""},
		{"Whitespace only", "   "},
	}

	analyzer := PasswordAnalyzer{}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := analyzer.Analyze(tt.password)

			if err == nil {
				t.Error("expected an error, got nil")
			}
		})
	}
}

func TestAnalyzeResult(t *testing.T) {
	analyzer := PasswordAnalyzer{}

	result, err := analyzer.Analyze("Hello123!")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Length != 9 {
		t.Errorf("expected length 9, got %d", result.Length)
	}

	if !result.HasUpper {
		t.Error("expected uppercase characters")
	}

	if !result.HasLower {
		t.Error("expected lowercase characters")
	}

	if !result.HasDigit {
		t.Error("expected digits")
	}

	if !result.HasSpecial {
		t.Error("expected special characters")
	}
}
