package fileutil

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

func decFile(hdr *tar.Header, tr *tar.Reader, to string) error {
	file := path.Join(to, hdr.Name)
	if dir := filepath.Dir(file); !IsExists(dir) {
		err := os.MkdirAll(dir, 0755)
		if err != nil {
			return err
		}
	}
	switch hdr.Typeflag {
	case tar.TypeSymlink:
		if err := os.Symlink(hdr.Linkname, file); err != nil {
			return fmt.Errorf("解压包中的软链接文件失败: %v", err)
		}
	default:
		fw, err := os.OpenFile(file, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, hdr.FileInfo().Mode())
		if err != nil {
			return err
		}
		defer fw.Close()

		_, err = io.Copy(fw, tr)
		return err
	}
	return nil
}

// 解压tar.gz文件
func UntarGz(from string, to string) error {
	fr, err := os.Open(from)
	if err != nil {
		return err
	}
	defer fr.Close()

	gr, err := gzip.NewReader(fr)
	if err != nil {
		return err
	}
	defer gr.Close()

	tr := tar.NewReader(gr)

	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 过滤以 ../ 开头的文件名, 防止压缩包穿透
		if strings.HasPrefix(hdr.FileInfo().Name(), "../") {
			continue
		}

		// 过滤以 ../ 开头的文件名, 防止压缩包穿透
		if strings.HasPrefix(hdr.Name, "../") {
			continue
		}

		if hdr.FileInfo().IsDir() {
			if err := os.MkdirAll(path.Join(to, hdr.Name), hdr.FileInfo().Mode()); err != nil {
				return err
			}
		} else {
			if err := decFile(hdr, tr, to); err != nil {
				return err
			}
		}
	}
	return nil
}
