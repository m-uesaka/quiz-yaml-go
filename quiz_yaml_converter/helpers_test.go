package quiz_yaml_converter

import (
	"os"
	"path/filepath"
	"testing"
)

// テスト用のYAMLファイルを一時ディレクトリに作成し，そのパスを返す．
func writeTempYAML(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "quiz.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write YAML file: %v", err)
	}
	return path
}
