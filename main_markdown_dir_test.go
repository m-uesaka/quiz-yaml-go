package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// -markdown-dir に存在しないディレクトリ・ファイル・Markdownファイルを含まないディレクトリを
// 指定した場合，再帰の有無にかかわらず終了コード1でエラーを表示し，既存の出力ファイルを
// 上書きしないことを確認する．
func TestRun_MarkdownDirInvalidDirectory(t *testing.T) {
	const existingOutput = "- question: 既存の問題\n  answer: 既存の答え\n"

	tests := []struct {
		name string
		// mdDirはフィクスチャ用のディレクトリを受け取り，-markdown-dirに渡すパスを返す
		mdDir   func(t *testing.T, dir string) string
		wantMsg string
	}{
		{
			name: "存在しないディレクトリ",
			mdDir: func(_ *testing.T, dir string) string {
				return filepath.Join(dir, "missing")
			},
			wantMsg: "Markdownディレクトリが見つかりません",
		},
		{
			name: "ディレクトリではなくファイル",
			mdDir: func(t *testing.T, dir string) string {
				return writeFile(t, dir, "quiz.md", "---\ntags: []\n---\n## Question\n\n問題文\n\n## Answer\n\n答え\n")
			},
			wantMsg: "Markdownディレクトリではありません",
		},
		{
			name: "空のディレクトリ",
			mdDir: func(t *testing.T, dir string) string {
				mdDir := filepath.Join(dir, "md")
				if err := os.Mkdir(mdDir, 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				return mdDir
			},
			wantMsg: "Markdownファイル（拡張子が .md のファイル）が見つかりません",
		},
		{
			name: "Markdownファイル以外のファイルのみを含むディレクトリ",
			mdDir: func(t *testing.T, dir string) string {
				mdDir := filepath.Join(dir, "md")
				if err := os.Mkdir(mdDir, 0o755); err != nil {
					t.Fatalf("failed to create directory: %v", err)
				}
				writeFile(t, mdDir, "memo.txt", "")
				return mdDir
			},
			wantMsg: "Markdownファイル（拡張子が .md のファイル）が見つかりません",
		},
	}

	for _, tt := range tests {
		for _, recursive := range []bool{false, true} {
			name := tt.name + "/再帰しない"
			if recursive {
				name = tt.name + "/再帰する"
			}
			t.Run(name, func(t *testing.T) {
				// Arrange
				dir := t.TempDir()
				mdDir := tt.mdDir(t, dir)
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
				if !strings.Contains(stderr, tt.wantMsg) {
					t.Errorf("stderr = %q, want to contain %q", stderr, tt.wantMsg)
				}
				if got := readFile(t, output); got != existingOutput {
					t.Errorf("output file was modified: got %q, want %q", got, existingOutput)
				}
			})
		}
	}
}
