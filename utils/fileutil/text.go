package fileutil

import (
	"bufio"
	"fmt"
	"os"
)

func WriteToFile(filename string, content string) error {
	f, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer f.Close()

	w := bufio.NewWriter(f)
	if _, err = w.WriteString(content); err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	return w.Flush()
}

// ReadLineFromFile 全文件按行写入数组变量, 只适合小文件。
func ReadLineFromFile(filename string) ([]string, error) {
	var content []string
	f, err := os.Open(filename)
	if err != nil {
		return content, fmt.Errorf("打开文件失败: %v", err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		content = append(content, scanner.Text())
	}

	return content, nil
}
