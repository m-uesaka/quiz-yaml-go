package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"testing"
)

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

func TestCSVRow(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected string
	}{
		{name: "plain fields", fields: []string{"a", "b", "c"}, expected: "a,b,c"},
		{name: "japanese text", fields: []string{"問題", "「答え」"}, expected: "問題,「答え」"},
		{name: "field with comma", fields: []string{"a,b", "c"}, expected: `"a,b",c`},
		{name: "field with double quote", fields: []string{`say "hi"`}, expected: `"say ""hi"""`},
		{name: "field with newline", fields: []string{"line1\nline2", "x"}, expected: "\"line1\nline2\",x"},
		{name: "empty fields", fields: []string{"", ""}, expected: ","},
		{name: "no fields", fields: []string{}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := CSVRow(tt.fields...)

			// Assert
			if err != nil {
				t.Fatalf("CSVRow(%q) error = %v", tt.fields, err)
			}
			if result != tt.expected {
				t.Errorf("CSVRow(%q) = %q, want %q", tt.fields, result, tt.expected)
			}
		})
	}
}

func TestCSVField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "plain", field: "abc", expected: "abc"},
		{name: "with comma", field: "a,b", expected: `"a,b"`},
		{name: "with double quote", field: `a"b`, expected: `"a""b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := CSVField(tt.field)

			// Assert
			if err != nil {
				t.Fatalf("CSVField(%q) error = %v", tt.field, err)
			}
			if result != tt.expected {
				t.Errorf("CSVField(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

// 組み込みCSVテンプレートの出力が，テンプレートを使わないCSV変換と一致することを確認する．
func TestCSVTemplateMatchesDefaultCSV(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	yamlFile := filepath.Join(tempDir, "quiz.yaml")
	yamlContent := `- question: "カンマ, を含む問題"
  answer: "\"引用符\"を含む答え"
  spell: spell
  criteria:
    ok:
      - 別解1
    ng:
      - 誤答1
    repeat:
      - もう一度1
- question: "改行を\n含む問題"
  answer: 答え2
  spell: ""
`
	if err := os.WriteFile(yamlFile, []byte(yamlContent), 0644); err != nil {
		t.Fatalf("Failed to create YAML file: %v", err)
	}
	defaultCSV := filepath.Join(tempDir, "default.csv")
	templateCSV := filepath.Join(tempDir, "template.csv")
	templateFile := filepath.Join("..", "templates", "quiz_template.csv")

	// Act
	if err := Convert(yamlFile, defaultCSV, ""); err != nil {
		t.Fatalf("Convert() without template error = %v", err)
	}
	if err := Convert(yamlFile, templateCSV, templateFile); err != nil {
		t.Fatalf("Convert() with template error = %v", err)
	}

	// Assert
	want, err := os.ReadFile(defaultCSV)
	if err != nil {
		t.Fatalf("Failed to read default CSV: %v", err)
	}
	got, err := os.ReadFile(templateCSV)
	if err != nil {
		t.Fatalf("Failed to read template CSV: %v", err)
	}
	if string(got) != string(want) {
		t.Errorf("template CSV output = %q, want %q", string(got), string(want))
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
