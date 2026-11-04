package quiz_yaml_converter

import (
	"strings"
	"testing"
)

func TestValidateYAMLFileValid(t *testing.T) {
	// Arrange
	path := writeTempYAML(t, `- question: 問題1
  answer: 答え1
  criteria:
    ok: [ok1]
    ng: [ng1]
    repeat: [rep1]
- question: 問題2
  answer: 答え2
`)

	// Act
	result := ValidateYAMLFile(path)

	// Assert
	if !result.IsValid || len(result.Errors) != 0 {
		t.Errorf("ValidateYAMLFile() = %+v, want valid", result)
	}
	if result.Items != 2 {
		t.Errorf("Items = %d, want 2", result.Items)
	}
}

func TestValidateYAMLFileInvalidCriteria(t *testing.T) {
	tests := []struct {
		name       string
		content    string
		wantErrors []string
	}{
		{
			name:       "empty criteria entries",
			content:    "- question: 問題\n  answer: 答え\n  criteria:\n    ok: [ok1, \"\"]\n    ng: [\" \"]\n    repeat: [\"\"]\n",
			wantErrors: []string{"問題 1: criteria.ok[1] が空です", "問題 1: criteria.ng[0] が空です", "問題 1: criteria.repeat[0] が空です"},
		},
		{
			name:       "unknown criteria key",
			content:    "- question: 問題\n  answer: 答え\n  criteria:\n    maybe: [m1]\n",
			wantErrors: []string{"問題 1: 不正なcriteriaキー: 'maybe'（4行目） (使用可能: ok, ng, repeat)"},
		},
		{
			name:       "unknown criteria key is reported with other errors",
			content:    "- question: 問題1\n  answer: 答え1\n  criteria:\n    maybe: [m1]\n- question: 問題2\n  answer: 答え2\n- question: 問題3\n  answer: \"\"\n",
			wantErrors: []string{"問題 1: 不正なcriteriaキー: 'maybe'（4行目）", "問題 3: 答え (answer) が空です"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			path := writeTempYAML(t, tt.content)

			// Act
			result := ValidateYAMLFile(path)

			// Assert
			if result.IsValid {
				t.Fatalf("ValidateYAMLFile() IsValid = true, want false")
			}
			if result.Items == 0 {
				t.Errorf("Items = 0, want the number of items in the file")
			}
			if len(result.Errors) != len(tt.wantErrors) {
				t.Fatalf("Errors = %q, want %d errors", result.Errors, len(tt.wantErrors))
			}
			for i, want := range tt.wantErrors {
				if !strings.Contains(result.Errors[i], want) {
					t.Errorf("Errors[%d] = %q, want it to contain %q", i, result.Errors[i], want)
				}
			}
		})
	}
}
