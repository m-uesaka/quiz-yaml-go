package quiz_yaml_converter

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
)

// 一時ファイルを出力先に置き換える関数．テストで置き換えの失敗を再現するために変数にしている．
var renameFile = os.Rename

// シンボリックリンクをたどる回数の上限．循環したリンクで無限ループにならないようにする．
const maxSymlinkDepth = 255

// dataをpathに書き込む．os.WriteFile と違い，既存のファイルを空にしてから書き込むのではなく，
// 出力先と同じディレクトリの一時ファイルに書き込み，Sync・Closeが成功してから os.Rename で置き換える．
// そのため，書き込みの途中で失敗した場合（ディスクが一杯，I/Oエラーなど）も既存のファイルは壊れず，
// 失敗した場合は一時ファイルを削除する．
//
//   - 新しくファイルを作る場合の権限は，os.WriteFile と同じくpermからumaskを引いたもの．
//     一時ファイルを os.OpenFile（O_CREATE|O_EXCL）でpermを指定して作ることで，umaskの適用をOSに任せる
//     （os.CreateTemp は権限が0600に固定されるため使わない）．
//   - 既存のファイルを置き換える場合は，元のファイルの権限を引き継ぐ．所有者・グループ・拡張属性・
//     ハードリンクは引き継がない（置き換え後は別のファイルになる）．
//   - 既存のファイルに書き込む権限が無い場合（読み取り専用にしたファイルなど）は，os.WriteFile と同じく
//     エラーを返す（置き換えはディレクトリの権限だけで行えてしまうため，事前に確かめる）．
//   - pathがシンボリックリンクの場合は，os.WriteFile と同じくリンク先のファイルを置き換え，リンクは残す．
//     リンク先が存在しない場合は，リンク先に新しくファイルを作る．
//   - pathが通常のファイル以外（/dev/stdout などのデバイスファイルや名前付きパイプ）の場合は，
//     置き換えずに os.WriteFile でそのまま書き込む．
//   - 既存のファイルには書き込めるが，ディレクトリに一時ファイルを作る権限が無い場合は，
//     os.WriteFile でそのまま書き込む（従来どおり．この場合は書き込みの途中で失敗すると既存のファイルは壊れる）．
//   - pathが既存のディレクトリの場合や，親ディレクトリが存在しない場合はエラーを返す．
//   - Windowsでは，置き換え先のファイルが他のプロセスで開かれている場合に os.Rename が失敗することがある．
//     その場合もエラーを返し，既存のファイルはそのまま残る．
//   - 置き換え後にディレクトリのSyncは行わない（Windowsで行えないため）．
//     電源断などの直後には，置き換えが反映されていない可能性がある．
//   - 書き込みの途中でプロセスが強制終了された場合（Ctrl-Cなど）は，一時ファイル（.quiz-yaml-*.tmp）が残ることがある．
func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	info, err := os.Stat(path)
	switch {
	case err == nil:
		if info.IsDir() {
			return &fs.PathError{Op: "open", Path: path, Err: errors.New("is a directory")}
		}
		if !info.Mode().IsRegular() {
			return os.WriteFile(path, data, perm)
		}
		if err := checkWritable(path); err != nil {
			return err
		}
		// 途中のディレクトリを含めてリンクを解決し，実際のファイルと同じディレクトリに一時ファイルを作る．
		target, err := filepath.EvalSymlinks(path)
		if err != nil {
			return err
		}
		mode := info.Mode().Perm()
		return replaceFile(target, data, perm, &mode)
	case errors.Is(err, fs.ErrNotExist):
		target, err := resolveDanglingSymlink(path)
		if err != nil {
			return err
		}
		return replaceFile(target, data, perm, nil)
	default:
		return err
	}
}

// 一時ファイルに書き込んでから，targetに置き換える．
// existingModeがnilでない場合は既存のファイルを置き換えるとみなし，一時ファイルの権限をexistingModeにする．
func replaceFile(target string, data []byte, perm fs.FileMode, existingMode *fs.FileMode) error {
	tmp, err := createTempFile(filepath.Dir(target), perm)
	if err != nil {
		if existingMode != nil && errors.Is(err, fs.ErrPermission) {
			// ディレクトリに書き込めなくても既存のファイルには書き込めるため，従来どおりそのまま書き込む．
			return os.WriteFile(target, data, perm)
		}
		return err
	}
	tmpName := tmp.Name()
	succeeded := false
	defer func() {
		if !succeeded {
			_ = tmp.Close()
			_ = os.Remove(tmpName)
		}
	}()

	if existingMode != nil {
		if err := tmp.Chmod(*existingMode); err != nil {
			return err
		}
	}
	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := renameFile(tmpName, target); err != nil {
		return err
	}
	succeeded = true
	return nil
}

// pathのファイルを書き込み用に開けるかどうかを確かめる（内容は変更しない）．
// 開けない場合は os.WriteFile と同じエラー（*fs.PathError）を返す．
func checkWritable(path string) error {
	f, err := os.OpenFile(path, os.O_WRONLY, 0)
	if err != nil {
		return err
	}
	return f.Close()
}

// pathが存在しないファイルを指す場合に，実際に新しく作るファイルのパスを返す．
// pathがリンク先の存在しないシンボリックリンク（の連なり）の場合はリンクをたどり，最後のリンク先を返す．
// filepath.EvalSymlinks はリンク先が存在しない場合にエラーになるため，os.Readlink で1段ずつたどる．
// 相対パスのリンク先は，カーネルと同じく，リンクが置かれたディレクトリの実際の場所（途中のリンクを解決したもの）を基準にする．
func resolveDanglingSymlink(path string) (string, error) {
	for range maxSymlinkDepth {
		info, err := os.Lstat(path)
		if errors.Is(err, fs.ErrNotExist) {
			return path, nil
		}
		if err != nil {
			return "", err
		}
		if info.Mode()&fs.ModeSymlink == 0 {
			return path, nil
		}
		link, err := os.Readlink(path)
		if err != nil {
			return "", err
		}
		if !filepath.IsAbs(link) {
			dir, err := filepath.EvalSymlinks(filepath.Dir(path))
			if err != nil {
				return "", err
			}
			link = filepath.Join(dir, link)
		}
		path = link
	}
	return "", &fs.PathError{Op: "open", Path: path, Err: errors.New("too many levels of symbolic links")}
}

// 一時ファイルの名前の先頭と末尾．出力先のファイル名を含めないことで，長いファイル名でも名前の長さの上限を超えないようにする．
const (
	tempFilePrefix = ".quiz-yaml-"
	tempFileSuffix = ".tmp"
)

// dirに一時ファイルを権限perm（umaskが適用される）で新しく作る．
// 名前が衝突した場合は別の名前で作り直す．
func createTempFile(dir string, perm fs.FileMode) (*os.File, error) {
	for range 10000 {
		name := filepath.Join(dir, tempFilePrefix+strconv.FormatUint(rand.Uint64(), 36)+tempFileSuffix)
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, &fs.PathError{Op: "createtemp", Path: filepath.Join(dir, tempFilePrefix+"*"+tempFileSuffix), Err: fs.ErrExist}
}
