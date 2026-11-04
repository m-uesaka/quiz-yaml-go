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

// testProg はテストでrunに渡すプログラム名
const testProg = "quiz-yaml-converter"

// usageHeader はヘルプメッセージの先頭部分
const usageHeader = "使用法: "

// wantUsage はtestProgで実行したときのヘルプメッセージ全体
const wantUsage = "使用法: quiz-yaml-converter [オプション]\n" +
	"\n" +
	"クイズYAMLファイルを指定されたフォーマットに変換します。\n" +
	"\n" +
	"オプション:\n" +
	"  -format string\n" +
	"    \t出力フォーマット（csv, html, markdown） (default \"csv\")\n" +
	"  -help\n" +
	"    \tヘルプを表示\n" +
	"  -input string\n" +
	"    \t入力するYAMLファイルのパス（-markdown-dir未指定時は必須）\n" +
	"  -markdown-dir string\n" +
	"    \t集約するMarkdownファイルが置かれたディレクトリのパス（指定時はMarkdown→YAML変換モードになる）\n" +
	"  -output string\n" +
	"    \t出力ファイルのパス（必須）\n" +
	"  -recursive\n" +
	"    \t-markdown-dir指定時，サブディレクトリも再帰的に辿るかどうか\n" +
	"  -template string\n" +
	"    \tテンプレートファイルのパス（formatに関係なく使用）\n" +
	"  -validate\n" +
	"    \tYAMLファイルのフォーマットをバリデーションのみ実行\n" +
	"\n" +
	"例:\n" +
	"  quiz-yaml-converter -input quiz.yaml -output quiz.csv\n" +
	"  quiz-yaml-converter -input quiz.yaml -output quiz.html -format html\n" +
	"  quiz-yaml-converter -input quiz.yaml -output quiz.md -template custom.tmpl\n" +
	"  quiz-yaml-converter -input quiz.yaml -output quiz.csv -template templates/quiz_template.csv\n" +
	"  quiz-yaml-converter -input quiz.yaml -validate\n" +
	"  quiz-yaml-converter -markdown-dir path/to/quiz -output quiz.yaml\n" +
	"  quiz-yaml-converter -markdown-dir path/to/quiz -recursive -output quiz.yaml\n"

// runCLI はrunを呼び出し，終了コードと標準出力・標準エラー出力の内容を返す
func runCLI(args ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = run(testProg, args, &outBuf, &errBuf)
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
			// ヘルプが1回だけ，全文どおりに表示されること
			if stderr != wantUsage {
				t.Errorf("stderr = %q, want %q", stderr, wantUsage)
			}
		})
	}
}

func TestRun_InvalidArguments(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantMsg string
	}{
		{
			name:    "-markdown-dirと-inputの同時指定",
			args:    []string{"-markdown-dir", "md", "-input", exampleYAML, "-output", "out.yaml"},
			wantMsg: "❌ エラー: -markdown-dirと-inputは同時に指定できません\n",
		},
		{
			name:    "-markdown-dir指定時に-outputが未指定",
			args:    []string{"-markdown-dir", "md"},
			wantMsg: "❌ エラー: 出力ファイルが指定されていません\n",
		},
		{
			name:    "引数なし",
			args:    []string{},
			wantMsg: "❌ エラー: 入力ファイルが指定されていません\n",
		},
		{
			name:    "変換時に-outputが未指定",
			args:    []string{"-input", exampleYAML},
			wantMsg: "❌ エラー: 出力ファイルが指定されていません\n",
		},
		{
			name: "未知のフォーマット",
			args: []string{"-input", exampleYAML, "-output", "out.xml", "-format", "xml"},
			wantMsg: "❌ エラー: サポートされていないフォーマットです: xml\n" +
				"サポートされているフォーマット: csv, html, markdown\n",
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
			// エラーメッセージ，空行，ヘルプの順に表示されること
			want := tt.wantMsg + "\n" + wantUsage
			if stderr != want {
				t.Errorf("stderr = %q, want %q", stderr, want)
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
	want := "flag provided but not defined: -unknown-flag\n" + wantUsage
	if stderr != want {
		t.Errorf("stderr = %q, want %q", stderr, want)
	}
}

func TestRun_ConversionErrors(t *testing.T) {
	tests := []struct {
		name string
		// argsはサブテストのtとフィクスチャ用のディレクトリを受け取り，runに渡す引数を返す
		args func(t *testing.T, dir string) []string
	}{
		{
			name: "入力ファイルが存在しない",
			args: func(_ *testing.T, dir string) []string {
				return []string{"-input", filepath.Join(dir, "missing.yaml"), "-output", filepath.Join(dir, "quiz.csv")}
			},
		},
		{
			name: "テンプレートファイルが存在しない",
			args: func(_ *testing.T, dir string) []string {
				return []string{"-input", exampleYAML, "-output", filepath.Join(dir, "quiz.txt"), "-template", filepath.Join(dir, "missing.tmpl")}
			},
		},
		{
			name: "Markdownディレクトリの集約に失敗",
			args: func(t *testing.T, dir string) []string {
				writeFile(t, dir, "broken.md", "## Question\n\n問題文だけ\n")
				return []string{"-markdown-dir", dir, "-output", filepath.Join(dir, "quiz.yaml")}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			args := tt.args(t, t.TempDir())

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
