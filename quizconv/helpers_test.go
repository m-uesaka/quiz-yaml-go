package quizconv

import (
	"os"
	"path/filepath"
	"testing"
)

// テスト用のファイルを dir/name に権限0o644で作成し，そのパスを返す．
// ルートのパッケージ（main）のテストの writeFile と同じ（テストのヘルパーはパッケージをまたいで使えないため，両方に置いている）．
func writeFile(t testing.TB, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

// path のファイルの内容を文字列で返す．
// ルートのパッケージ（main）のテストの readFile と同じ．
func readFile(t testing.TB, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}

// テスト用のYAMLファイルを一時ディレクトリに作成し，そのパスを返す．
func writeTempYAML(t testing.TB, content string) string {
	t.Helper()
	return writeFile(t, t.TempDir(), "quiz.yaml", content)
}
