package analyzer

import "testing"

func TestAnalyze(t *testing.T) {
	tests := []struct {
		name     string
		password string
		score    int
		strength string
		length   int
	}{
		{"weak", "hello", 1, "Weak", 5},
		{"medium", "Hello123", 4, "Medium", 8},
		{"strong", "Hello123!", 5, "Strong", 9},
		{"unicode rune count", "Éclair12!", 5, "Strong", 9},
	}

	a := PasswordAnalyzer{}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := a.Analyze(tc.password)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Score != tc.score || result.Strength != tc.strength || result.Length != tc.length {
				t.Errorf("got score=%d strength=%s length=%d; want score=%d strength=%s length=%d",
					result.Score, result.Strength, result.Length, tc.score, tc.strength, tc.length)
			}
		})
	}
}

func TestAnalyzeInvalidInput(t *testing.T) {
	for _, password := range []string{"", "   ", "\n\t"} {
		if _, err := (PasswordAnalyzer{}).Analyze(password); err == nil {
			t.Errorf("expected error for whitespace-only input %q", password)
		}
	}
}

func TestAnalyzeCharacterFlags(t *testing.T) {
	result, err := (PasswordAnalyzer{}).Analyze("Hello123!")
	if err != nil {
		t.Fatal(err)
	}
	if !result.HasUpper || !result.HasLower || !result.HasDigit || !result.HasSpecial {
		t.Errorf("expected all character categories, got %+v", result)
	}
}
