package quiz_yaml_converter

import (
	"bytes"
	"flag"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// -update を指定すると，組み込みテンプレートのスナップショット（testdata/*.golden）を現在の出力で更新する．
//
//	go test ./quiz_yaml_converter -run TestBuiltinTemplateSnapshot -update
//
// -update はこのパッケージのテストにだけ定義されているため，パッケージを指定して実行する．
// `go test ./... -update` とすると，-update を定義していないほかのパッケージのテストが
// "flag provided but not defined" で失敗する．
var update = flag.Bool("update", false, "update golden files in testdata")

// 常に同じ時刻を返す関数を返す．テンプレート関数 now の時刻を固定するために使う．
func fixedClock(now time.Time) func() time.Time {
	return func() time.Time { return now }
}

// テンプレートを実行して出力を文字列で返すテスト用の補助関数．
func executeTemplate(t *testing.T, content string, htmlMode bool, items []QuizItem) (string, error) {
	t.Helper()
	tmpl, err := parseTemplate(content, htmlMode)
	if err != nil {
		t.Fatalf("parseTemplate(%q) error = %v", content, err)
	}
	var buf bytes.Buffer
	err = tmpl.Execute(&buf, TemplateData{Items: items})
	return buf.String(), err
}

// 個別アクセサ（.OK / .NG / .Repeat）とquoteAllで組み立てたテンプレートの出力が，
// formatCriteriaの出力と一致することを確認する．
// HTMLモードでは，どちらの出力もhtml/templateでエスケープした FormatCriteria() と一致することを確認する．
func TestAccessorsWithQuoteAllMatchFormatCriteria(t *testing.T) {
	// formatCriteriaと同じく，空でないセクションだけを「／」で区切って連結する．
	const accessorTemplate = `{{range .Items}}{{$sep := ""}}` +
		`{{with .OK}}{{quoteAll . ""}}{{$sep = "／"}}{{end}}` +
		`{{with .NG}}{{$sep}}{{quoteAll . "は誤答"}}{{$sep = "／"}}{{end}}` +
		`{{with .Repeat}}{{$sep}}{{quoteAll . "はもう一度"}}{{end}}` +
		`{{end}}`
	const formatCriteriaTemplate = `{{range .Items}}{{formatCriteria .Criteria}}{{end}}`

	criteriaCases := []struct {
		name     string
		criteria *Criteria
	}{
		{name: "nil criteria", criteria: nil},
		{name: "empty criteria", criteria: &Criteria{}},
		{name: "only ok", criteria: &Criteria{OK: []string{"ok1", "ok2"}}},
		{name: "only ng", criteria: &Criteria{NG: []string{"ng1"}}},
		{name: "only repeat", criteria: &Criteria{Repeat: []string{"rep1"}}},
		{name: "ok and ng", criteria: &Criteria{OK: []string{"ok1"}, NG: []string{"ng1", "ng2"}}},
		{name: "ok and repeat", criteria: &Criteria{OK: []string{"ok1"}, Repeat: []string{"rep1"}}},
		{name: "ng and repeat", criteria: &Criteria{NG: []string{"ng1"}, Repeat: []string{"rep1"}}},
		{
			name: "all sections",
			criteria: &Criteria{
				OK:     []string{"ok1", "ok2"},
				NG:     []string{"ng1"},
				Repeat: []string{"rep1"},
			},
		},
		{name: "empty lists are skipped", criteria: &Criteria{OK: []string{}, NG: []string{"ng1"}, Repeat: []string{}}},
		{
			name: "already quoted items",
			criteria: &Criteria{
				OK:     []string{"「古典絵画館」", "古典美術館", "「アルテ・マイスター美術館」（おまけ）"},
				Repeat: []string{"えんががわ（読みが違うのでもう一度）"},
			},
		},
		{
			name: "HTML special characters",
			criteria: &Criteria{
				OK:     []string{"A&B", "<b>太字</b>"},
				NG:     []string{`"引用"`, "it's"},
				Repeat: []string{"1+1>1"},
			},
		},
	}
	modes := []struct {
		name     string
		htmlMode bool
		// FormatCriteria() の出力を，そのモードでテンプレートが出力する形に変換する．
		escape func(t *testing.T, s string) string
	}{
		{name: "text", htmlMode: false, escape: func(_ *testing.T, s string) string { return s }},
		{name: "html", htmlMode: true, escape: htmlEscapeLikeTemplate},
	}

	for _, mode := range modes {
		for _, tt := range criteriaCases {
			t.Run(mode.name+"/"+tt.name, func(t *testing.T) {
				// Arrange
				items := []QuizItem{{Question: "q", Answer: "a", Criteria: tt.criteria}}

				// Act
				got, err := executeTemplate(t, accessorTemplate, mode.htmlMode, items)
				want, wantErr := executeTemplate(t, formatCriteriaTemplate, mode.htmlMode, items)
				expected := mode.escape(t, tt.criteria.Format())

				// Assert
				if err != nil || wantErr != nil {
					t.Fatalf("Execute() error = %v, %v", err, wantErr)
				}
				if got != want {
					t.Errorf("accessor template output = %q, formatCriteria output = %q", got, want)
				}
				if want != expected {
					t.Errorf("formatCriteria template output = %q, want %q", want, expected)
				}
			})
		}
	}
}

// 文字列をhtml/templateのHTMLテキストの文脈と同じ規則でエスケープする．
func htmlEscapeLikeTemplate(t *testing.T, s string) string {
	t.Helper()
	tmpl := htmltemplate.Must(htmltemplate.New("escape").Parse(`{{.}}`))
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, s); err != nil {
		t.Fatalf("failed to escape %q: %v", s, err)
	}
	return buf.String()
}

// quoteAllの第2引数（suffix）が空文字の場合と，リストがnil・空の場合の組み合わせを確認する．
func TestQuoteAllSuffixAndNilListCombinations(t *testing.T) {
	tests := []struct {
		name     string
		criteria *Criteria
		template string
		expected string
	}{
		{name: "nil criteria with empty suffix", criteria: nil, template: `{{quoteAll .NG ""}}`, expected: ""},
		{name: "nil criteria with suffix", criteria: nil, template: `{{quoteAll .NG "は誤答"}}`, expected: ""},
		{name: "missing key with empty suffix", criteria: &Criteria{OK: []string{"ok1"}}, template: `{{quoteAll .NG ""}}`, expected: ""},
		{name: "missing key with suffix", criteria: &Criteria{OK: []string{"ok1"}}, template: `{{quoteAll .Repeat "はもう一度"}}`, expected: ""},
		{name: "empty list with empty suffix", criteria: &Criteria{NG: []string{}}, template: `{{quoteAll .NG ""}}`, expected: ""},
		{name: "empty list with suffix", criteria: &Criteria{NG: []string{}}, template: `{{quoteAll .NG "は誤答"}}`, expected: ""},
		{name: "literal nil with empty suffix", criteria: nil, template: `{{quoteAll nil ""}}`, expected: ""},
		{name: "literal nil with suffix", criteria: nil, template: `{{quoteAll nil "は誤答"}}`, expected: ""},
		{name: "single item with empty suffix", criteria: &Criteria{NG: []string{"ng1"}}, template: `{{quoteAll .NG ""}}`, expected: "「ng1」"},
		{name: "single item with suffix", criteria: &Criteria{NG: []string{"ng1"}}, template: `{{quoteAll .NG "は誤答"}}`, expected: "「ng1」は誤答"},
		{name: "multiple items with empty suffix", criteria: &Criteria{OK: []string{"ok1", "「ok2」"}}, template: `{{quoteAll .OK ""}}`, expected: "「ok1」「ok2」"},
		{name: "multiple items with suffix", criteria: &Criteria{Repeat: []string{"rep1", "rep2"}}, template: `{{quoteAll .Repeat "はもう一度"}}`, expected: "「rep1」「rep2」はもう一度"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			items := []QuizItem{{Question: "q", Answer: "a", Criteria: tt.criteria}}

			template := `{{range .Items}}` + tt.template + `{{end}}`

			// Act
			got, err := executeTemplate(t, template, false, items)

			// Assert
			if err != nil {
				t.Fatalf("Execute(%q) error = %v", template, err)
			}
			if got != tt.expected {
				t.Errorf("Execute(%q) = %q, want %q", template, got, tt.expected)
			}
		})
	}
}

// quoteAllに不正な引数を渡した場合，引数の型・数に関するエラーでテンプレートの実行が失敗することを確認する．
func TestQuoteAllInvalidArguments(t *testing.T) {
	tests := []struct {
		name     string
		template string
		wantErr  string
	}{
		{name: "string instead of list", template: `{{range .Items}}{{quoteAll .Answer ""}}{{end}}`, wantErr: "wrong type for value; expected []string; got string"},
		{name: "criteria instead of list", template: `{{range .Items}}{{quoteAll .Criteria ""}}{{end}}`, wantErr: "wrong type for value; expected []string; got *quiz_yaml_converter.Criteria"},
		{name: "missing suffix", template: `{{range .Items}}{{quoteAll .OK}}{{end}}`, wantErr: "wrong number of args for quoteAll: want 2 got 1"},
		{name: "non-string suffix", template: `{{range .Items}}{{quoteAll .OK 1}}{{end}}`, wantErr: "expected string; found 1"},
		{name: "too many arguments", template: `{{range .Items}}{{quoteAll .OK "" ""}}{{end}}`, wantErr: "wrong number of args for quoteAll: want 2 got 3"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			items := []QuizItem{{Question: "q", Answer: "a", Criteria: &Criteria{OK: []string{"ok1"}}}}

			// Act
			_, err := executeTemplate(t, tt.template, false, items)

			// Assert
			if err == nil {
				t.Fatalf("Execute(%q) expected an error, got nil", tt.template)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Execute(%q) error = %q, want it to contain %q", tt.template, err.Error(), tt.wantErr)
			}
		})
	}
}

// 個別アクセサが通常の[]stringとして，組み込み関数やテンプレート関数と組み合わせられることを確認する．
func TestAccessorsWorkWithTemplateFuncs(t *testing.T) {
	criteria := &Criteria{
		OK:     []string{"ok1", "ok2", "ok3"},
		NG:     []string{"ng1"},
		Repeat: []string{},
	}

	tests := []struct {
		name     string
		criteria *Criteria
		template string
		expected string
	}{
		{name: "len", criteria: criteria, template: `{{len .OK}}/{{len .NG}}/{{len .Repeat}}`, expected: "3/1/0"},
		{name: "len of nil criteria", criteria: nil, template: `{{len .OK}}/{{len .NG}}/{{len .Repeat}}`, expected: "0/0/0"},
		{name: "len in pipeline", criteria: criteria, template: `{{.OK | len}}`, expected: "3"},
		{name: "join", criteria: criteria, template: `{{join .OK "、"}}`, expected: "ok1、ok2、ok3"},
		{name: "join of nil criteria", criteria: nil, template: `[{{join .OK "、"}}]`, expected: "[]"},
		{name: "range", criteria: criteria, template: `{{range .OK}}<{{.}}>{{end}}`, expected: "<ok1><ok2><ok3>"},
		{name: "range with index", criteria: criteria, template: `{{range $i, $v := .OK}}{{add $i 1}}:{{$v}} {{end}}`, expected: "1:ok1 2:ok2 3:ok3 "},
		{name: "range else on empty list", criteria: criteria, template: `{{range .Repeat}}{{.}}{{else}}なし{{end}}`, expected: "なし"},
		{name: "range else on nil criteria", criteria: nil, template: `{{range .NG}}{{.}}{{else}}なし{{end}}`, expected: "なし"},
		{name: "range with addQuotes", criteria: criteria, template: `{{range .OK}}{{addQuotes .}}{{end}}`, expected: "「ok1」「ok2」「ok3」"},
		{name: "index", criteria: criteria, template: `{{index .OK 0}},{{index .NG 0}}`, expected: "ok1,ng1"},
		{name: "slice", criteria: criteria, template: `{{join (slice .OK 1) ","}}`, expected: "ok2,ok3"},
		{name: "if and with", criteria: criteria, template: `{{if .OK}}OK{{end}}{{with .NG}}NG{{end}}{{if .Repeat}}R{{else}}-{{end}}`, expected: "OKNG-"},
		{name: "comparison with len", criteria: criteria, template: `{{if gt (len .OK) 2}}many{{end}}{{if eq (len .NG) 1}}one{{end}}`, expected: "manyone"},
		{name: "upper on each item", criteria: criteria, template: `{{range .NG}}{{upper .}}{{end}}`, expected: "NG1"},
		{name: "csvRow with joined list", criteria: criteria, template: `{{csvRow .Question (join .OK ",")}}`, expected: `q,"ok1,ok2,ok3"`},
		{name: "csvRow keeps formula as is", criteria: criteria, template: `{{csvRow (printf "=%s" .Question) "-1"}}`, expected: `=q,-1`},
		{name: "csvRowSafe escapes formula", criteria: criteria, template: `{{csvRowSafe (printf "=%s" .Question) "-1" "-A1" .Answer}}`, expected: `'=q,-1,'-A1,a`},
		{name: "csvFieldSafe escapes formula", criteria: criteria, template: `{{csvFieldSafe (printf "@%s,x" .Question)}}`, expected: `"'@q,x"`},
		{name: "printf", criteria: criteria, template: `{{printf "%q" .NG}}`, expected: `["ng1"]`},
		{name: "criteria fields", criteria: criteria, template: `{{join .Criteria.OK ","}}/{{len .Criteria.NG}}/{{len .Criteria.Repeat}}`, expected: "ok1,ok2,ok3/1/0"},
		{name: "criteria fields guarded by if", criteria: nil, template: `{{if .Criteria}}{{len .Criteria.OK}}{{else}}なし{{end}}`, expected: "なし"},
		{name: "formatCriteria of nil criteria", criteria: nil, template: `[{{formatCriteria .Criteria}}]`, expected: "[]"},
		{name: "HasCriteria", criteria: criteria, template: `{{if .HasCriteria}}あり{{end}}`, expected: "あり"},
		{name: "HasCriteria of nil criteria", criteria: nil, template: `{{if .HasCriteria}}あり{{else}}なし{{end}}`, expected: "なし"},
		{name: "HasCriteria of empty criteria", criteria: &Criteria{}, template: `{{if .HasCriteria}}あり{{else}}なし{{end}}`, expected: "なし"},
		{name: "IsEmpty of nil criteria", criteria: nil, template: `{{if .Criteria.IsEmpty}}空{{end}}`, expected: "空"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			items := []QuizItem{{Question: "q", Answer: "a", Criteria: tt.criteria}}
			template := `{{range .Items}}` + tt.template + `{{end}}`

			// Act
			got, err := executeTemplate(t, template, false, items)

			// Assert
			if err != nil {
				t.Fatalf("Execute(%q) error = %v", template, err)
			}
			if got != tt.expected {
				t.Errorf("Execute(%q) = %q, want %q", template, got, tt.expected)
			}
		})
	}
}

// テンプレート関数 now が，渡した時刻を「2006年01月02日 15:04:05」の形式で出力することを確認する．
// タイムゾーンは変換せず，渡した時刻のタイムゾーンのまま出力する．
func TestNowTemplateFunc(t *testing.T) {
	jst := time.FixedZone("JST", 9*60*60)

	tests := []struct {
		name     string
		now      time.Time
		expected string
	}{
		{
			name:     "1桁の月日と時分秒は0で埋める",
			now:      time.Date(2026, 1, 2, 3, 4, 5, 0, jst),
			expected: "2026年01月02日 03:04:05",
		},
		{
			name:     "2桁の月日と時分秒",
			now:      time.Date(2026, 12, 31, 23, 59, 59, 0, jst),
			expected: "2026年12月31日 23:59:59",
		},
		{
			// 1つ上のケースと同じ瞬間をUTCで表した時刻．JSTに変換されずUTCのまま出力されることを確認する．
			name:     "同じ瞬間でもUTCの時刻はUTCのまま出力する",
			now:      time.Date(2026, 12, 31, 23, 59, 59, 0, jst).UTC(),
			expected: "2026年12月31日 14:59:59",
		},
		{
			name:     "秒未満は切り捨てる",
			now:      time.Date(2026, 6, 15, 12, 30, 45, 999999999, time.UTC),
			expected: "2026年06月15日 12:30:45",
		},
	}

	for _, tt := range tests {
		for _, htmlMode := range []bool{false, true} {
			mode := "text"
			if htmlMode {
				mode = "html"
			}
			t.Run(tt.name+"/"+mode, func(t *testing.T) {
				// Arrange
				tmpl, err := parseTemplateWithClock("{{now}}", htmlMode, fixedClock(tt.now))
				if err != nil {
					t.Fatalf("parseTemplateWithClock() error = %v", err)
				}
				var buf bytes.Buffer

				// Act
				err = tmpl.Execute(&buf, TemplateData{})

				// Assert
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if got := buf.String(); got != tt.expected {
					t.Errorf("now (htmlMode=%v) = %q, want %q", htmlMode, got, tt.expected)
				}
			})
		}
	}
}

// 組み込みテンプレートの出力をtestdata/*.goldenと比較するスナップショットテスト．
// 生成日時（now）は実行ごとに変わるため，タイムゾーンを含めて固定した時刻でテンプレートを実行する．
// （ConvertToTemplate では時刻を指定できないため，読み込み・HTMLモードの判定・実行を同じ手順で行う．）
func TestBuiltinTemplateSnapshot(t *testing.T) {
	clock := fixedClock(time.Date(2026, 1, 2, 3, 4, 5, 0, time.FixedZone("JST", 9*60*60)))

	tests := []struct {
		name       string
		template   string
		outputName string
		golden     string
	}{
		{name: "html", template: "quiz_template.html", outputName: "output.html", golden: "quiz_template.html.golden"},
		{name: "markdown", template: "quiz_template.md", outputName: "output.md", golden: "quiz_template.md.golden"},
		{name: "csv", template: "quiz_template.csv", outputName: "output.csv", golden: "quiz_template.csv.golden"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			data, err := LoadYAMLData(filepath.Join("testdata", "quiz.yaml"))
			if err != nil {
				t.Fatalf("LoadYAMLData() error = %v", err)
			}
			templateFile := filepath.Join("..", "templates", tt.template)
			content, err := os.ReadFile(templateFile)
			if err != nil {
				t.Fatalf("Failed to read template file: %v", err)
			}
			goldenFile := filepath.Join("testdata", tt.golden)
			htmlMode := isHTMLPath(templateFile) || isHTMLPath(tt.outputName)

			// Act
			tmpl, err := parseTemplateWithClock(string(content), htmlMode, clock)
			if err != nil {
				t.Fatalf("parseTemplateWithClock() error = %v", err)
			}
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, TemplateData{Items: data}); err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			got := buf.String()

			// Assert
			if *update {
				if err := os.WriteFile(goldenFile, []byte(got), 0o644); err != nil {
					t.Fatalf("Failed to update golden file: %v", err)
				}
			}
			want, err := os.ReadFile(goldenFile)
			if err != nil {
				t.Fatalf("Failed to read golden file (run with -update to create it): %v", err)
			}
			if got != string(want) {
				t.Errorf("output of %s does not match %s (run with -update to accept the change)\ngot:\n%s\nwant:\n%s", tt.template, goldenFile, got, string(want))
			}
		})
	}
}

// 中身が空の判定基準（criteria: {} など）を持つ問題で，組み込みテンプレートが空の判定行を出力しないことを確認する．
func TestBuiltinTemplateSkipsEmptyCriteria(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		htmlMode bool
		absent   string
	}{
		{name: "markdown", path: "../templates/quiz_template.md", htmlMode: false, absent: "**Criteria:**"},
		{name: "html", path: "../templates/quiz_template.html", htmlMode: true, absent: "判定:"},
	}

	for _, tt := range tests {
		for _, criteria := range []*Criteria{{}, {OK: []string{}, NG: []string{}, Repeat: []string{}}} {
			t.Run(tt.name, func(t *testing.T) {
				// Arrange
				content, err := os.ReadFile(tt.path)
				if err != nil {
					t.Fatalf("failed to read template: %v", err)
				}
				items := []QuizItem{{Question: "q", Answer: "a", Criteria: criteria}}

				// Act
				got, err := executeTemplate(t, string(content), tt.htmlMode, items)

				// Assert
				if err != nil {
					t.Fatalf("Execute() error = %v", err)
				}
				if strings.Contains(got, tt.absent) {
					t.Errorf("output contains %q for empty criteria %+v:\n%s", tt.absent, criteria, got)
				}
			})
		}
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
					Criteria: &Criteria{
						OK: []string{"正答1", "正答2"},
						NG: []string{"誤答1"},
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
					Criteria: &Criteria{
						OK:     []string{"正答1", "正答2"},
						NG:     []string{"誤答1"},
						Repeat: []string{"もう一度1"},
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
		Criteria: &Criteria{OK: []string{"A&B <b>"}},
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

// nl2brが，各種の改行を <br> に置き換えて値を1行にまとめることを確認する．
func TestNl2br(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{name: "empty", input: "", expected: ""},
		{name: "no newline", input: "改行なし", expected: "改行なし"},
		{name: "LF", input: "1行目\n2行目", expected: "1行目<br>2行目"},
		{name: "CRLF", input: "1行目\r\n2行目", expected: "1行目<br>2行目"},
		{name: "CR", input: "1行目\r2行目", expected: "1行目<br>2行目"},
		{name: "blank line", input: "1行目\n\n3行目", expected: "1行目<br><br>3行目"},
		{name: "trailing newline", input: "1行目\n", expected: "1行目<br>"},
		{name: "markdown block markers", input: "# 見出し\n- リスト\n> 引用", expected: "# 見出し<br>- リスト<br>> 引用"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			items := []QuizItem{{Question: tt.input}}

			// Act
			got, err := executeTemplate(t, `{{range .Items}}{{nl2br .Question}}{{end}}`, false, items)

			// Assert
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("nl2br(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

// 組み込みのMarkdownテンプレートで，改行を含む値が1行に収まり，
// 2行目以降が見出し・リスト・引用などとして解釈される行にならないことを確認する．
func TestBuiltinMarkdownTemplateKeepsMultilineValuesOnOneLine(t *testing.T) {
	const multiline = "1行目\n# 見出し\n- リスト\n> 引用\n\n<b>空行</b>のあと"
	const wantValue = "1行目<br># 見出し<br>- リスト<br>&gt; 引用<br><br>&lt;b&gt;空行&lt;/b&gt;のあと"

	tests := []struct {
		name     string
		item     QuizItem
		wantLine string
	}{
		{
			name:     "question",
			item:     QuizItem{Question: multiline, Answer: "a"},
			wantLine: "**Q:** " + wantValue,
		},
		{
			name:     "answer",
			item:     QuizItem{Question: "q", Answer: multiline},
			wantLine: "**Answer:** " + wantValue,
		},
		{
			name:     "comments",
			item:     QuizItem{Question: "q", Answer: "a", Comments: []string{multiline}},
			wantLine: "**Comments:** " + wantValue,
		},
		{
			name:     "criteria",
			item:     QuizItem{Question: "q", Answer: "a", Criteria: &Criteria{NG: []string{multiline}}},
			wantLine: "**Criteria:** 「" + wantValue + "」は誤答",
		},
		{
			name:     "CRLF",
			item:     QuizItem{Question: "1行目\r\n2行目", Answer: "a"},
			wantLine: "**Q:** 1行目<br>2行目",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			content, err := os.ReadFile(filepath.Join("..", "templates", "quiz_template.md"))
			if err != nil {
				t.Fatalf("failed to read template: %v", err)
			}

			// Act
			got, err := executeTemplate(t, string(content), false, []QuizItem{tt.item})

			// Assert
			if err != nil {
				t.Fatalf("Execute() error = %v", err)
			}
			lines := strings.Split(got, "\n")
			found := false
			for _, line := range lines {
				if line == tt.wantLine {
					found = true
				}
				for _, marker := range []string{"# 見出し", "- リスト", "&gt; 引用", "&lt;b&gt;空行", "2行目"} {
					if strings.HasPrefix(line, marker) {
						t.Errorf("output has a line starting with %q:\n%s", marker, got)
					}
				}
				if strings.Contains(line, "\r") {
					t.Errorf("output has a line containing CR: %q", line)
				}
			}
			if !found {
				t.Errorf("output does not contain the line %q:\n%s", tt.wantLine, got)
			}
		})
	}
}

func TestBuiltinMarkdownTemplateEscapesHTML(t *testing.T) {
	tempDir := t.TempDir()
	outputFile := filepath.Join(tempDir, "output.md")
	data := []QuizItem{{
		Question: `<img src=x onerror=alert(1)>`,
		Answer:   "<b>答え</b>",
		Comments: []string{"<i>c</i>"},
		Criteria: &Criteria{OK: []string{"<u>別解</u>"}},
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

// テンプレートファイルを読み込めない場合と，テンプレートとして解析できない場合のエラーメッセージを確認する．
func TestConvertToTemplateReadAndParseErrors(t *testing.T) {
	tests := []struct {
		name       string
		template   *string // nil の場合はテンプレートファイルを作らない
		wantPrefix string
	}{
		{name: "missing template file", template: nil, wantPrefix: "テンプレートファイルの読み込みに失敗しました: "},
		{name: "invalid template", template: ptr("{{range .Items}}"), wantPrefix: "テンプレートの解析に失敗しました: "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			templatePath := filepath.Join(dir, "template.txt")
			if tt.template != nil {
				if err := os.WriteFile(templatePath, []byte(*tt.template), 0o644); err != nil {
					t.Fatalf("failed to write template: %v", err)
				}
			}
			output := filepath.Join(dir, "out.txt")

			// Act
			err := ConvertToTemplate([]QuizItem{{Question: "問題", Answer: "答え"}}, templatePath, output)

			// Assert
			if err == nil || !strings.HasPrefix(err.Error(), tt.wantPrefix) {
				t.Errorf("ConvertToTemplate() error = %v, want prefix %q", err, tt.wantPrefix)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Errorf("output file should not be written: stat error = %v", statErr)
			}
		})
	}
}

func ptr(s string) *string { return &s }
