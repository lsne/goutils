/*
 * @Author: lsne
 * @Date: 2025-12-07 14:51:43
 */

package system

import (
	"fmt"

	"github.com/lsne/goutils/utils/gocmd"
)

func ChownAll(path, user, group string) error {
	cmd := fmt.Sprintf("chown -R %s:%s %s", user, group, path)
	sh := gocmd.Shell{}
	if _, stderr, err := sh.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}

func Chown(path, user, group string) error {
	cmd := fmt.Sprintf("chown %s:%s %s", user, group, path)
	sh := gocmd.Shell{}
	if _, stderr, err := sh.Run(cmd); err != nil {
		return fmt.Errorf("修改数据目录所属用户失败: %v, 标准错误输出: %s", err, stderr)
	}
	return nil
}
