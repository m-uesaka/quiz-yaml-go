package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -markdown-dir に不正なパスを指定した場合，再帰の有無にかかわらず終了コード1で
// エラーを表示し，既存の出力ファイルを上書きしないことを確認する．
// （エラーの種類ごとのメッセージはライブラリ側の TestAggregateMarkdownDir_InvalidDirectory で確認する．）
func TestRun_MarkdownDirInvalidDirectory(t *testing.T) {
	const existingOutput = "- question: 既存の問題\n  answer: 既存の答え\n"

	for _, recursive := range []bool{false, true} {
		name := "再帰しない"
		if recursive {
			name = "再帰する"
		}
		t.Run(name, func(t *testing.T) {
			// Arrange: Markdownファイルを含まないディレクトリ
			dir := t.TempDir()
			mdDir := filepath.Join(dir, "md")
			if err := os.Mkdir(mdDir, 0o755); err != nil {
				t.Fatalf("failed to create directory: %v", err)
			}
			output := writeFile(t, dir, "quiz.yaml", existingOutput)
			args := []string{"-markdown-dir", mdDir, "-output", output}
			if recursive {
				args = append(args, "-recursive")
			}

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
			if got := readFile(t, output); got != existingOutput {
				t.Errorf("output file was modified: got %q, want %q", got, existingOutput)
			}
		})
	}
}
