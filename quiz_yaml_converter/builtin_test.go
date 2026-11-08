package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// 組み込みテンプレートが，カレントディレクトリに関係なく（templates/ が見えない場所でも）使えることを確認する．
func TestConvertWithBuiltinTemplate(t *testing.T) {
	tests := []struct {
		name       string
		builtin    BuiltinTemplate
		outputName string
		want       []string
	}{
		{name: "csv", builtin: BuiltinCSV, outputName: "quiz.csv", want: []string{"question,answer,spell,criteria\n", "問題1,答え1,,「別解1」\n"}},
		{name: "html", builtin: BuiltinHTML, outputName: "quiz.html", want: []string{"<html", "&lt;b&gt;"}},
		{name: "markdown", builtin: BuiltinMarkdown, outputName: "quiz.md", want: []string{"答え1", "別解1"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeTestFile(t, dir, "quiz.yaml", "- question: 問題1\n  answer: 答え1\n  criteria:\n    ok: [別解1]\n- question: <b>問題2</b>\n  answer: 答え2\n")
			output := filepath.Join(dir, tt.outputName)
			t.Chdir(t.TempDir())

			// Act
			warnings, err := ConvertWithBuiltinTemplateWithWarnings(input, output, tt.builtin)

			// Assert
			if err != nil {
				t.Fatalf("ConvertWithBuiltinTemplateWithWarnings() error = %v", err)
			}
			if len(warnings) != 0 {
				t.Errorf("warnings = %+v, want none", warnings)
			}
			got, err := os.ReadFile(output)
			if err != nil {
				t.Fatalf("failed to read output: %v", err)
			}
			for _, w := range tt.want {
				if !strings.Contains(string(got), w) {
					t.Errorf("output does not contain %q:\n%s", w, got)
				}
			}
		})
	}
}

// 組み込みテンプレートでの変換が失敗した場合にエラーを返し，出力ファイルを作らないことを確認する．
func TestConvertWithBuiltinTemplateErrors(t *testing.T) {
	tests := []struct {
		name    string
		builtin BuiltinTemplate
		// outputはフィクスチャ用のディレクトリを受け取り，出力ファイルのパスを返す
		output  func(dir string) string
		wantErr string
	}{
		{
			name:    "unsupported builtin template",
			builtin: BuiltinTemplate("json"),
			output:  func(dir string) string { return filepath.Join(dir, "quiz.json") },
			wantErr: `unsupported builtin template: "json"`,
		},
		{
			name:    "csv into nonexistent directory",
			builtin: BuiltinCSV,
			output:  func(dir string) string { return filepath.Join(dir, "missing", "quiz.csv") },
			wantErr: "failed to write output file",
		},
		{
			name:    "html into nonexistent directory",
			builtin: BuiltinHTML,
			output:  func(dir string) string { return filepath.Join(dir, "missing", "quiz.html") },
			wantErr: "failed to write output file",
		},
		{
			name:    "markdown into nonexistent directory",
			builtin: BuiltinMarkdown,
			output:  func(dir string) string { return filepath.Join(dir, "missing", "quiz.md") },
			wantErr: "failed to write output file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeTestFile(t, dir, "quiz.yaml", "- question: 問題\n  answer: 答え\n")
			output := tt.output(dir)

			// Act
			err := ConvertWithBuiltinTemplate(input, output, tt.builtin)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("ConvertWithBuiltinTemplate() error = %v, want it to contain %q", err, tt.wantErr)
			}
			if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
				t.Errorf("output file should not be created: stat error = %v", statErr)
			}
		})
	}
}

// 出力ファイルに書き込めない場合に，各変換関数がエラーを返すことを確認する．
func TestConvertWriteError(t *testing.T) {
	tests := []struct {
		name string
		// convertは入力ファイル，出力ファイル，テンプレートファイルのパスを受け取り，変換する
		convert    func(input, output, templateFile string) error
		outputName string
	}{
		{
			name:       "ConvertYAMLToCSV",
			convert:    func(input, output, _ string) error { return ConvertYAMLToCSV(input, output) },
			outputName: "quiz.csv",
		},
		{
			name:       "Convert without template",
			convert:    func(input, output, _ string) error { return Convert(input, output, "") },
			outputName: "quiz.csv",
		},
		{
			name:       "Convert with template",
			convert:    Convert,
			outputName: "quiz.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeTestFile(t, dir, "quiz.yaml", "- question: 問題\n  answer: 答え\n")
			templateFile := writeTestFile(t, dir, "quiz.tmpl", "{{range .Items}}{{.Answer}}\n{{end}}")
			output := filepath.Join(dir, "missing", tt.outputName)

			// Act
			err := tt.convert(input, output, templateFile)

			// Assert
			if err == nil || !strings.Contains(err.Error(), "failed to write output file") {
				t.Fatalf("error = %v, want it to contain %q", err, "failed to write output file")
			}
		})
	}
}
