package quiz_yaml_converter

import (
	"reflect"
	"testing"

	"gopkg.in/yaml.v3"
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

// << で取り込んだ内容が ok / ng / repeat のみの場合，マージ結果が読み込まれ，不正なキーが記録されないことを確認する．
func TestCriteriaUnmarshalYAMLValidMerge(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []Criteria
	}{
		{
			name: "alias of a custom field",
			content: `- question: Q1
  common: &b
    ok: [foo]
  criteria:
    <<: *b
    ng: [x]
`,
			want: []Criteria{{OK: []string{"foo"}, NG: []string{"x"}}},
		},
		{
			name: "sequence of aliases",
			content: `- question: Q1
  a: &a
    ok: [foo]
  b: &b
    repeat: [bar]
  criteria:
    <<: [*a, *b]
`,
			want: []Criteria{{OK: []string{"foo"}, Repeat: []string{"bar"}}},
		},
		{
			name: "nested merge",
			content: `- question: Q1
  base: &base
    ng: [x]
  mid: &mid
    <<: *base
    ok: [foo]
  criteria:
    <<: *mid
`,
			want: []Criteria{{OK: []string{"foo"}, NG: []string{"x"}}},
		},
		{
			name: "inline mapping",
			content: `- question: Q1
  criteria:
    <<: {ok: [foo]}
`,
			want: []Criteria{{OK: []string{"foo"}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var items []QuizItem

			// Act
			err := yaml.Unmarshal([]byte(tt.content), &items)

			// Assert
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}
			if len(items) != len(tt.want) {
				t.Fatalf("len(items) = %d, want %d", len(items), len(tt.want))
			}
			for i, want := range tt.want {
				if !reflect.DeepEqual(*items[i].Criteria, want) {
					t.Errorf("items[%d].Criteria = %+v, want %+v", i, *items[i].Criteria, want)
				}
			}
		})
	}
}

// << で取り込んだ内容に ok / ng / repeat 以外のキーがある場合，取り込んだ側のキーの行番号とともに記録されることを確認する．
func TestCriteriaUnmarshalYAMLUnknownKeysInMerge(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    [][]unknownCriteriaKey // 問題ごとの不正なキー
	}{
		{
			name: "alias of a custom field (issue #37)",
			content: `- question: Q1
  answer: A1
  common: &b
    okk: [foo]
  criteria:
    <<: *b
    ng: [x]
`,
			want: [][]unknownCriteriaKey{{{name: "okk", line: 4}}},
		},
		{
			name: "sequence of aliases with a typo in one",
			content: `- question: Q1
  a: &a
    ok: [foo]
  b: &b
    rpeat: [bar]
  criteria:
    <<: [*a, *b]
`,
			want: [][]unknownCriteriaKey{{{name: "rpeat", line: 5}}},
		},
		{
			name: "nested merge with a typo in the inner anchor",
			content: `- question: Q1
  base: &base
    ngg: [x]
  mid: &mid
    <<: *base
    ok: [foo]
  criteria:
    <<: *mid
`,
			want: [][]unknownCriteriaKey{{{name: "ngg", line: 3}}},
		},
		{
			name: "inline mapping",
			content: `- question: Q1
  criteria:
    <<: {okk: [foo]}
`,
			want: [][]unknownCriteriaKey{{{name: "okk", line: 3}}},
		},
		{
			name: "merged and direct unknown keys are recorded in key order",
			content: `- question: Q1
  common: &b
    okk: [foo]
  criteria:
    maybe: [m]
    <<: *b
    close: [c]
`,
			want: [][]unknownCriteriaKey{{{name: "maybe", line: 5}, {name: "okk", line: 3}, {name: "close", line: 7}}},
		},
		{
			name: "same mapping reached twice is recorded once",
			content: `- question: Q1
  a: &a
    okk: [foo]
  b: &b
    <<: *a
    ng: [x]
  criteria:
    <<: [*a, *b]
`,
			want: [][]unknownCriteriaKey{{{name: "okk", line: 3}}},
		},
		{
			name: "anchor on another item's criteria is reported for each item",
			content: `- question: Q1
  criteria: &common
    okk: [foo]
- question: Q2
  criteria:
    <<: *common
    ng: [x]
`,
			want: [][]unknownCriteriaKey{{{name: "okk", line: 3}}, {{name: "okk", line: 3}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			var items []QuizItem

			// Act
			err := yaml.Unmarshal([]byte(tt.content), &items)

			// Assert
			if err != nil {
				t.Fatalf("yaml.Unmarshal() error = %v", err)
			}
			if len(items) != len(tt.want) {
				t.Fatalf("len(items) = %d, want %d", len(items), len(tt.want))
			}
			for i, want := range tt.want {
				if !reflect.DeepEqual(items[i].Criteria.unknownKeys, want) {
					t.Errorf("items[%d].Criteria.unknownKeys = %+v, want %+v", i, items[i].Criteria.unknownKeys, want)
				}
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
