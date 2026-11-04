package quiz_yaml_converter

import (
	"bytes"
	"flag"
	htmltemplate "html/template"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// -update を指定すると，組み込みテンプレートのスナップショット（testdata/*.golden）を現在の出力で更新する．
//
//	go test ./quiz_yaml_converter -run TestBuiltinTemplateSnapshot -update
//
// -update はこのパッケージのテストにだけ定義されているため，パッケージを指定して実行する．
// `go test ./... -update` とすると，-update を定義していないほかのパッケージのテストが
// "flag provided but not defined" で失敗する．
var update = flag.Bool("update", false, "update golden files in testdata")

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
		{name: "printf", criteria: criteria, template: `{{printf "%q" .NG}}`, expected: `["ng1"]`},
		{name: "criteria fields", criteria: criteria, template: `{{join .Criteria.OK ","}}/{{len .Criteria.NG}}/{{len .Criteria.Repeat}}`, expected: "ok1,ok2,ok3/1/0"},
		{name: "criteria fields guarded by if", criteria: nil, template: `{{if .Criteria}}{{len .Criteria.OK}}{{else}}なし{{end}}`, expected: "なし"},
		{name: "formatCriteria of nil criteria", criteria: nil, template: `[{{formatCriteria .Criteria}}]`, expected: "[]"},
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

// 組み込みテンプレートの出力をtestdata/*.goldenと比較するスナップショットテスト．
// 生成日時（now）は実行ごとに変わるため，比較前に固定の文字列に置き換える．
// 置き換えるのはHTMLテンプレートの「生成日時: 」の直後だけで，問題文などに含まれる同じ形式の日時は置き換えない．
// TODO(#32): nowの時刻を差し替えられるようになったら，固定の時刻を渡してこの置き換えをなくす．
func TestBuiltinTemplateSnapshot(t *testing.T) {
	nowPattern := regexp.MustCompile(`(<p>生成日時: )\d{4}年\d{2}月\d{2}日 \d{2}:\d{2}:\d{2}(</p>)`)
	const nowReplacement = "${1}YYYY年MM月DD日 hh:mm:ss${2}"

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
			yamlFile := filepath.Join("testdata", "quiz.yaml")
			templateFile := filepath.Join("..", "templates", tt.template)
			outputFile := filepath.Join(t.TempDir(), tt.outputName)
			goldenFile := filepath.Join("testdata", tt.golden)

			// Act
			if err := Convert(yamlFile, outputFile, templateFile); err != nil {
				t.Fatalf("Convert() error = %v", err)
			}
			output, err := os.ReadFile(outputFile)
			if err != nil {
				t.Fatalf("Failed to read output file: %v", err)
			}
			got := nowPattern.ReplaceAllString(string(output), nowReplacement)

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
