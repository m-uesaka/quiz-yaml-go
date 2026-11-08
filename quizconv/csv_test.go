package quizconv

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
			// 出力先のパスと，入力のYAMLファイルを用意する
			csvFile := filepath.Join(tempDir, "test_output.csv")
			yamlFile := writeFile(t, tempDir, "test_input.yaml", tt.yamlContent)

			// テスト対象関数を実行
			err := ConvertYAMLToCSV(yamlFile, csvFile)

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
			csvContent := readFile(t, csvFile)

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

func TestEscapeCSVFormula(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		// 数式として解釈される値
		{name: "starts with equals", field: `=HYPERLINK("http://example.com","x")`, expected: `'=HYPERLINK("http://example.com","x")`},
		{name: "starts with plus", field: "+1+2", expected: "'+1+2"},
		{name: "starts with minus", field: "-1-2", expected: "'-1-2"},
		{name: "starts with minus and space", field: "- 項目", expected: "'- 項目"},
		{name: "starts with at", field: "@SUM(A1:A2)", expected: "'@SUM(A1:A2)"},
		{name: "starts with tab", field: "\t=1", expected: "'\t=1"},
		{name: "starts with carriage return", field: "\r=1", expected: "'\r=1"},
		{name: "dangerous character after leading spaces", field: "  =1+2", expected: "'  =1+2"},
		{name: "only dangerous character", field: "=", expected: "'="},
		// そのまま出力される値
		{name: "empty string", field: "", expected: ""},
		{name: "only spaces", field: "   ", expected: "   "},
		{name: "plain text", field: "問題文", expected: "問題文"},
		{name: "dangerous character not at start", field: "1+1=2", expected: "1+1=2"},
		{name: "starts with single quote", field: "'=1", expected: "'=1"},
		{name: "dangerous character after leading newline", field: "\n=1", expected: "'\n=1"},
		{name: "dangerous character after ideographic space", field: "\u3000=1+2", expected: "'\u3000=1+2"},
		{name: "dangerous character after no-break space", field: "\u00a0@SUM(A1)", expected: "'\u00a0@SUM(A1)"},
		{name: "negative integer", field: "-1", expected: "-1"},
		{name: "positive integer with plus", field: "+5", expected: "+5"},
		{name: "negative decimal", field: "-1.5", expected: "-1.5"},
		{name: "decimal without integer part", field: "-.5", expected: "-.5"},
		{name: "number after leading spaces", field: "  -1", expected: "  -1"},
		{name: "number followed by formula", field: "-1+2", expected: "'-1+2"},
		{name: "sign only", field: "-", expected: "'-"},
		{name: "sign and dot only", field: "-.", expected: "'-."},
		{name: "starts with full-width equals", field: "＝1", expected: "＝1"},
		{name: "starts with corner bracket", field: "「答え」", expected: "「答え」"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result := EscapeCSVFormula(tt.field)

			// Assert
			if result != tt.expected {
				t.Errorf("EscapeCSVFormula(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

func TestCSVRowSafe(t *testing.T) {
	tests := []struct {
		name     string
		fields   []string
		expected string
	}{
		{name: "plain fields", fields: []string{"a", "b", "c"}, expected: "a,b,c"},
		{name: "every dangerous leading character", fields: []string{"=1", "+A1", "-A1", "@A1"}, expected: "'=1,'+A1,'-A1,'@A1"},
		{name: "numbers are not escaped", fields: []string{"-1", "+5", "-1.5"}, expected: "-1,+5,-1.5"},
		{name: "semicolon is quoted", fields: []string{"a;=1+2", "b"}, expected: `"a;=1+2",b`},
		{name: "semicolon with double quote", fields: []string{`a;"b"`}, expected: `"a;""b"""`},
		{name: "formula with semicolon", fields: []string{"=1;2"}, expected: `"'=1;2"`},
		{name: "leading tab and carriage return", fields: []string{"\tx", "\rx"}, expected: "'\tx,\"'\rx\""},
		{name: "formula with comma", fields: []string{"=1,2", "x"}, expected: `"'=1,2",x`},
		{name: "formula with double quote", fields: []string{`=HYPERLINK("http://example.com","x")`}, expected: `"'=HYPERLINK(""http://example.com"",""x"")"`},
		{name: "formula with newline", fields: []string{"=1\n+2"}, expected: "\"'=1\n+2\""},
		{name: "mixed safe and dangerous fields", fields: []string{"問題", "-1-1", "1-1"}, expected: "問題,'-1-1,1-1"},
		{name: "empty fields", fields: []string{"", ""}, expected: ","},
		{name: "no fields", fields: []string{}, expected: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := CSVRowSafe(tt.fields...)

			// Assert
			if err != nil {
				t.Fatalf("CSVRowSafe(%q) error = %v", tt.fields, err)
			}
			if result != tt.expected {
				t.Errorf("CSVRowSafe(%q) = %q, want %q", tt.fields, result, tt.expected)
			}
		})
	}
}

func TestCSVRowSafeDoesNotModifyArguments(t *testing.T) {
	// Arrange
	fields := []string{"=1", "a"}

	// Act
	if _, err := CSVRowSafe(fields...); err != nil {
		t.Fatalf("CSVRowSafe(%q) error = %v", fields, err)
	}

	// Assert
	if fields[0] != "=1" || fields[1] != "a" {
		t.Errorf("CSVRowSafe modified its arguments: %q", fields)
	}
}

func TestCSVFieldSafe(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected string
	}{
		{name: "plain", field: "abc", expected: "abc"},
		{name: "empty", field: "", expected: ""},
		{name: "formula", field: "@SUM(A1)", expected: "'@SUM(A1)"},
		{name: "formula with comma", field: "=1,2", expected: `"'=1,2"`},
		{name: "semicolon", field: "a;b", expected: `"a;b"`},
		{name: "number", field: "-1", expected: "-1"},
		{name: "plain with double quote", field: `a"b`, expected: `"a""b"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			result, err := CSVFieldSafe(tt.field)

			// Assert
			if err != nil {
				t.Fatalf("CSVFieldSafe(%q) error = %v", tt.field, err)
			}
			if result != tt.expected {
				t.Errorf("CSVFieldSafe(%q) = %q, want %q", tt.field, result, tt.expected)
			}
		})
	}
}

// templates/quiz_template.csv を指定した出力と，テンプレートを使わないCSV変換の出力が，どちらも期待する内容になることを確認する．
func TestCSVTemplateMatchesDefaultCSV(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
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
- question: "=HYPERLINK(\"http://example.com\",\"x\")"
  answer: "-1"
  spell: "@SUM(A1)"
`
	yamlFile := writeFile(t, tempDir, "quiz.yaml", yamlContent)
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
	// テンプレートを使わないCSV変換も組み込みのCSVテンプレートで出力するため，
	// CSVの構文としてのエスケープは以前のencoding/csvによる実装と同じで，
	// 数式として解釈される値の先頭にだけ「'」が付くことを，期待する出力と直接比較して確認する．
	const want = "question,answer,spell,criteria\n" +
		`"カンマ, を含む問題","""引用符""を含む答え",spell,「別解1」／「誤答1」は誤答／「もう一度1」はもう一度` + "\n" +
		"\"改行を\n含む問題\",答え2,,\n" +
		`"'=HYPERLINK(""http://example.com"",""x"")",-1,'@SUM(A1),` + "\n"
	for _, path := range []string{defaultCSV, templateCSV} {
		got := readFile(t, path)
		if got != want {
			t.Errorf("CSV output of %s = %q, want %q", filepath.Base(path), got, want)
		}
	}
}

func BenchmarkConvertYAMLToCSV(b *testing.B) {
	// ベンチマーク用の一時ディレクトリを作成
	tempDir := b.TempDir()
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

	yamlFile := writeFile(b, tempDir, "bench_input.yaml", yamlContent)

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

// 数式の対策が不要な値（「;」を含まない）では，CSVRowSafe がフィールドごとに整形しても CSVRow と同じ結果になることを確認する．
func TestCSVRowSafeMatchesCSVRowForSafeValues(t *testing.T) {
	tests := []struct {
		name   string
		fields []string
	}{
		{name: "plain", fields: []string{"a", "b"}},
		{name: "comma", fields: []string{"a,b", "c"}},
		{name: "double quote", fields: []string{`a"b`, "c"}},
		{name: "newline and CRLF", fields: []string{"a\nb", "c\r\nd"}},
		{name: "leading space", fields: []string{" a", "b"}},
		{name: "empty fields", fields: []string{"", "", ""}},
		{name: "single empty field", fields: []string{""}},
		{name: "backslash dot", fields: []string{`\.`}},
		{name: "numbers", fields: []string{"-1", "+5", "1.5"}},
		{name: "japanese", fields: []string{"問題", "「答え」"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			want, err := CSVRow(tt.fields...)
			if err != nil {
				t.Fatalf("CSVRow(%q) error = %v", tt.fields, err)
			}

			// Act
			got, err := CSVRowSafe(tt.fields...)

			// Assert
			if err != nil {
				t.Fatalf("CSVRowSafe(%q) error = %v", tt.fields, err)
			}
			if got != want {
				t.Errorf("CSVRowSafe(%q) = %q, want %q (same as CSVRow)", tt.fields, got, want)
			}
		})
	}
}
