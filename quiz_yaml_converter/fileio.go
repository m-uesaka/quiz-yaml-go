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
//   - pathがシンボリックリンクの場合は，os.WriteFile と同じくリンク先のファイルを置き換え，リンクは残す．
//     リンク先が存在しない場合は，リンク先に新しくファイルを作る．
//   - pathが既存のディレクトリの場合や，親ディレクトリが存在しない場合はエラーを返す．
//   - Windowsでは，置き換え先のファイルが他のプロセスで開かれている場合や読み取り専用の場合に
//     os.Rename が失敗することがある．その場合もエラーを返し，既存のファイルはそのまま残る．
//   - 置き換え後にディレクトリのSyncは行わない（Windowsで行えないため）．
//     電源断などの直後には，置き換えが反映されていない可能性がある．
func writeFileAtomic(path string, data []byte, perm fs.FileMode) error {
	target, err := resolveSymlink(path)
	if err != nil {
		return err
	}

	existingMode, exists, err := existingFileMode(target)
	if err != nil {
		return err
	}

	tmp, err := createTempFile(filepath.Dir(target), filepath.Base(target), perm)
	if err != nil {
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

	if exists {
		if err := tmp.Chmod(existingMode); err != nil {
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

// pathの最後の要素がシンボリックリンクの場合，リンクをたどって実際に書き込むファイルのパスを返す．
// filepath.EvalSymlinks はリンク先が存在しない場合にエラーになるため，os.Readlink で1段ずつたどる．
// 途中のディレクトリのシンボリックリンクは解決しない（一時ファイルも同じディレクトリに作るため，問題にならない）．
func resolveSymlink(path string) (string, error) {
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
			link = filepath.Join(filepath.Dir(path), link)
		}
		path = link
	}
	return "", &fs.PathError{Op: "open", Path: path, Err: errors.New("too many levels of symbolic links")}
}

// pathのファイルが存在する場合，その権限を返す．
// pathがディレクトリの場合はエラーを返す．
func existingFileMode(path string) (mode fs.FileMode, exists bool, err error) {
	info, err := os.Stat(path)
	if errors.Is(err, fs.ErrNotExist) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	if info.IsDir() {
		return 0, false, &fs.PathError{Op: "open", Path: path, Err: errors.New("is a directory")}
	}
	return info.Mode().Perm(), true, nil
}

// dirに，baseをもとにした名前の一時ファイルを権限perm（umaskが適用される）で新しく作る．
// 名前が衝突した場合は別の名前で作り直す．
func createTempFile(dir, base string, perm fs.FileMode) (*os.File, error) {
	for range 10000 {
		name := filepath.Join(dir, "."+base+".tmp"+strconv.FormatUint(rand.Uint64(), 36))
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, perm)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, &fs.PathError{Op: "createtemp", Path: filepath.Join(dir, "."+base+".tmp*"), Err: fs.ErrExist}
}
