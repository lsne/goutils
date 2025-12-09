/*
 * @Author: lsne
 * @Date: 2025-11-10 19:40:23
 */

package gossh

import (
	"bufio"
	"bytes"
	"dbup/internal/utils/fileutil"
	"dbup/internal/utils/gocmd"
	"dbup/internal/utils/strutil"
	"fmt"
	"io"
	"net"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

const GOSSH_ERR_FORMAT = "在机器: %s 上, 执行(%s)失败: %v, 标准输出: %s, 标准错误: %s"

type RunOptions struct {
	hide         bool
	sudoUser     string
	sudoPassword string
	sudoPattern  string
	Watchers     []Watcher
}

// Option 是一个函数类型，用于修改 RunOptions
type Option func(*RunOptions)

// WithHide 返回一个 Option，用于设置 hide
func WithHide(hide bool) Option {
	return func(o *RunOptions) {
		o.hide = hide
	}
}

// WithWatchers 返回一个 Option，用于设置 Watchers
func WithWatchers(watchers ...Watcher) Option {
	return func(o *RunOptions) {
		o.Watchers = append(o.Watchers, watchers...)
	}
}

// WithSudoUser 只用于 sudo 函数
func WithSudoUser(user string) Option {
	return func(o *RunOptions) {
		o.sudoUser = user
	}
}

// WithSudoPassword 只用于 sudo 函数
func WithSudoPassword(password string) Option {
	return func(o *RunOptions) {
		o.sudoPassword = password
	}
}

// WithSudoPattern 只用于 sudo 函数
func WithSudoPattern(pattern string) Option {
	return func(o *RunOptions) {
		o.sudoPattern = pattern
	}
}

//type SudoOptions struct {
//	SudoUser     string
//	SudoPassword string
//	SudoPattern  string
//	Watchers     []Watcher
//}

type Connection struct {
	Host         string
	Port         uint16
	User         string
	Password     string
	KeyFile      string
	Timeout      int64
	auth         []ssh.AuthMethod
	clientConfig *ssh.ClientConfig
	sshClient    *ssh.Client
	*sftp.Client
}

func NewConnection(host string, port uint16, user string, password string, keyfile string, timeout int64) (*Connection, error) {
	var err error
	conn := &Connection{Host: host, Port: port, User: user, Password: password, KeyFile: keyfile, Timeout: timeout}

	if conn.Port == 0 {
		conn.Port = 22
	}

	conn.auth = make([]ssh.AuthMethod, 0)

	if password != "" {
		auth := ssh.Password(password)
		conn.auth = append(conn.auth, auth)
	}

	if keyfile != "" {
		key, err := os.ReadFile(keyfile)
		if err != nil {
			return nil, err
		}
		signer, err := ssh.ParsePrivateKey(key)
		if err != nil {
			return nil, err
		}
		conn.auth = append(conn.auth, ssh.PublicKeys(signer))
	}

	conn.clientConfig = &ssh.ClientConfig{
		User:            user,
		Auth:            conn.auth,
		HostKeyCallback: ssh.HostKeyCallback(func(hostname string, remote net.Addr, key ssh.PublicKey) error { return nil }),
		Timeout:         time.Duration(timeout) * time.Second,
	}

	address := fmt.Sprintf("%s:%d", conn.Host, conn.Port)

	if conn.sshClient, err = ssh.Dial("tcp", address, conn.clientConfig); err != nil {
		return nil, err
	}

	if conn.Client, err = sftp.NewClient(conn.sshClient); err != nil {
		return nil, err
	}

	return conn, nil
}

func (conn *Connection) Run(cmd string, opts ...Option) (stdoutByte []byte, stderrByte []byte, err error) {
	var stdin io.WriteCloser
	var stdouts io.Reader
	var stdout io.Reader
	var stderr = new(bytes.Buffer)

	options := &RunOptions{
		hide: false, // 默认显示输出
	}

	for _, opt := range opts {
		opt(options)
	}

	cmd = fmt.Sprintf("PATH=$PATH:/usr/bin:/usr/sbin %s", cmd)
	session, err := conn.sshClient.NewSession()
	if err != nil {
		return stdoutByte, stderrByte, err
	}
	defer session.Close()

	modes := ssh.TerminalModes{
		ssh.ECHO:          0,      // disable echoing
		ssh.TTY_OP_ISPEED: 144000, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 144000, // output speed = 14.4kbaud
	}

	if err = session.RequestPty("xterm", 80, 40, modes); err != nil {
		return stdoutByte, stderrByte, err
	}
	if stdin, err = session.StdinPipe(); err != nil {
		return stdoutByte, stderrByte, err
	}
	if stdout, err = session.StdoutPipe(); err != nil {
		return stdoutByte, stderrByte, err
	}
	session.Stderr = stderr

	// 👇 关键：用 TeeReader 同时写入 os.Stdout 和供 watchers 读取
	if !options.hide {
		stdouts = io.TeeReader(stdout, os.Stdout) // 实时输出到终端！
	} else {
		stdouts = stdout
	}

	var wg sync.WaitGroup
	wg.Go(func() {
		watchers(stdin, stdouts, &stdoutByte, options.Watchers)
	})
	err = session.Run(cmd)
	wg.Wait()

	if err != nil {
		return stdoutByte, stderr.Bytes(), err
	}
	return stdoutByte, stderr.Bytes(), nil
}

func (conn *Connection) Sudo(cmd string, opts ...Option) (stdoutByte []byte, stderrByte []byte, err error) {
	options := &RunOptions{
		hide: false, // 默认显示输出
	}

	for _, opt := range opts {
		opt(options)
	}

	if options.sudoUser == "" {
		options.sudoUser = "root"
	}

	if options.sudoPassword == "" {
		options.sudoPassword = conn.Password
	}

	if options.sudoPattern == "" {
		options.sudoPattern = "[sudo] password: "
	}

	cmd = fmt.Sprintf("sudo -S -p '%s' -H -u %s /bin/bash -c \"cd; %s\"", options.sudoPattern, options.sudoUser, cmd)
	watcher := Watcher{Pattern: options.sudoPattern, Response: options.sudoPassword}
	options.Watchers = append(options.Watchers, watcher)

	return conn.Run(cmd, WithHide(options.hide), WithWatchers(options.Watchers...))
}

// Scp 实现本地文件/目录上传到远程服务器
// 如果source是文件, target是文件, 直接调用copy
// 如果source是文件, target是目录, target 需要加文件名: path.Join(target, path.Base(source))
// 如果source是目录, target是文件, 报错,目标不是一个路径
// 如果source是目录, target是目录, 直接遍历copy, 如果target不存在,则自动创建。 如果存在,则创建下一级与 path.Join(target, path.Base(source)) 同名的目录(如果存在, 则报错)
func (conn *Connection) Scp(source, target string) error {
	// 通过ssh协议传输文件的目标机器全都是linux系统, 所以将目标路径强制转换为Linux格式
	target = filepath.ToSlash(target)

	if source == "" {
		return fmt.Errorf("源文件不能为空")
	}
	if target == "" {
		return fmt.Errorf("目标路径不能为空")
	}
	if !fileutil.IsExists(source) {
		return fmt.Errorf("文件 %s 不存在", source)
	}

	if !fileutil.IsDir(source) {
		if conn.IsDir(target) {
			target = filepath.ToSlash(path.Join(target, path.Base(filepath.ToSlash(source))))
		}
		return conn.Copy(source, target)
	}

	if !conn.IsExists(target) {
		return conn.LoopCopy(source, target)
	}

	if !conn.IsDir(target) {
		return fmt.Errorf("远程已经存在同名文件: %s", target)
	}

	target = filepath.ToSlash(path.Join(target, path.Base(filepath.ToSlash(source))))
	return conn.LoopCopy(source, target)
}

func (conn *Connection) singleCopy(source, target string, path string, info os.FileInfo) error {
	relative, err := filepath.Rel(source, path)
	if err != nil {
		fmt.Println("获取相对路径失败: ", err)
	}

	if info.IsDir() {
		if err := conn.MkdirAll(filepath.ToSlash(filepath.Join(target, relative))); err != nil {
			return err
		}
	} else {
		dir, _ := filepath.Split(filepath.Join(target, relative))
		if err := conn.MkdirAll(filepath.ToSlash(dir)); err != nil {
			return err
		}
		if err := conn.Copy(path, filepath.ToSlash(filepath.Join(target, relative))); err != nil {
			return err
		}
	}
	return err
}

func (conn *Connection) LoopCopy(source, target string) error {
	if err := filepath.Walk(source, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		return conn.singleCopy(source, target, path, info)
	}); err != nil {
		return err
	}
	return nil
}

func (conn *Connection) Copy(source, target string) error {
	var sf *os.File
	var df *sftp.File
	var err error

	// 通过ssh协议传输文件的目标机器全都是linux系统, 所以将目标路径强制转换为Linux格式
	target = filepath.ToSlash(target)

	if sf, err = os.Open(source); err != nil {
		return err
	}
	defer sf.Close()

	if df, err = conn.Create(target); err != nil {
		return err
	}
	defer df.Close()
	if _, err = df.ReadFrom(sf); err != nil {
		return err
	}
	return nil
}

func (conn *Connection) IsExists(path string) bool {
	_, err := conn.Stat(path)
	return err == nil
}

func (conn *Connection) IsDir(path string) bool {
	s, err := conn.Stat(path)
	if err != nil {
		return false
	}
	return s.IsDir()
}

func (conn *Connection) IsEmpty(path string) (bool, error) {
	fs, err := conn.ReadDir(path)
	if err != nil {
		return false, err
	}
	if len(fs) == 0 {
		return true, nil
	}
	return false, err
}

func (conn *Connection) IsDirEmptyOrNotExists(dir string) error {
	dir = filepath.ToSlash(dir)
	if !conn.IsExists(dir) {
		return nil
	}

	if !conn.IsDir(dir) {
		return fmt.Errorf("在机器: %s 上, 指定的路径(%s)不是目录", conn.Host, dir)
	}

	empty, err := conn.IsEmpty(dir)
	if err != nil {
		return err
	}
	if !empty {
		return fmt.Errorf("在机器: %s 上, 数据目录(%s)不为空", conn.Host, dir)
	}
	return nil
}

func (conn *Connection) ClearDir(dir string) error {
	if dir == "" {
		return nil
	}
	if slices.Contains(gocmd.SystemDirs, strings.TrimSuffix(strings.TrimSpace(dir), "/")) {
		return fmt.Errorf("目录(%s)是系统目录， 不允许删除", dir)
	}

	if !conn.IsDir(filepath.ToSlash(dir)) {
		return fmt.Errorf("在远程机器 %s 上: %s 不是一个目录", conn.Host, dir)
	}

	cmd := fmt.Sprintf("cd %s; rm -rf *", strutil.Quote(filepath.ToSlash(dir)))
	if stdout, stderr, err := conn.Run(cmd); err != nil {
		return fmt.Errorf(GOSSH_ERR_FORMAT, conn.Host, cmd, err, stdout, stderr)
	}

	return nil
}

func (conn *Connection) Hostsanalysis(hostnamelist []string) error {
	file, err := conn.Open("/etc/hosts")
	if err != nil {
		return fmt.Errorf("无法打开/etc/hosts文件: %v ", err)
	}
	defer file.Close()

	// 创建一个Scanner来逐行读取文件内容
	scanner := bufio.NewScanner(file)
	hlist := []string{}
	// 逐行检查域名解析
	for scanner.Scan() {
		line := scanner.Text()
		// 跳过注释行和空行
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue // 跳过无效行
		}
		ip := fields[0]
		hostname := fields[1]
		hlist = append(hlist, hostname)
		addrs, err := net.LookupHost(hostname)
		if err != nil {
			fmt.Printf("无法解析域名 %s: %v\n", hostname, err)
			continue
		}

		if !conn.Contains(addrs, ip) {
			return fmt.Errorf("域名 %s 解析到的IP地址与期望的地址 %s 不一致", hostname, ip)
		}
	}

	for _, hn := range hostnamelist {
		if hn != "" {
			if !conn.Contains(hlist, hn) {
				return fmt.Errorf("域名 %s 在主机 %s 的 /etc/hosts 文件未配置解析", hn, conn.Host)
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("读取 /etc/hosts 文件时发生错误: %v", err)
	}

	return nil
}

func (conn *Connection) Contains(addrs []string, ip string) bool {
	panic("unimplemented")
}
