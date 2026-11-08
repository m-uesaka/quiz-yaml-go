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

// csvHeader はCSV出力の先頭行
const csvHeader = "question,answer,spell,criteria\n"

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
	"  quiz-yaml-converter -input quiz.yaml -output quiz.md -format markdown\n" +
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

// 組み込みテンプレートはバイナリに埋め込まれているため，リポジトリの外をカレントディレクトリにしても変換できることを確認する．
func TestRun_ConvertByFormatOutsideRepository(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		outputName  string
		wantContent string
	}{
		{name: "csv", format: "csv", outputName: "quiz.csv", wantContent: "ジョン・アタナソフ"},
		{name: "html", format: "html", outputName: "quiz.html", wantContent: "<html"},
		{name: "markdown", format: "markdown", outputName: "quiz.md", wantContent: "ジョン・アタナソフ"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			input, err := filepath.Abs(exampleYAML)
			if err != nil {
				t.Fatalf("failed to get absolute path: %v", err)
			}
			dir := t.TempDir()
			t.Chdir(dir)

			// Act
			code, _, stderr := runCLI("-input", input, "-output", tt.outputName, "-format", tt.format)

			// Assert
			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if got := readFile(t, filepath.Join(dir, tt.outputName)); !strings.Contains(got, tt.wantContent) {
				t.Errorf("output does not contain %q", tt.wantContent)
			}
		})
	}
}

// -formatを省略した場合は，出力ファイルの拡張子が.csv（大文字・小文字を区別しない）であればCSVで出力することを確認する．
func TestRun_ConvertDefaultFormatIsCSV(t *testing.T) {
	tests := []struct {
		name       string
		outputName string
	}{
		{name: "拡張子が.csv", outputName: "quiz.csv"},
		{name: "拡張子が.CSV", outputName: "quiz.CSV"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			output := filepath.Join(t.TempDir(), tt.outputName)

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
			if got := readFile(t, output); !strings.HasPrefix(got, csvHeader) {
				t.Errorf("output = %q, want prefix %q", got, csvHeader)
			}
		})
	}
}

// -formatを明示した場合は，出力ファイルの拡張子に関係なく指定したフォーマットで出力することを確認する（#29）．
func TestRun_ConvertExplicitFormatIgnoresExtension(t *testing.T) {
	tests := []struct {
		name        string
		format      string
		outputName  string
		wantMessage string
		wantPrefix  string
	}{
		{name: "csvを.txtに出力", format: "csv", outputName: "quiz.txt", wantMessage: "✅ CSV変換完了: ", wantPrefix: csvHeader},
		{name: "csvを拡張子なしのファイルに出力", format: "csv", outputName: "quiz", wantMessage: "✅ CSV変換完了: ", wantPrefix: csvHeader},
		{name: "csvを.htmlに出力", format: "csv", outputName: "quiz.html", wantMessage: "✅ CSV変換完了: ", wantPrefix: csvHeader},
		{name: "htmlを.csvに出力", format: "html", outputName: "quiz.csv", wantMessage: "✅ HTML変換完了: ", wantPrefix: "<!DOCTYPE html>"},
		{name: "markdownを.txtに出力", format: "markdown", outputName: "quiz.txt", wantMessage: "✅ Markdown変換完了: ", wantPrefix: "# "},
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
			if got := readFile(t, output); !strings.HasPrefix(got, tt.wantPrefix) {
				t.Errorf("output = %q, want prefix %q", got, tt.wantPrefix)
			}
		})
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
		{
			name: "-format未指定で出力ファイルの拡張子が.html",
			args: []string{"-input", exampleYAML, "-output", "out.html"},
			wantMsg: "❌ エラー: 出力ファイルの拡張子が.csvではないため，出力フォーマットを判定できません: out.html\n" +
				"-formatまたは-templateを指定してください\n",
		},
		{
			name: "-format未指定で出力ファイルの拡張子が.txt",
			args: []string{"-input", exampleYAML, "-output", "out.txt"},
			wantMsg: "❌ エラー: 出力ファイルの拡張子が.csvではないため，出力フォーマットを判定できません: out.txt\n" +
				"-formatまたは-templateを指定してください\n",
		},
		{
			name: "-format未指定で出力ファイルに拡張子がない",
			args: []string{"-input", exampleYAML, "-output", "out"},
			wantMsg: "❌ エラー: 出力ファイルの拡張子が.csvではないため，出力フォーマットを判定できません: out\n" +
				"-formatまたは-templateを指定してください\n",
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
			name: "CSVの出力先ディレクトリが存在しない",
			args: func(_ *testing.T, dir string) []string {
				return []string{"-input", exampleYAML, "-output", filepath.Join(dir, "missing", "quiz.csv")}
			},
		},
		{
			name: "HTMLの出力先ディレクトリが存在しない",
			args: func(_ *testing.T, dir string) []string {
				return []string{"-input", exampleYAML, "-output", filepath.Join(dir, "missing", "quiz.html"), "-format", "html"}
			},
		},
		{
			name: "Markdownの出力先ディレクトリが存在しない",
			args: func(_ *testing.T, dir string) []string {
				return []string{"-input", exampleYAML, "-output", filepath.Join(dir, "missing", "quiz.md"), "-format", "markdown"}
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

// 変換時にバリデーションの問題があっても，警告を表示して変換を続け，終了コード0になることを確認する．
func TestRun_ConvertWithWarnings(t *testing.T) {
	const content = "- question: 問題文1\n  answer: \"\"\n- question: \"\"\n  answer: 答え2\n  tags: [\"\"]\n"
	const wantStderr = "⚠️ 警告: 問題 1: 答え (answer) が空です\n" +
		"⚠️ 警告: 問題 2: 問題文 (question) が空です\n" +
		"⚠️ 警告: 問題 2: tags[0] が空です\n"

	tests := []struct {
		name        string
		outputName  string
		extraArgs   func(t *testing.T, dir string) []string
		wantMessage string
		wantContent string
	}{
		{
			name:        "csv",
			outputName:  "quiz.csv",
			extraArgs:   func(_ *testing.T, _ string) []string { return []string{"-format", "csv"} },
			wantMessage: "✅ CSV変換完了: ",
			wantContent: "問題文1",
		},
		{
			name:        "markdown",
			outputName:  "quiz.md",
			extraArgs:   func(_ *testing.T, _ string) []string { return []string{"-format", "markdown"} },
			wantMessage: "✅ Markdown変換完了: ",
			wantContent: "答え2",
		},
		{
			name:       "template",
			outputName: "quiz.txt",
			extraArgs: func(t *testing.T, dir string) []string {
				return []string{"-template", writeFile(t, dir, "custom.tmpl", "{{range .Items}}[{{.Answer}}]{{end}}")}
			},
			wantContent: "[][答え2]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			input := writeFile(t, dir, "warn.yaml", content)
			output := filepath.Join(dir, tt.outputName)
			args := append([]string{"-input", input, "-output", output}, tt.extraArgs(t, dir)...)

			// Act
			code, stdout, stderr := runCLI(args...)

			// Assert
			if code != 0 {
				t.Fatalf("exit code = %d, want 0 (stderr: %s)", code, stderr)
			}
			if stderr != wantStderr {
				t.Errorf("stderr = %q, want %q", stderr, wantStderr)
			}
			if !strings.HasPrefix(stdout, "✅ ") || !strings.Contains(stdout, tt.wantMessage) {
				t.Errorf("stdout = %q, want success message %q", stdout, tt.wantMessage)
			}
			if got := readFile(t, output); !strings.Contains(got, tt.wantContent) {
				t.Errorf("output = %q, want it to contain %q", got, tt.wantContent)
			}
		})
	}
}

// criteria の不正なキーは警告ではなく，従来どおり変換エラーになることを確認する．
func TestRun_ConvertUnknownCriteriaKeyIsError(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	input := writeFile(t, dir, "invalid.yaml", "- question: 問題文\n  answer: \"\"\n  criteria:\n    maybe: [m1]\n")
	output := filepath.Join(dir, "quiz.csv")

	// Act
	code, stdout, stderr := runCLI("-input", input, "-output", output)

	// Assert
	if code != 1 {
		t.Errorf("exit code = %d, want 1", code)
	}
	if stdout != "" {
		t.Errorf("stdout = %q, want empty", stdout)
	}
	if !strings.HasPrefix(stderr, "❌ エラー: ") || !strings.Contains(stderr, "不正なcriteriaキー: 'maybe'") {
		t.Errorf("stderr = %q, want unknown criteria key error", stderr)
	}
	if strings.Contains(stderr, "⚠️ 警告: ") {
		t.Errorf("stderr should not contain warnings: %q", stderr)
	}
}

// Markdownの集約時にバリデーションの問題があっても，警告を表示してYAMLを書き出し，終了コード0になることを確認する．
func TestRun_MarkdownDirToYAMLWithWarnings(t *testing.T) {
	// Arrange
	mdDir := t.TempDir()
	writeFile(t, mdDir, "a.md", "---\ntitle: 自作問題-A\n---\n## Question\n\n問題文A\n\n## Answer\n\n答えA\n")
	writeFile(t, mdDir, "b.md", "---\ntitle: 自作問題-B\n---\n## Question\n\n問題文B\n\n## Answer\n\n## Spell\n\n")
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
	wantStderr := "⚠️ 警告: 問題 2 (" + filepath.Join(mdDir, "b.md") + "): 答え (answer) が空です\n"
	if stderr != wantStderr {
		t.Errorf("stderr = %q, want %q", stderr, wantStderr)
	}
	if got := readFile(t, output); !strings.Contains(got, "問題文B") {
		t.Errorf("output does not contain %q: %q", "問題文B", got)
	}
}

// 一部のMarkdownファイルの変換に失敗した場合も，変換できたファイルの警告をエラーと一緒に表示することを確認する．
func TestRun_MarkdownDirToYAMLWarningsWithParseError(t *testing.T) {
	// Arrange
	mdDir := t.TempDir()
	writeFile(t, mdDir, "a.md", "frontmatterなし\n")
	writeFile(t, mdDir, "b.md", "---\ntitle: 自作問題-B\n---\n## Question\n\n問題文B\n\n## Answer\n\n")
	output := filepath.Join(t.TempDir(), "quiz.yaml")

	// Act
	code, _, stderr := runCLI("-markdown-dir", mdDir, "-output", output)

	// Assert
	if code != 1 {
		t.Fatalf("exit code = %d, want 1 (stderr: %s)", code, stderr)
	}
	wantWarning := "⚠️ 警告: 問題 1 (" + filepath.Join(mdDir, "b.md") + "): 答え (answer) が空です\n"
	if !strings.Contains(stderr, wantWarning) {
		t.Errorf("stderr = %q, want it to contain %q", stderr, wantWarning)
	}
	if !strings.Contains(stderr, "❌ エラー: ") || !strings.Contains(stderr, filepath.Join(mdDir, "a.md")) {
		t.Errorf("stderr = %q, want parse error for a.md", stderr)
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Errorf("output file should not be written: stat error = %v", err)
	}
}
