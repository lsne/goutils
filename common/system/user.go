/*
 * @Author: lsne
 * @Date: 2025-12-07 14:54:13
 */

package system

import (
	"fmt"
	"os/user"
	"regexp"

	"github.com/lsne/goutils/utils/gocmd"
	"github.com/lsne/goutils/utils/logger"
)

var (
	validName = regexp.MustCompile(`^[a-z_][a-z0-9_-]*$`)
	maxLength = 32
)

func IsValidName(s string) bool {
	return len(s) <= maxLength && validName.MatchString(s)
}

// 如果用户已经存在,则返回真正的所属组名
func CreateUser(username, groupName string) (string, string, error) {
	logger.Infof("创建 linux 系统用户: %s", username)

	if !IsValidName(username) {
		return "", "", fmt.Errorf("invalid username: %s", username)
	}
	if !IsValidName(groupName) {
		return "", "", fmt.Errorf("invalid group name: %s", groupName)
	}

	u, err := user.Lookup(username)
	if err == nil { // 如果用户已经存在,则返回真正的所属组名
		g, _ := user.LookupGroupId(u.Gid)
		return username, g.Name, nil
	}
	// groupadd -f <group-name>
	groupAdd := fmt.Sprintf("%s -f %s", GroupAddCmd, groupName)

	// useradd -g <group-name> <user-name>
	userAdd := fmt.Sprintf("%s -g %s %s", UserAddCmd, groupName, username)

	sh := gocmd.Shell{}
	if _, stderr, err := sh.Run(groupAdd); err != nil {
		return "", "", fmt.Errorf("创建用户组(%s)失败: %v, 标准错误输出: %s", groupName, err, stderr)
	}
	if _, stderr, err := sh.Run(userAdd); err != nil {
		return "", "", fmt.Errorf("创建用户(%s)失败: %v, 标准错误输出: %s", username, err, stderr)
	}
	return username, groupName, nil
}
