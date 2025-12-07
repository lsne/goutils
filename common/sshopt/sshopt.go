/*
 * @Author: lsne
 * @Date: 2025-12-07 14:45:05
 */

package sshopt

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/lsne/goutils/common/system"
	"github.com/lsne/goutils/environment"
	"github.com/lsne/goutils/utils/gocmd"
	"github.com/lsne/goutils/utils/netutil"
)

var keyfile = filepath.Join(environment.GlobalEnv().HomePath, ".ssh", "id_rsa")

// SshOptions ssh连接选项
type SshOptions struct {
	Host     string `yaml:"host" ini:"host"`
	Port     uint16 `yaml:"port" ini:"ssh-port"`
	Username string `yaml:"username" ini:"ssh-user"`
	Password string `yaml:"password" ini:"ssh-password"`
	KeyFile  string `yaml:"keyfile" ini:"ssh-keyfile"`
	TmpDir   string `yaml:"tmp-dir" ini:"tmp-dir"`
}

func (o *SshOptions) SetDefault(tmpdir string) {
	if o.Port == 0 {
		o.Port = 22
	}

	if o.TmpDir == "" {
		o.TmpDir = tmpdir
	}

	if o.Password == "" && o.KeyFile == "" {
		o.KeyFile = keyfile
	}
}

func (o *SshOptions) ValidateAndHost() error {
	if err := o.ValidateHost(); err != nil {
		return err
	}
	return o.Validate()
}

func (o *SshOptions) Validate() error {
	if err := o.ValidatePort(); err != nil {
		return err
	}
	if err := o.ValidateUsername(); err != nil {
		return err
	}
	return o.ValidateTmpDir()
}

func (o *SshOptions) ValidateHost() error {
	if !netutil.ValidHostnameOrIP(o.Host) {
		return fmt.Errorf("arbiter (%s) 既不是合法的 IP 地址, 也不是合法的主机名", o.Host)
	}
	return nil
}

func (o *SshOptions) ValidatePort() error {
	if o.Port < 1 {
		return fmt.Errorf("端口号(%d)必须在 1 ~ 65535 之间", o.Port)
	}
	return nil
}

func (o *SshOptions) ValidateUsername() error {
	if o.Username == "" {
		return fmt.Errorf("linux 操作系统用户名不能为空")
	}
	if !system.IsValidName(o.Username) {
		return fmt.Errorf("linux 操作系统用户名(%s)格式不合法", o.Username)
	}
	return nil
}

func (o *SshOptions) ValidateTmpDir() error {
	if o.TmpDir == "" {
		return fmt.Errorf("临时目录不能为空")
	}

	if slices.Contains(gocmd.SystemDirs, strings.TrimSpace(o.TmpDir)) {
		return fmt.Errorf("临时目录(%s)不能使用系统目录", o.TmpDir)
	}
	return nil
}
