package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

// testProg はテストでrunに渡すプログラム名
const testProg = "quiz-yaml-converter"

// runCLI はrunを呼び出し，終了コードと標準出力・標準エラー出力の内容を返す
func runCLI(args ...string) (code int, stdout, stderr string) {
	var outBuf, errBuf bytes.Buffer
	code = run(testProg, args, &outBuf, &errBuf)
	return code, outBuf.String(), errBuf.String()
}

// テスト用のファイルを dir/name に権限0o644で作成し，そのパスを返す．
// quiz_yaml_converter パッケージのテストの writeFile と同じ（テストのヘルパーはパッケージをまたいで使えないため，両方に置いている）．
func writeFile(t testing.TB, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write %s: %v", path, err)
	}
	return path
}

// path のファイルの内容を文字列で返す．
// quiz_yaml_converter パッケージのテストの readFile と同じ．
func readFile(t testing.TB, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}
