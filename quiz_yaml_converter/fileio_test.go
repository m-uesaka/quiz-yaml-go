package quiz_yaml_converter

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// dirにあるファイル名の一覧を返す．一時ファイルが残っていないことの確認に使う．
func listDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read dir %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	return names
}

// os.WriteFile で新しく作った場合の権限を返す．umaskを直接取得せずに期待値を求めるために使う．
func modeByWriteFile(t *testing.T, perm fs.FileMode) fs.FileMode {
	t.Helper()
	probe := filepath.Join(t.TempDir(), "probe")
	if err := os.WriteFile(probe, nil, perm); err != nil {
		t.Fatalf("failed to write probe file: %v", err)
	}
	info, err := os.Stat(probe)
	if err != nil {
		t.Fatalf("failed to stat probe file: %v", err)
	}
	return info.Mode().Perm()
}

func fileMode(t *testing.T, path string) fs.FileMode {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("failed to stat %s: %v", path, err)
	}
	return info.Mode().Perm()
}

func readFileString(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	return string(b)
}

// 置き換え（os.Rename）を必ず失敗させる．テストの終了時に元に戻す．
// パッケージ変数を書き換えるため，このヘルパーを使うテストでは t.Parallel を使わない．
func failRename(t *testing.T) error {
	t.Helper()
	errRename := errors.New("injected rename failure")
	orig := renameFile
	renameFile = func(string, string) error { return errRename }
	t.Cleanup(func() { renameFile = orig })
	return errRename
}

func TestWriteFileAtomicCreatesNewFile(t *testing.T) {
	tests := []struct {
		name string
		perm fs.FileMode
	}{
		{name: "perm 0666", perm: 0o666},
		{name: "perm 0644", perm: 0o644},
		{name: "perm 0600", perm: 0o600},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			path := filepath.Join(dir, "out.txt")
			const content = "新しい内容\n"

			// Act
			err := writeFileAtomic(path, []byte(content), tt.perm)

			// Assert
			if err != nil {
				t.Fatalf("writeFileAtomic() error = %v", err)
			}
			if got := readFileString(t, path); got != content {
				t.Errorf("content = %q, want %q", got, content)
			}
			if got := listDir(t, dir); !slices.Equal(got, []string{"out.txt"}) {
				t.Errorf("dir entries = %v, want only out.txt", got)
			}
			if runtime.GOOS == "windows" {
				return
			}
			if got, want := fileMode(t, path), modeByWriteFile(t, tt.perm); got != want {
				t.Errorf("mode = %v, want %v (same as os.WriteFile)", got, want)
			}
		})
	}
}

func TestWriteFileAtomicReplacesExistingFile(t *testing.T) {
	tests := []struct {
		name         string
		existingMode fs.FileMode
		perm         fs.FileMode
	}{
		{name: "keeps 0600 even if perm is 0666", existingMode: 0o600, perm: 0o666},
		{name: "keeps 0640 even if perm is 0644", existingMode: 0o640, perm: 0o644},
		{name: "keeps 0755", existingMode: 0o755, perm: 0o644},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			path := filepath.Join(dir, "out.txt")
			if err := os.WriteFile(path, []byte("古い内容"), 0o600); err != nil {
				t.Fatalf("failed to write existing file: %v", err)
			}
			if err := os.Chmod(path, tt.existingMode); err != nil {
				t.Fatalf("failed to chmod existing file: %v", err)
			}
			const content = "新しい内容"

			// Act
			err := writeFileAtomic(path, []byte(content), tt.perm)

			// Assert
			if err != nil {
				t.Fatalf("writeFileAtomic() error = %v", err)
			}
			if got := readFileString(t, path); got != content {
				t.Errorf("content = %q, want %q", got, content)
			}
			if got := listDir(t, dir); !slices.Equal(got, []string{"out.txt"}) {
				t.Errorf("dir entries = %v, want only out.txt", got)
			}
			if runtime.GOOS == "windows" {
				return
			}
			if got := fileMode(t, path); got != tt.existingMode {
				t.Errorf("mode = %v, want %v", got, tt.existingMode)
			}
		})
	}
}

// 置き換えに失敗した場合，既存のファイルの内容・権限が変わらず，一時ファイルが残らないことを確認する．
func TestWriteFileAtomicRenameFailureKeepsExistingFile(t *testing.T) {
	// Arrange
	errRename := failRename(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "out.txt")
	const existing = "既存の内容"
	if err := os.WriteFile(path, []byte(existing), 0o600); err != nil {
		t.Fatalf("failed to write existing file: %v", err)
	}
	wantMode := fileMode(t, path)

	// Act
	err := writeFileAtomic(path, []byte("新しい内容"), 0o666)

	// Assert
	if !errors.Is(err, errRename) {
		t.Fatalf("writeFileAtomic() error = %v, want %v", err, errRename)
	}
	if got := readFileString(t, path); got != existing {
		t.Errorf("content = %q, want %q", got, existing)
	}
	if got := fileMode(t, path); got != wantMode {
		t.Errorf("mode = %v, want %v", got, wantMode)
	}
	if got := listDir(t, dir); !slices.Equal(got, []string{"out.txt"}) {
		t.Errorf("dir entries = %v, want only out.txt (temp file must be removed)", got)
	}
}

func TestWriteFileAtomicErrors(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T, dir string) string
		// エラーの後のdirの中身
		wantEntries []string
	}{
		{
			name: "target is a directory",
			setup: func(t *testing.T, dir string) string {
				path := filepath.Join(dir, "out")
				if err := os.Mkdir(path, 0o755); err != nil {
					t.Fatalf("failed to mkdir: %v", err)
				}
				return path
			},
			wantEntries: []string{"out"},
		},
		{
			name: "parent directory does not exist",
			setup: func(t *testing.T, dir string) string {
				return filepath.Join(dir, "missing", "out.txt")
			},
			wantEntries: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			path := tt.setup(t, dir)

			// Act
			err := writeFileAtomic(path, []byte("内容"), 0o666)

			// Assert
			if err == nil {
				t.Fatal("writeFileAtomic() expected an error, got nil")
			}
			if got := listDir(t, dir); !slices.Equal(got, tt.wantEntries) {
				t.Errorf("dir entries = %v, want %v", got, tt.wantEntries)
			}
		})
	}
}

// 出力先がシンボリックリンクの場合，os.WriteFile と同じくリンク先に書き込み，リンク自体は残ることを確認する．
func TestWriteFileAtomicSymlink(t *testing.T) {
	tests := []struct {
		name         string
		targetExists bool
	}{
		{name: "link to existing file", targetExists: true},
		{name: "dangling link", targetExists: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			targetDir := filepath.Join(dir, "real")
			if err := os.Mkdir(targetDir, 0o755); err != nil {
				t.Fatalf("failed to mkdir: %v", err)
			}
			target := filepath.Join(targetDir, "out.txt")
			if tt.targetExists {
				if err := os.WriteFile(target, []byte("古い内容"), 0o600); err != nil {
					t.Fatalf("failed to write target: %v", err)
				}
			}
			link := filepath.Join(dir, "link.txt")
			if err := os.Symlink(filepath.Join("real", "out.txt"), link); err != nil {
				t.Skipf("symlink is not available: %v", err)
			}
			const content = "新しい内容"

			// Act
			err := writeFileAtomic(link, []byte(content), 0o666)

			// Assert
			if err != nil {
				t.Fatalf("writeFileAtomic() error = %v", err)
			}
			info, err := os.Lstat(link)
			if err != nil {
				t.Fatalf("failed to lstat link: %v", err)
			}
			if info.Mode()&fs.ModeSymlink == 0 {
				t.Errorf("link was replaced by a regular file: mode = %v", info.Mode())
			}
			if got := readFileString(t, target); got != content {
				t.Errorf("target content = %q, want %q", got, content)
			}
			if got := listDir(t, targetDir); !slices.Equal(got, []string{"out.txt"}) {
				t.Errorf("target dir entries = %v, want only out.txt", got)
			}
			if got := listDir(t, dir); !slices.Equal(got, []string{"link.txt", "real"}) {
				t.Errorf("dir entries = %v, want [link.txt real]", got)
			}
			if runtime.GOOS == "windows" || !tt.targetExists {
				return
			}
			if got := fileMode(t, target); got != 0o600 {
				t.Errorf("target mode = %v, want %v", got, fs.FileMode(0o600))
			}
		})
	}
}

// 呼び出し元（テンプレートでの出力と SaveYAMLData）で，置き換えに失敗した場合に既存のファイルが壊れず，
// 従来どおりのエラーメッセージを返すことを確認する．
func TestWriteCallersKeepExistingFileOnFailure(t *testing.T) {
	tests := []struct {
		name    string
		file    string
		write   func(path string) error
		wantErr string
	}{
		{
			name: "builtin template",
			file: "quiz.csv",
			write: func(path string) error {
				return executeTemplateToFile("{{range .Items}}{{.Question}}{{end}}", false, []QuizItem{{Question: "q"}}, path)
			},
			wantErr: "failed to write output file",
		},
		{
			name: "SaveYAMLData",
			file: "quiz.yaml",
			write: func(path string) error {
				return SaveYAMLData([]QuizItem{{Question: "q", Answer: "a"}}, path)
			},
			wantErr: "failed to write YAML file",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			failRename(t)
			dir := t.TempDir()
			path := filepath.Join(dir, tt.file)
			const existing = "既存の内容"
			if err := os.WriteFile(path, []byte(existing), 0o644); err != nil {
				t.Fatalf("failed to write existing file: %v", err)
			}

			// Act
			err := tt.write(path)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if got := readFileString(t, path); got != existing {
				t.Errorf("content = %q, want %q", got, existing)
			}
			if got := listDir(t, dir); !slices.Equal(got, []string{tt.file}) {
				t.Errorf("dir entries = %v, want only %s", got, tt.file)
			}
		})
	}
}

// SaveYAMLData で新しく作るファイルの権限が，従来どおり 0644 から umask を引いたものであることを確認する．
func TestSaveYAMLDataNewFileMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits are not meaningful on Windows")
	}
	// Arrange
	path := filepath.Join(t.TempDir(), "quiz.yaml")

	// Act
	err := SaveYAMLData([]QuizItem{{Question: "q", Answer: "a"}}, path)

	// Assert
	if err != nil {
		t.Fatalf("SaveYAMLData() error = %v", err)
	}
	if got, want := fileMode(t, path), modeByWriteFile(t, 0o644); got != want {
		t.Errorf("mode = %v, want %v", got, want)
	}
}
