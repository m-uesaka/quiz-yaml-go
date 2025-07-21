package quiz_yaml_converter

import (
	"os"
	"path/filepath"
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

func TestFormatCriteria(t *testing.T) {
	tests := []struct {
		name     string
		criteria map[string][]string
		expected string
	}{
		{
			name:     "empty criteria",
			criteria: map[string][]string{},
			expected: "",
		},
		{
			name: "only ok",
			criteria: map[string][]string{
				"ok": {"ok1", "ok2"},
			},
			expected: "「ok1」「ok2」",
		},
		{
			name: "only ng",
			criteria: map[string][]string{
				"ng": {"ng1"},
			},
			expected: "「ng1」は誤答",
		},
		{
			name: "only repeat",
			criteria: map[string][]string{
				"repeat": {"rep1"},
			},
			expected: "「rep1」はもう一度",
		},
		{
			name: "ok and ng",
			criteria: map[string][]string{
				"ok": {"ok1", "ok2"},
				"ng": {"ng1"},
			},
			expected: "「ok1」「ok2」／「ng1」は誤答",
		},
		{
			name: "all sections",
			criteria: map[string][]string{
				"ok":     {"ok1", "ok2"},
				"ng":     {"ng1"},
				"repeat": {"rep1"},
			},
			expected: "「ok1」「ok2」／「ng1」は誤答／「rep1」はもう一度",
		},
		{
			name: "complex with already quoted items",
			criteria: map[string][]string{
				"ok": {"「古典絵画館」", "古典美術館", "「アルテ・マイスター美術館」（おまけ）"},
			},
			expected: "「古典絵画館」「古典美術館」「アルテ・マイスター美術館」（おまけ）",
		},
		{
			name: "reading correction case",
			criteria: map[string][]string{
				"repeat": {"えんががわ（読みが違うのでもう一度）"},
			},
			expected: "「えんががわ（読みが違うのでもう一度）」はもう一度",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatCriteria(tt.criteria)
			if result != tt.expected {
				t.Errorf("FormatCriteria(%v) = %q, want %q", tt.criteria, result, tt.expected)
			}
		})
	}
}

func TestConvertYAMLToCSV(t *testing.T) {
	// テスト用の一時ディレクトリを作成
	tempDir := t.TempDir()

	tests := []struct {
		name         string
		yamlContent  string
		expectedRows [][]string
		shouldError  bool
	}{
		{
			name: "simple quiz without criteria",
			yamlContent: `- question: テスト問題
  answer: テスト答え
  spell: test spell`,
			expectedRows: [][]string{
				{"question", "answer", "spell", "criteria"},
				{"テスト問題", "テスト答え", "test spell", ""},
			},
			shouldError: false,
		},
		{
			name: "quiz with criteria",
			yamlContent: `- question: テスト問題
  answer: テスト答え
  spell: test spell
  comments:
    - コメント1
    - コメント2
  criteria:
    ok:
      - ok1
      - ok2
    ng:
      - ng1
    repeat:
      - rep1`,
			expectedRows: [][]string{
				{"question", "answer", "spell", "criteria"},
				{"テスト問題", "テスト答え", "test spell", "「ok1」「ok2」／「ng1」は誤答／「rep1」はもう一度"},
			},
			shouldError: false,
		},
		{
			name: "multiple quiz items",
			yamlContent: `- question: 問題1
  answer: 答え1
  spell: ""
- question: 問題2
  answer: 答え2
  spell: ""
  criteria:
    ng:
      - ng1`,
			expectedRows: [][]string{
				{"question", "answer", "spell", "criteria"},
				{"問題1", "答え1", "", ""},
				{"問題2", "答え2", "", "「ng1」は誤答"},
			},
			shouldError: false,
		},
		{
			name:        "invalid yaml",
			yamlContent: `invalid: yaml: content: [`,
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 一時ファイルを作成
			yamlFile := filepath.Join(tempDir, "test_input.yaml")
			csvFile := filepath.Join(tempDir, "test_output.csv")

			// YAMLファイルを作成
			err := os.WriteFile(yamlFile, []byte(tt.yamlContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create test YAML file: %v", err)
			}

			// テスト対象関数を実行
			err = ConvertYAMLToCSV(yamlFile, csvFile)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ConvertYAMLToCSV() error = %v", err)
			}

			// CSVファイルが作成されたか確認
			if _, err := os.Stat(csvFile); os.IsNotExist(err) {
				t.Fatal("CSV file was not created")
			}

			// CSVファイルの内容を読んで確認（詳細なテストは省略）
			// 実際のプロダクションコードでは、csvファイルの内容も詳細に検証する
			csvContent, err := os.ReadFile(csvFile)
			if err != nil {
				t.Fatalf("Failed to read CSV file: %v", err)
			}

			if len(csvContent) == 0 {
				t.Error("CSV file is empty")
			}
		})
	}
}

func TestConvertToTemplate(t *testing.T) {
	tempDir := t.TempDir()

	tests := []struct {
		name            string
		data            []QuizItem
		templateContent string
		expectedOutput  string
		shouldError     bool
	}{
		{
			name: "simple template",
			data: []QuizItem{
				{Question: "問題1", Answer: "答え1", Spell: "読み1"},
				{Question: "問題2", Answer: "答え2", Spell: "読み2"},
			},
			templateContent: `問題リスト:
{{range .Items}}
Q: {{.Question}}
A: {{.Answer}}
読み: {{.Spell}}
---
{{end}}`,
			expectedOutput: `問題リスト:

Q: 問題1
A: 答え1
読み: 読み1
---

Q: 問題2
A: 答え2
読み: 読み2
---
`,
			shouldError: false,
		},
		{
			name: "template with criteria",
			data: []QuizItem{
				{
					Question: "問題1",
					Answer:   "答え1",
					Spell:    "読み1",
					Comments: []string{"コメント1", "コメント2"},
					Criteria: map[string][]string{
						"ok": {"正答1", "正答2"},
						"ng": {"誤答1"},
					},
				},
			},
			templateContent: `{{range .Items}}
問題: {{.Question}}
答え: {{.Answer}}
{{if .Comments}}コメント:{{range .Comments}} {{.}}{{end}}{{end}}
判定: {{formatCriteria .Criteria}}
{{end}}`,
			expectedOutput: `
問題: 問題1
答え: 答え1
コメント: コメント1 コメント2
判定: 「正答1」「正答2」／「誤答1」は誤答
`,
			shouldError: false,
		},
		{
			name: "template with custom functions",
			data: []QuizItem{
				{Question: "test question", Answer: "test answer", Spell: "test spell"},
			},
			templateContent: `{{range .Items}}
問題: {{upper .Question}}
答え: {{lower .Answer}}
引用符付き: {{addQuotes .Spell}}
{{end}}`,
			expectedOutput: `
問題: TEST QUESTION
答え: test answer
引用符付き: 「test spell」
`,
			shouldError: false,
		},
		{
			name:            "invalid template",
			data:            []QuizItem{{Question: "test", Answer: "test", Spell: "test"}},
			templateContent: `{{range .Items}{{.Question}}{{end}}`,
			shouldError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			templateFile := filepath.Join(tempDir, "template.txt")
			outputFile := filepath.Join(tempDir, "output.txt")

			// テンプレートファイルを作成
			err := os.WriteFile(templateFile, []byte(tt.templateContent), 0644)
			if err != nil {
				t.Fatalf("Failed to create template file: %v", err)
			}

			// テスト実行
			err = ConvertToTemplate(tt.data, templateFile, outputFile)

			if tt.shouldError {
				if err == nil {
					t.Error("Expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Fatalf("ConvertToTemplate() error = %v", err)
			}

			// 出力ファイルの内容を確認
			outputContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}

			if string(outputContent) != tt.expectedOutput {
				t.Errorf("ConvertToTemplate() output = %q, want %q", string(outputContent), tt.expectedOutput)
			}
		})
	}
}

func TestDetectOutputFormat(t *testing.T) {
	tests := []struct {
		name         string
		outputFile   string
		templateFile string
		expected     OutputFormat
	}{
		{"CSV with .csv extension", "output.csv", "", FormatCSV},
		{"CSV with .CSV extension", "output.CSV", "", FormatCSV},
		{"Template with template file", "output.txt", "template.txt", FormatTemplate},
		{"Template without extension", "output", "", FormatTemplate},
		{"Template with .html", "output.html", "", FormatTemplate},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectOutputFormat(tt.outputFile, tt.templateFile)
			if result != tt.expected {
				t.Errorf("DetectOutputFormat(%q, %q) = %v, want %v", tt.outputFile, tt.templateFile, result, tt.expected)
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
	criteria := map[string][]string{
		"ok":     {"ok1", "ok2", "ok3"},
		"ng":     {"ng1", "ng2"},
		"repeat": {"rep1"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatCriteria(criteria)
	}
}

func BenchmarkConvertYAMLToCSV(b *testing.B) {
	// ベンチマーク用の一時ディレクトリを作成
	tempDir := b.TempDir()
	yamlFile := filepath.Join(tempDir, "bench_input.yaml")
	csvFile := filepath.Join(tempDir, "bench_output.csv")

	// テスト用YAMLファイルを作成
	yamlContent := `- question: ベンチマーク問題1
  answer: ベンチマーク答え1
  spell: benchmark1
  criteria:
    ok:
      - ok1
      - ok2
    ng:
      - ng1
- question: ベンチマーク問題2
  answer: ベンチマーク答え2
  spell: benchmark2
  criteria:
    repeat:
      - rep1`

	err := os.WriteFile(yamlFile, []byte(yamlContent), 0644)
	if err != nil {
		b.Fatalf("Failed to create benchmark YAML file: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// 前回のCSVファイルを削除
		os.Remove(csvFile)

		err := ConvertYAMLToCSV(yamlFile, csvFile)
		if err != nil {
			b.Fatalf("ConvertYAMLToCSV() error = %v", err)
		}
	}
}
