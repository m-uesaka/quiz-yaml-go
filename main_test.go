package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// exampleYAML はテストで入力に使うYAMLファイル（go testはパッケージのディレクトリで実行される）
const exampleYAML = "yaml/example.yaml"

// usageHeader はヘルプメッセージの先頭部分
const usageHeader = "使用法: "

// runCLI はrunを呼び出し，終了コードと標準出力・標準エラー出力の内容を返す
func runCLI(args ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = run(args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// writeFile はdir/nameにcontentを書き込み，そのパスを返す
func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	p := filepath.Join(dir, name)
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write fixture: %v", err)
	}
	return p
}

// readFile はpathの内容を返す
func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}
	return string(b)
}

func TestRun_ConvertByFormat(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		outputName  string
		wantMessage string
		wantContent string
	}{
		{name: "csv", format: "csv", outputName: "quiz.csv", wantMessage: "✅ CSV変換完了: ", wantContent: "ジョン・アタナソフ"},
		{name: "html", format: "html", outputName: "quiz.html", wantMessage: "✅ HTML変換完了: ", wantContent: "<html"},
		{name: "markdown", format: "markdown", outputName: "quiz.md", wantMessage: "✅ Markdown変換完了: ", wantContent: "ジョン・アタナソフ"},
		{name: "md", format: "md", outputName: "quiz.md", wantMessage: "✅ Markdown変換完了: ", wantContent: "ジョン・アタナソフ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			output := filepath.Join(t.TempDir(), tt.outputName)

			// Act
			code, stdout, stderr := runCLI("-input", exampleYAML, "-output", output, "-format", tt.format)

			// Assert
			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			wantStdout := tt.wantMessage + exampleYAML + " → " + output + "\n"
			if stdout != wantStdout {
				t.Errorf("stdout = %q, want %q", stdout, wantStdout)
			}
			if stderr != "" {
				t.Errorf("stderr = %q, want empty", stderr)
			}
			if got := readFile(t, output); !strings.Contains(got, tt.wantContent) {
				t.Errorf("output does not contain %q", tt.wantContent)
			}
		})
	}
}

func TestRun_ConvertDefaultFormatIsCSV(t *testing.T) {
	// Arrange
	output := filepath.Join(t.TempDir(), "quiz.csv")

	// Act
	code, stdout, stderr := runCLI("-input", exampleYAML, "-output", output)

	// Assert
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	wantStdout := "✅ CSV変換完了: " + exampleYAML + " → " + output + "\n"
	if stdout != wantStdout {
		t.Errorf("stdout = %q, want %q", stdout, wantStdout)
	}
}

func TestRun_ConvertWithTemplateTakesPrecedenceOverFormat(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	tmpl := writeFile(t, dir, "custom.txt", "{{range .Items}}[{{.Answer}}]\n{{end}}")
	output := filepath.Join(dir, "quiz.txt")

	// Act
	code, stdout, stderr := runCLI("-input", exampleYAML, "-output", output, "-template", tmpl, "-format", "unknown")

	// Assert
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	wantStdout := "✅ テンプレート変換完了: " + exampleYAML + " + " + tmpl + " → " + output + "\n"
	if stdout != wantStdout {
		t.Errorf("stdout = %q, want %q", stdout, wantStdout)
	}
	if got := readFile(t, output); !strings.HasPrefix(got, "[ジョン・アタナソフ]\n") {
		t.Errorf("output = %q, want prefix %q", got, "[ジョン・アタナソフ]\n")
	}
}

func TestRun_MarkdownDirToYAML(t *testing.T) {
	// Arrange
	mdDir := t.TempDir()
	writeFile(t, mdDir, "a.md", "---\ntitle: 自作問題-A\ndate: 2026-01-01\n---\n"+
		"## Question\n\n問題文\n\n## Answer\n\n答えA\n\n## Spell\n\n## Criteria\n\n### OK\n\n### NG\n\n### Close\n\n## Comment\n")
	output := filepath.Join(t.TempDir(), "quiz.yaml")

	// Act
	code, stdout, stderr := runCLI("-markdown-dir", mdDir, "-output", output)

	// Assert
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	wantStdout := "✅ Markdown集約完了: " + mdDir + " → " + output + "\n"
	if stdout != wantStdout {
		t.Errorf("stdout = %q, want %q", stdout, wantStdout)
	}
	if got := readFile(t, output); !strings.Contains(got, "答えA") {
		t.Errorf("output does not contain %q: %q", "答えA", got)
	}
}

func TestRun_ValidateSuccess(t *testing.T) {
	// Arrange
	args := []string{"-input", exampleYAML, "-validate"}

	// Act
	code, stdout, stderr := runCLI(args...)

	// Assert
	if code != 0 {
		t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
	}
	if !strings.Contains(stdout, "✅ バリデーション成功: ") {
		t.Errorf("stdout = %q, want validation success message", stdout)
	}
	if stderr != "" {
		t.Errorf("stderr = %q, want empty", stderr)
	}
}

func TestRun_Help(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{name: "-help", args: []string{"-help"}},
		{name: "-h", args: []string{"-h"}},
		{name: "--help", args: []string{"--help"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			code, stdout, stderr := runCLI(tt.args...)

			// Assert
			if code != 0 {
				t.Errorf("exit code = %d, want 0", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.HasPrefix(stderr, usageHeader) {
				t.Errorf("stderr does not start with %q: %q", usageHeader, stderr)
			}
			if !strings.Contains(stderr, "-markdown-dir") {
				t.Errorf("stderr does not list flags: %q", stderr)
			}
		})
	}
}

func TestRun_InvalidArguments(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		wantPrefix string
	}{
		{
			name:       "-markdown-dirと-inputの同時指定",
			args:       []string{"-markdown-dir", "md", "-input", exampleYAML, "-output", "out.yaml"},
			wantPrefix: "❌ エラー: -markdown-dirと-inputは同時に指定できません\n\n" + usageHeader,
		},
		{
			name:       "-markdown-dir指定時に-outputが未指定",
			args:       []string{"-markdown-dir", "md"},
			wantPrefix: "❌ エラー: 出力ファイルが指定されていません\n\n" + usageHeader,
		},
		{
			name:       "引数なし",
			args:       []string{},
			wantPrefix: "❌ エラー: 入力ファイルが指定されていません\n\n" + usageHeader,
		},
		{
			name:       "変換時に-outputが未指定",
			args:       []string{"-input", exampleYAML},
			wantPrefix: "❌ エラー: 出力ファイルが指定されていません\n\n" + usageHeader,
		},
		{
			name: "未知のフォーマット",
			args: []string{"-input", exampleYAML, "-output", "out.xml", "-format", "xml"},
			wantPrefix: "❌ エラー: サポートされていないフォーマットです: xml\n" +
				"サポートされているフォーマット: csv, html, markdown\n\n" + usageHeader,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Act
			code, stdout, stderr := runCLI(tt.args...)

			// Assert
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.HasPrefix(stderr, tt.wantPrefix) {
				t.Errorf("stderr = %q, want prefix %q", stderr, tt.wantPrefix)
			}
		})
	}
}

func TestRun_UndefinedFlag(t *testing.T) {
	// Act
	code, stdout, stderr := runCLI("-unknown-flag")

	// Assert
	if code != 2 {
		t.Errorf("exit code = %d, want 2", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	wantPrefix := "flag provided but not defined: -unknown-flag\n" + usageHeader
	if !strings.HasPrefix(stderr, wantPrefix) {
		t.Errorf("stderr = %q, want prefix %q", stderr, wantPrefix)
	}
}

func TestRun_ConversionErrors(t *testing.T) {
	tests := []struct {
		name string
		args func(dir string) []string
	}{
		{
			name: "入力ファイルが存在しない",
			args: func(dir string) []string {
				return []string{"-input", filepath.Join(dir, "missing.yaml"), "-output", filepath.Join(dir, "quiz.csv")}
			},
		},
		{
			name: "テンプレートファイルが存在しない",
			args: func(dir string) []string {
				return []string{"-input", exampleYAML, "-output", filepath.Join(dir, "quiz.txt"), "-template", filepath.Join(dir, "missing.tmpl")}
			},
		},
		{
			name: "Markdownディレクトリの集約に失敗",
			args: func(dir string) []string {
				writeFile(t, dir, "broken.md", "## Question\n\n問題文だけ\n")
				return []string{"-markdown-dir", dir, "-output", filepath.Join(dir, "quiz.yaml")}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			args := tt.args(t.TempDir())

			// Act
			code, stdout, stderr := runCLI(args...)

			// Assert
			if code != 1 {
				t.Errorf("exit code = %d, want 1", code)
			}
			if stdout != "" {
				t.Errorf("stdout = %q, want empty", stdout)
			}
			if !strings.HasPrefix(stderr, "❌ エラー: ") {
				t.Errorf("stderr = %q, want prefix %q", stderr, "❌ エラー: ")
			}
			if strings.Contains(stderr, usageHeader) {
				t.Errorf("stderr should not contain usage: %q", stderr)
			}
		})
	}
}

func TestRun_ValidateFailure(t *testing.T) {
	// Arrange
	input := writeFile(t, t.TempDir(), "invalid.yaml", "- question: 問題文\n  answer: \"\"\n")

	// Act
	code, stdout, stderr := runCLI("-input", input, "-validate")

	// Assert
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if !strings.Contains(stdout, "❌ バリデーション失敗: 1個のエラーが見つかりました\n") {
		t.Errorf("stdout = %q, want validation failure message", stdout)
	}
	if !strings.Contains(stderr, "  • 問題 1: 答え (answer) が空です\n") {
		t.Errorf("stderr = %q, want error detail", stderr)
	}
}
