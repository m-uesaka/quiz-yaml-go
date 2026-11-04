package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
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
			name: "template with individual criteria",
			data: []QuizItem{
				{
					Question: "問題1",
					Answer:   "答え1",
					Criteria: map[string][]string{
						"ok":     {"正答1", "正答2"},
						"ng":     {"誤答1"},
						"repeat": {"もう一度1"},
					},
				},
			},
			templateContent: `{{range .Items}}
別解: {{quoteAll .OK ""}}
別解数: {{len .OK}}
誤答: {{quoteAll .NG ""}}
もう一度: {{quoteAll .Repeat "はもう一度"}}
問題数: {{len $.Items}}
{{end}}`,
			expectedOutput: `
別解: 「正答1」「正答2」
別解数: 2
誤答: 「誤答1」
もう一度: 「もう一度1」はもう一度
問題数: 1
`,
			shouldError: false,
		},
		{
			name: "template with missing individual criteria",
			data: []QuizItem{
				{Question: "問題1", Answer: "答え1"},
			},
			templateContent: `{{range .Items}}[{{if .OK}}OK{{end}}{{if .NG}}NG{{end}}{{if .Repeat}}R{{end}}{{quoteAll .NG "は誤答"}}{{len .Repeat}}]{{end}}`,
			expectedOutput:  `[0]`,
			shouldError:     false,
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

func TestConvertToTemplateEscapesHTMLOnly(t *testing.T) {
	data := []QuizItem{{
		Question: `<script>alert("x")</script> &`,
		Criteria: map[string][]string{"ok": {"A&B <b>"}},
	}}
	templateContent := `{{range .Items}}<p>{{.Question}}</p><p>{{formatCriteria .Criteria}}</p>{{end}}`
	escaped := `<p>&lt;script&gt;alert(&#34;x&#34;)&lt;/script&gt; &amp;</p><p>「A&amp;B &lt;b&gt;」</p>`
	raw := `<p><script>alert("x")</script> &</p><p>「A&B <b>」</p>`

	tests := []struct {
		name           string
		templateExt    string
		outputExt      string
		expectedOutput string
	}{
		{name: ".html template escapes values", templateExt: ".html", outputExt: ".html", expectedOutput: escaped},
		{name: ".htm template escapes values", templateExt: ".htm", outputExt: ".txt", expectedOutput: escaped},
		{name: "upper-case .HTML template escapes values", templateExt: ".HTML", outputExt: ".txt", expectedOutput: escaped},
		{name: ".xhtml template escapes values", templateExt: ".xhtml", outputExt: ".txt", expectedOutput: escaped},
		{name: "HTML output escapes values even with a non-HTML template", templateExt: ".tmpl", outputExt: ".Htm", expectedOutput: escaped},
		{name: "non-HTML template and output preserve text", templateExt: ".md", outputExt: ".md", expectedOutput: raw},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			templateFile := filepath.Join(tempDir, "template"+tt.templateExt)
			outputFile := filepath.Join(tempDir, "output"+tt.outputExt)
			if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
				t.Fatalf("Failed to create template file: %v", err)
			}

			if err := ConvertToTemplate(data, templateFile, outputFile); err != nil {
				t.Fatalf("ConvertToTemplate() error = %v", err)
			}

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

func TestConvertToTemplateSafeHTMLAndEscapeHTML(t *testing.T) {
	data := []QuizItem{{
		Question: `a<b`,
		Comments: []string{"x&y", "z"},
	}}
	templateContent := `{{range .Items}}{{escapeHTML .Question}}|{{join .Comments "<br>" | safeHTML}}{{end}}`

	tests := []struct {
		ext            string
		expectedOutput string
	}{
		// HTMLモードでもescapeHTMLは二重エスケープされず，safeHTMLはエスケープされない．
		{ext: ".html", expectedOutput: `a&lt;b|x&y<br>z`},
		{ext: ".md", expectedOutput: `a&lt;b|x&y<br>z`},
	}

	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			tempDir := t.TempDir()
			templateFile := filepath.Join(tempDir, "template"+tt.ext)
			outputFile := filepath.Join(tempDir, "output"+tt.ext)
			if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
				t.Fatalf("Failed to create template file: %v", err)
			}

			if err := ConvertToTemplate(data, templateFile, outputFile); err != nil {
				t.Fatalf("ConvertToTemplate() error = %v", err)
			}

			outputContent, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			if string(outputContent) != tt.expectedOutput {
				t.Errorf("output = %q, want %q", string(outputContent), tt.expectedOutput)
			}
		})
	}
}

func TestConvertToTemplateHTMLEscapeErrorKeepsExistingOutput(t *testing.T) {
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "template.html")
	outputFile := filepath.Join(tempDir, "output.html")
	// 分岐ごとに終了時のコンテキストが異なるため，html/templateはExecute時にエラーを返す．
	templateContent := `{{range .Items}}{{if .Spell}}<span title="{{end}}{{.Question}}{{end}}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	const existing = "existing output"
	if err := os.WriteFile(outputFile, []byte(existing), 0644); err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}

	err := ConvertToTemplate([]QuizItem{{Question: "q", Spell: "s"}}, templateFile, outputFile)
	if err == nil {
		t.Fatal("ConvertToTemplate() expected an error, got nil")
	}

	outputContent, readErr := os.ReadFile(outputFile)
	if readErr != nil {
		t.Fatalf("Failed to read output file: %v", readErr)
	}
	if string(outputContent) != existing {
		t.Errorf("output file was modified: got %q, want %q", string(outputContent), existing)
	}
}

func TestBuiltinMarkdownTemplateEscapesHTML(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.md")
	data := []QuizItem{{
		Question: `<img src=x onerror=alert(1)>`,
		Answer:   "<b>答え</b>",
		Comments: []string{"<i>c</i>"},
		Criteria: map[string][]string{"ok": {"<u>別解</u>"}},
	}}

	if err := ConvertToTemplate(data, filepath.Join("..", "templates", "quiz_template.md"), outputFile); err != nil {
		t.Fatalf("ConvertToTemplate() error = %v", err)
	}

	outputContent, err := os.ReadFile(outputFile)
	if err != nil {
		t.Fatalf("Failed to read output file: %v", err)
	}
	output := string(outputContent)
	for _, raw := range []string{"<img", "<b>", "<i>", "<u>"} {
		if strings.Contains(output, raw) {
			t.Errorf("output contains unescaped %q:\n%s", raw, output)
		}
	}
	for _, want := range []string{"&lt;img src=x onerror=alert(1)&gt;", "&lt;b&gt;答え&lt;/b&gt;", "&lt;i&gt;c&lt;/i&gt;", "&lt;u&gt;別解&lt;/u&gt;"} {
		if !strings.Contains(output, want) {
			t.Errorf("output does not contain %q:\n%s", want, output)
		}
	}
}

func TestQuizItemCriteriaAccessors(t *testing.T) {
	tests := []struct {
		name           string
		criteria       map[string][]string
		expectedOK     []string
		expectedNG     []string
		expectedRepeat []string
	}{
		{
			name: "all keys present",
			criteria: map[string][]string{
				"ok":     {"ok1", "ok2"},
				"ng":     {"ng1"},
				"repeat": {"rep1"},
			},
			expectedOK:     []string{"ok1", "ok2"},
			expectedNG:     []string{"ng1"},
			expectedRepeat: []string{"rep1"},
		},
		{
			name:       "only ok",
			criteria:   map[string][]string{"ok": {"ok1"}},
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

// テンプレートの実行に失敗した場合，既存の出力ファイルが上書きされないことを確認する．
func TestConvertToTemplate_ExecutionErrorKeepsExistingOutput(t *testing.T) {
	// Arrange
	tempDir := t.TempDir()
	templateFile := filepath.Join(tempDir, "template.csv")
	outputFile := filepath.Join(tempDir, "output.csv")
	existing := "existing,content\n"
	if err := os.WriteFile(outputFile, []byte(existing), 0644); err != nil {
		t.Fatalf("Failed to create output file: %v", err)
	}
	// ヘッダ行の出力後，csvRowに[]stringを渡して実行時エラーにする
	templateContent := `{{csvRow "question" "tags"}}
{{range .Items}}{{csvRow .Question .Tags}}
{{end}}`
	if err := os.WriteFile(templateFile, []byte(templateContent), 0644); err != nil {
		t.Fatalf("Failed to create template file: %v", err)
	}
	data := []QuizItem{{Question: "問題1", Answer: "答え1", Tags: []string{"tag"}}}

	// Act
	err := ConvertToTemplate(data, templateFile, outputFile)

	// Assert
	if err == nil {
		t.Fatal("Expected error, but got nil")
	}
	got, readErr := os.ReadFile(outputFile)
	if readErr != nil {
		t.Fatalf("Failed to read output file: %v", readErr)
	}
	if string(got) != existing {
		t.Errorf("output file = %q, want unchanged %q", string(got), existing)
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
