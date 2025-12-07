package fileutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// 判断文件是否存在
func IsExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// 判断所给路径是否为目录, 是否为文件可以用 !IsDir
func IsDir(path string) bool {
	s, err := os.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

// 目录是否为空
func IsEmpty(path string) (bool, error) {
	f, err := os.Open(path)
	if err != nil {
		return false, err
	}
	defer f.Close()

	_, err = f.Readdirnames(1)
	if err == io.EOF {
		return true, nil
	}
	return false, err
}

// 验证数据目录, 存在看是否为空
func IsDirEmptyOrNotExists(dir string) error {
	if !IsExists(dir) {
		return nil
	}

	if !IsDir(dir) {
		return fmt.Errorf("指定的路径(%s)不是目录", dir)
	}

	empty, err := IsEmpty(dir)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("数据目录(%s)不为空", dir)
	}

	return nil
}

// IsDirOrNotExists 判断 path 是否不存在，或者存在且是一个目录。
// 返回 true 表示可以安全地将其视为一个目录（即使当前不存在）。
func IsDirOrNotExists(path string) bool {
	if !IsExists(path) {
		return true // 不存在，视为合法
	}
	return IsDir(path) // 存在，则必须是目录
}

// 开机自动创建 /var/run 目录下的文件或路径
func CreateRunDir(filename, dir, user, group string) error {
	filename = filepath.Join("/usr/lib/tmpfiles.d", filename)
	f, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	title := fmt.Sprintf("d /var/run/%s 0755 %s %s", dir, user, group)
	if _, err := fmt.Fprintln(w, title); err != nil {
		return err
	}
	return w.Flush()
}

func Rename(src, dst string) error {
	return os.Rename(src, dst)
}

func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func CopyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode())
		}

		return CopyFile(path, target)
	})
}

func backupSuffix() string {
	return ".bak." + time.Now().Format("20060102150405")
}

func MoveToBackup(src string) error {
	dst := filepath.Clean(src) + backupSuffix()
	return Rename(src, dst)
}

func BackupFile(src string) error {
	dst := filepath.Clean(src) + backupSuffix()
	return CopyFile(src, dst)
}

func BackupDir(src string) error {
	dst := filepath.Clean(src) + backupSuffix()
	return CopyDir(src, dst)
}

func CreateFileIfNotExists(filepath string, perm os.FileMode) error {
	// O_CREATE: 如果文件不存在则创建
	// O_EXCL:   与 O_CREATE 一起使用时，确保文件必须不存在（原子性创建）
	file, err := os.OpenFile(filepath, os.O_CREATE|os.O_EXCL, perm)
	if err != nil {
		if os.IsExist(err) {
			// 文件已存在，按需求“不做任何操作”，视为成功
			return nil
		}
		// 其他错误（如权限不足、路径不存在等）
		return fmt.Errorf("failed to create file %s: %w", filepath, err)
	}
	// 文件是新建的，立即关闭（内容为空）
	file.Close()
	return nil
}

func HasPermOrNotExists(filePath, username, perm string) error {
	if !IsExists(filePath) {
		return nil
	}

	ok, err := HasPermission(filePath, username, perm)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("用户 %s 对 %s 没有 %s 权限", username, filePath, perm)
	}
	return nil
}

func HasPerm(filePath, username, perm string) error {
	ok, err := HasPermission(filePath, username, perm)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("用户 %s 对 %s 没有 %s 权限", username, filePath, perm)
	}
	return nil
}

// HasPermission 检查指定用户对给定路径是否具有指定的权限组合。
// perm 是一个包含 'r'、'w'、'x' 的字符串（如 "rw", "wx", "rwx", "x"）。
// 注意：对目录而言，'x' 表示可进入，'r' 表示可列出内容，'w' 表示可创建/删除文件。
func HasPermission(path, username, perm string) (bool, error) {
	// 1. 解析权限需求
	wantRead := strings.Contains(perm, "r")
	wantWrite := strings.Contains(perm, "w")
	wantExec := strings.Contains(perm, "x")

	if !wantRead && !wantWrite && !wantExec {
		return true, nil // 无权限要求，默认满足
	}

	// 2. 获取用户 UID/GID
	u, err := user.Lookup(username)
	if err != nil {
		return false, fmt.Errorf("用户 %s 不存在: %w", username, err)
	}

	uid, err := strconv.ParseUint(u.Uid, 10, 32)
	if err != nil {
		return false, fmt.Errorf("无法解析 UID: %w", err)
	}
	gid, err := strconv.ParseUint(u.Gid, 10, 32)
	if err != nil {
		return false, fmt.Errorf("无法解析 GID: %w", err)
	}

	// 3. 获取文件/目录元信息
	fileInfo, err := os.Stat(path)
	if err != nil {
		return false, fmt.Errorf("无法访问路径 %s: %w", path, err)
	}

	stat, ok := fileInfo.Sys().(*syscall.Stat_t)
	if !ok {
		return false, fmt.Errorf("不支持的操作系统（非 Unix）")
	}

	fileUID := stat.Uid
	fileGID := stat.Gid
	mode := stat.Mode

	// 4. 判断用户类别并提取对应权限位
	var hasRead, hasWrite, hasExec bool

	if uint32(uid) == fileUID {
		// 所有者
		hasRead = (mode & 0o400) != 0  // S_IRUSR
		hasWrite = (mode & 0o200) != 0 // S_IWUSR
		hasExec = (mode & 0o100) != 0  // S_IXUSR
	} else if uint32(gid) == fileGID {
		// 所属组
		hasRead = (mode & 0o040) != 0  // S_IRGRP
		hasWrite = (mode & 0o020) != 0 // S_IWGRP
		hasExec = (mode & 0o010) != 0  // S_IXGRP
	} else {
		// 其他用户
		hasRead = (mode & 0o004) != 0  // S_IROTH
		hasWrite = (mode & 0o002) != 0 // S_IWOTH
		hasExec = (mode & 0o001) != 0  // S_IXOTH
	}

	// 5. 检查是否满足所有要求的权限
	if wantRead && !hasRead {
		return false, nil
	}
	if wantWrite && !hasWrite {
		return false, nil
	}
	if wantExec && !hasExec {
		return false, nil
	}

	return true, nil
}
