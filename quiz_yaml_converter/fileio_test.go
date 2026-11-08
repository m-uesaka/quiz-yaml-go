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
			if got := readFile(t, path); got != content {
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
			if got := readFile(t, path); got != content {
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
	if got := readFile(t, path); got != existing {
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
			if got := readFile(t, target); got != content {
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
			wantErr: "出力ファイルの書き込みに失敗しました",
		},
		{
			name: "SaveYAMLData",
			file: "quiz.yaml",
			write: func(path string) error {
				return SaveYAMLData([]QuizItem{{Question: "q", Answer: "a"}}, path)
			},
			wantErr: "YAMLファイルの書き込みに失敗しました",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			failRename(t)
			dir := t.TempDir()
			const existing = "既存の内容"
			path := writeFile(t, dir, tt.file, existing)

			// Act
			err := tt.write(path)

			// Assert
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error = %v, want it to contain %q", err, tt.wantErr)
			}
			if got := readFile(t, path); got != existing {
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

// 途中のディレクトリがシンボリックリンクで，その中のリンクが相対パスで「..」を含む場合に，
// カーネル（os.WriteFile）と同じく，ディレクトリの実際の場所を基準にリンク先を解決することを確認する．
func TestWriteFileAtomicRelativeSymlinkThroughSymlinkedDir(t *testing.T) {
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
			// dir/proj/out/quiz.csv -> ../shared/quiz.csv（実体は dir/proj/shared/quiz.csv）
			// dir/outlink -> dir/proj/out
			dir := t.TempDir()
			for _, d := range []string{filepath.Join(dir, "proj", "out"), filepath.Join(dir, "proj", "shared")} {
				if err := os.MkdirAll(d, 0o755); err != nil {
					t.Fatalf("failed to mkdir: %v", err)
				}
			}
			target := filepath.Join(dir, "proj", "shared", "quiz.csv")
			if tt.targetExists {
				writeFile(t, filepath.Join(dir, "proj", "shared"), "quiz.csv", "古い内容")
			}
			if err := os.Symlink(filepath.Join("..", "shared", "quiz.csv"), filepath.Join(dir, "proj", "out", "quiz.csv")); err != nil {
				t.Skipf("symlink is not available: %v", err)
			}
			outlink := filepath.Join(dir, "outlink")
			if err := os.Symlink(filepath.Join(dir, "proj", "out"), outlink); err != nil {
				t.Skipf("symlink is not available: %v", err)
			}
			const content = "新しい内容"

			// Act
			err := writeFileAtomic(filepath.Join(outlink, "quiz.csv"), []byte(content), 0o666)

			// Assert
			if err != nil {
				t.Fatalf("writeFileAtomic() error = %v", err)
			}
			if got := readFile(t, target); got != content {
				t.Errorf("target content = %q, want %q", got, content)
			}
			if got := listDir(t, dir); !slices.Equal(got, []string{"outlink", "proj"}) {
				t.Errorf("dir entries = %v, want [outlink proj] (no file created in the wrong place)", got)
			}
			if got := listDir(t, filepath.Join(dir, "proj", "shared")); !slices.Equal(got, []string{"quiz.csv"}) {
				t.Errorf("shared dir entries = %v, want only quiz.csv", got)
			}
		})
	}
}

// 出力先が通常のファイル以外（デバイスファイル）の場合に，置き換えずにそのまま書き込むことを確認する．
func TestWriteFileAtomicNonRegularFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("/dev/null is not available on Windows")
	}
	tests := []struct {
		name    string
		useLink bool
	}{
		{name: "device file", useLink: false},
		{name: "symlink to device file", useLink: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			dir := t.TempDir()
			path := os.DevNull
			if tt.useLink {
				path = filepath.Join(dir, "null")
				if err := os.Symlink(os.DevNull, path); err != nil {
					t.Skipf("symlink is not available: %v", err)
				}
			}

			// Act
			err := writeFileAtomic(path, []byte("内容"), 0o666)

			// Assert
			if err != nil {
				t.Fatalf("writeFileAtomic(%q) error = %v", path, err)
			}
			info, err := os.Stat(os.DevNull)
			if err != nil {
				t.Fatalf("failed to stat %s: %v", os.DevNull, err)
			}
			if info.Mode().IsRegular() {
				t.Errorf("%s was replaced by a regular file", os.DevNull)
			}
			if tt.useLink {
				if got := listDir(t, dir); !slices.Equal(got, []string{"null"}) {
					t.Errorf("dir entries = %v, want only the link", got)
				}
			}
		})
	}
}

// 書き込む権限の無い既存のファイル（読み取り専用）を置き換えず，エラーを返すことを確認する．
func TestWriteFileAtomicReadOnlyFile(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only files")
	}
	// Arrange
	dir := t.TempDir()
	const original = "保護した内容"
	path := writeFile(t, dir, "quiz.csv", original)
	if err := os.Chmod(path, 0o444); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(path, 0o644) })

	// Act
	err := writeFileAtomic(path, []byte("新しい内容"), 0o666)

	// Assert
	if !errors.Is(err, fs.ErrPermission) {
		t.Fatalf("writeFileAtomic() error = %v, want permission error", err)
	}
	if got := readFile(t, path); got != original {
		t.Errorf("content = %q, want %q (unchanged)", got, original)
	}
	if got := listDir(t, dir); !slices.Equal(got, []string{"quiz.csv"}) {
		t.Errorf("dir entries = %v, want only quiz.csv", got)
	}
}

// ディレクトリに書き込む権限が無くても，既存のファイルに書き込めれば，従来どおりそのまま書き込むことを確認する．
func TestWriteFileAtomicReadOnlyDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("directory permissions are not supported on Windows")
	}
	if os.Geteuid() == 0 {
		t.Skip("root can write to read-only directories")
	}
	// Arrange
	dir := filepath.Join(t.TempDir(), "readonly")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatalf("failed to mkdir: %v", err)
	}
	path := writeFile(t, dir, "quiz.csv", "古い内容")
	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("failed to chmod: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) })
	const content = "新しい内容"

	// Act
	err := writeFileAtomic(path, []byte(content), 0o666)

	// Assert
	if err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}
	if got := readFile(t, path); got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
	if got := listDir(t, dir); !slices.Equal(got, []string{"quiz.csv"}) {
		t.Errorf("dir entries = %v, want only quiz.csv", got)
	}
}

// 名前の長さの上限に近い長いファイル名でも書き込めることを確認する（一時ファイルの名前に出力先の名前を含めない）．
func TestWriteFileAtomicLongFileName(t *testing.T) {
	// Arrange
	dir := t.TempDir()
	// 多くのファイルシステムで名前の長さの上限は255バイト（または255文字）．
	// 以前の一時ファイルの名前（「.」+ 出力先の名前 + 「.tmp」+ 乱数）では上限を超える長さにする．
	name := strings.Repeat("a", 247) + ".csv" // 251文字
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte("古い内容"), 0o644); err != nil {
		t.Skipf("the file system does not support this file name: %v", err)
	}
	const content = "新しい内容"

	// Act
	err := writeFileAtomic(path, []byte(content), 0o666)

	// Assert
	if err != nil {
		t.Fatalf("writeFileAtomic() error = %v", err)
	}
	if got := readFile(t, path); got != content {
		t.Errorf("content = %q, want %q", got, content)
	}
	if got := listDir(t, dir); !slices.Equal(got, []string{name}) {
		t.Errorf("dir entries = %v, want only %q", got, name)
	}
}
