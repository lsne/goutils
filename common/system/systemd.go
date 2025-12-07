/*
 * @Author: lsne
 * @Date: 2025-12-07 14:50:58
 */

package system

import (
	"fmt"

	"github.com/lsne/goutils/utils/gocmd"
)

func SystemdDaemonReload() error {
	cmd := "systemctl daemon-reload"
	sh := gocmd.Shell{}
	if stdout, stderr, err := sh.Run(cmd); err != nil {
		return fmt.Errorf("daemon-reload失败: %v, 标准输出: %s, 标准错误: %s", err, stdout, stderr)
	}
	return nil
}

func SystemCtl(serviceName, action string) error {
	cmd := fmt.Sprintf("systemctl %s %s", action, serviceName)
	sh := gocmd.Shell{Timeout: 300}
	if stdout, stderr, err := sh.Run(cmd); err != nil {
		return fmt.Errorf("执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", cmd, err, stdout, stderr)
	}
	return nil
}

func SystemResourceLimit(serviceName, limit string) error {
	cmd := fmt.Sprintf("systemctl set-property %s %s", serviceName, limit)
	sh := gocmd.Shell{}
	if stdout, stderr, err := sh.Run(cmd); err != nil {
		return fmt.Errorf("执行(%s)失败: %v, 标准输出: %s, 标准错误: %s", cmd, err, stdout, stderr)
	}
	return nil
}
