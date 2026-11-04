package quiz_yaml_converter

import (
	"testing"
)

func TestAddQuotesIfNeeded(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no quotes",
			input:    "test",
			expected: "「test」",
		},
		{
			name:     "already has both quotes",
			input:    "「test」",
			expected: "「test」",
		},
		{
			name:     "has opening quote only",
			input:    "「test",
			expected: "「test」",
		},
		{
			name:     "has closing quote only",
			input:    "test」",
			expected: "「test」",
		},
		{
			name:     "complex case with parentheses",
			input:    "「美術館」（おまけ）",
			expected: "「美術館」（おまけ）",
		},
		{
			name:     "opening quote with closing quote inside",
			input:    "「test」something",
			expected: "「test」something",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := AddQuotesIfNeeded(tt.input)
			if result != tt.expected {
				t.Errorf("AddQuotesIfNeeded(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatCriteriaSection(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		suffix   string
		expected string
	}{
		{
			name:     "empty items",
			items:    []string{},
			suffix:   "",
			expected: "",
		},
		{
			name:     "single item no suffix",
			items:    []string{"ok1"},
			suffix:   "",
			expected: "「ok1」",
		},
		{
			name:     "single item with suffix",
			items:    []string{"ng1"},
			suffix:   "は誤答",
			expected: "「ng1」は誤答",
		},
		{
			name:     "multiple items no suffix",
			items:    []string{"ok1", "ok2"},
			suffix:   "",
			expected: "「ok1」「ok2」",
		},
		{
			name:     "multiple items with suffix",
			items:    []string{"ng1", "ng2"},
			suffix:   "は誤答",
			expected: "「ng1」「ng2」は誤答",
		},
		{
			name:     "items already quoted",
			items:    []string{"「ok1」", "ok2"},
			suffix:   "",
			expected: "「ok1」「ok2」",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatCriteriaSection(tt.items, tt.suffix)
			if result != tt.expected {
				t.Errorf("formatCriteriaSection(%v, %q) = %q, want %q", tt.items, tt.suffix, result, tt.expected)
			}
		})
	}
}

func TestCriteriaFormat(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		expected string
	}{
		{
			name:     "nil criteria",
			criteria: nil,
			expected: "",
		},
		{
			name:     "empty criteria",
			criteria: &Criteria{},
			expected: "",
		},
		{
			name: "only ok",
			criteria: &Criteria{
				OK: []string{"ok1", "ok2"},
			},
			expected: "「ok1」「ok2」",
		},
		{
			name: "only ng",
			criteria: &Criteria{
				NG: []string{"ng1"},
			},
			expected: "「ng1」は誤答",
		},
		{
			name: "only repeat",
			criteria: &Criteria{
				Repeat: []string{"rep1"},
			},
			expected: "「rep1」はもう一度",
		},
		{
			name: "ok and ng",
			criteria: &Criteria{
				OK: []string{"ok1", "ok2"},
				NG: []string{"ng1"},
			},
			expected: "「ok1」「ok2」／「ng1」は誤答",
		},
		{
			name: "all sections",
			criteria: &Criteria{
				OK:     []string{"ok1", "ok2"},
				NG:     []string{"ng1"},
				Repeat: []string{"rep1"},
			},
			expected: "「ok1」「ok2」／「ng1」は誤答／「rep1」はもう一度",
		},
		{
			name: "complex with already quoted items",
			criteria: &Criteria{
				OK: []string{"「古典絵画館」", "古典美術館", "「アルテ・マイスター美術館」（おまけ）"},
			},
			expected: "「古典絵画館」「古典美術館」「アルテ・マイスター美術館」（おまけ）",
		},
		{
			name: "reading correction case",
			criteria: &Criteria{
				Repeat: []string{"えんががわ（読みが違うのでもう一度）"},
			},
			expected: "「えんががわ（読みが違うのでもう一度）」はもう一度",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.criteria.Format()
			if result != tt.expected {
				t.Errorf("(%+v).Format() = %q, want %q", tt.criteria, result, tt.expected)
			}
		})
	}
}

func TestCriteriaIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		expected bool
	}{
		{name: "nil", criteria: nil, expected: true},
		{name: "zero value", criteria: &Criteria{}, expected: true},
		{name: "empty lists", criteria: &Criteria{OK: []string{}, NG: []string{}, Repeat: []string{}}, expected: true},
		{name: "only ok", criteria: &Criteria{OK: []string{"ok1"}}, expected: false},
		{name: "only ng", criteria: &Criteria{NG: []string{"ng1"}}, expected: false},
		{name: "only repeat", criteria: &Criteria{Repeat: []string{"rep1"}}, expected: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := tt.criteria.IsEmpty()

			// Assert
			if result != tt.expected {
				t.Errorf("(%+v).IsEmpty() = %v, want %v", tt.criteria, result, tt.expected)
			}
		})
	}
}

// ベンチマークテスト
func BenchmarkAddQuotesIfNeeded(b *testing.B) {
	testCases := []string{
		"test",
		"「test」",
		"「test",
		"test」",
		"「美術館」（おまけ）",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tc := range testCases {
			AddQuotesIfNeeded(tc)
		}
	}
}

func BenchmarkFormatCriteria(b *testing.B) {
	criteria := &Criteria{
		OK:     []string{"ok1", "ok2", "ok3"},
		NG:     []string{"ng1", "ng2"},
		Repeat: []string{"rep1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		criteria.Format()
	}
}
