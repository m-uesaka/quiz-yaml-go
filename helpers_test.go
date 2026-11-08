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
