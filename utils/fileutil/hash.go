package fileutil

import (
	"crypto/md5"
	"encoding/hex"
	"io"
	"os"
)

// 生成指定文件的md5效验码
func FileMD5(file string) (string, error) {
	tarball, err := os.OpenFile(file, os.O_RDONLY, 0)
	if err != nil {
		return "", err
	}
	defer tarball.Close()
	m := md5.New()
	if _, err := io.Copy(m, tarball); err != nil {
		return "", err
	}

	checksum := hex.EncodeToString(m.Sum(nil))
	return checksum, nil
}

// 生成指定字符数组的md5效验码
func BytesMD5(b []byte) string {
	m := md5.New()
	m.Write(b)
	return hex.EncodeToString(m.Sum(nil))
}
