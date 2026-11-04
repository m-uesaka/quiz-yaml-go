package quiz_yaml_converter

import (
	"reflect"
	"testing"
)

func TestQuizItemHasCriteria(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		expected bool
	}{
		{name: "nil", criteria: nil, expected: false},
		{name: "zero value", criteria: &Criteria{}, expected: false},
		{name: "empty lists", criteria: &Criteria{OK: []string{}}, expected: false},
		{name: "with entries", criteria: &Criteria{NG: []string{"ng1"}}, expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			item := QuizItem{Question: "q", Answer: "a", Criteria: tt.criteria}

			// Act
			result := item.HasCriteria()

			// Assert
			if result != tt.expected {
				t.Errorf("HasCriteria() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestQuizItemCriteriaAccessors(t *testing.T) {
	tests := []struct {
		name           string
		criteria       *Criteria
		expectedOK     []string
		expectedNG     []string
		expectedRepeat []string
	}{
		{
			name: "all keys present",
			criteria: &Criteria{
				OK:     []string{"ok1", "ok2"},
				NG:     []string{"ng1"},
				Repeat: []string{"rep1"},
			},
			expectedOK:     []string{"ok1", "ok2"},
			expectedNG:     []string{"ng1"},
			expectedRepeat: []string{"rep1"},
		},
		{
			name:       "only ok",
			criteria:   &Criteria{OK: []string{"ok1"}},
			expectedOK: []string{"ok1"},
		},
		{
			name:     "nil criteria",
			criteria: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			item := QuizItem{Question: "q", Answer: "a", Criteria: tt.criteria}

			// Act
			ok, ng, repeat := item.OK(), item.NG(), item.Repeat()

			// Assert
			if !reflect.DeepEqual(ok, tt.expectedOK) {
				t.Errorf("OK() = %v, want %v", ok, tt.expectedOK)
			}
			if !reflect.DeepEqual(ng, tt.expectedNG) {
				t.Errorf("NG() = %v, want %v", ng, tt.expectedNG)
			}
			if !reflect.DeepEqual(repeat, tt.expectedRepeat) {
				t.Errorf("Repeat() = %v, want %v", repeat, tt.expectedRepeat)
			}
		})
	}
}
