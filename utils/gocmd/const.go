/*
@Author : lsne
@Date : 2020-12-07 15:38
*/

package gocmd

// 操作系统命令
const (
	UserAddCmd  = "/usr/sbin/useradd"
	UserDelCmd  = "/usr/sbin/userdel"
	GroupAddCmd = "/usr/sbin/groupadd"
)

var SystemDirs = []string{
	"/",
	"/bin",
	"/sbin",
	"/etc",
	"/lib",
	"/lib64",
	"/usr",
	"/var",
	"/proc",
	"/sys",
	"/dev",
	"/root",
	"/home",
	"/boot",
	"/run",
	"/tmp",
	"/var/tmp",
}
