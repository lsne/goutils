/*
@Author : lsne
@Date : 2020-12-02 18:23
*/

package gocmd

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os/exec"
	"time"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
)

// Shell execute the command at local host.
type Shell struct {
	Timeout int
	User    string // sudo 用户名。 默认为空时, 执行sudo不传-u参数, 以默认root执行
	Locale  string // the locale used when executing the command
}

func (sh *Shell) Run(cmd string) ([]byte, []byte, error) {
	// set a basic PATH in case it's empty on login
	cmd = fmt.Sprintf("PATH=$PATH:/usr/bin:/usr/sbin %s", cmd)

	if sh.Locale != "" {
		cmd = fmt.Sprintf("export LANG=%s; %s", sh.Locale, cmd)
	}

	if sh.Timeout == 0 {
		sh.Timeout = 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sh.Timeout)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "/bin/sh", "-c", cmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()

	if err != nil {
		return stdout.Bytes(), stderr.Bytes(), err
	}

	return stdout.Bytes(), stderr.Bytes(), nil
}

func (sh *Shell) Sudo(cmd string) ([]byte, []byte, error) {
	var sudoStr string
	if sh.User != "" {
		sudoStr = " -u " + sh.User
	}
	cmd = fmt.Sprintf("sudo -S -H %s /bin/bash -c \"cd; %s\"", sudoStr, cmd)
	return sh.Run(cmd)
}

func (sh *Shell) WinRun(cmd string) ([]byte, []byte, error) {
	if sh.Timeout == 0 {
		sh.Timeout = 60
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(sh.Timeout)*time.Second)
	defer cancel()

	command := exec.CommandContext(ctx, "cmd", "/c", cmd)

	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	command.Stdout = stdout
	command.Stderr = stderr

	err := command.Run()

	stdoutBytes, _ := GbkToUtf8(stdout.Bytes())
	stderrBytes, _ := GbkToUtf8(stderr.Bytes())

	if err != nil {
		return stdoutBytes, stderrBytes, err
	}

	return stdoutBytes, stderrBytes, nil
}

func GbkToUtf8(s []byte) ([]byte, error) {
	reader := transform.NewReader(bytes.NewReader(s), simplifiedchinese.GBK.NewDecoder())
	d, e := io.ReadAll(reader)
	if e != nil {
		return nil, e
	}
	return d, nil
}
