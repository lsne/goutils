/*
 * @Author: lsne
 * @Date: 2025-12-07 14:55:13
 */

package systemd

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/lsne/goutils/common/system"
	"github.com/lsne/goutils/environment"
	"github.com/lsne/goutils/utils/fileutil"

	"gopkg.in/ini.v1"
)

// systemd 相关常量
const (
	SystemdPath         = "/usr/lib/systemd/system"
	SystemdTemplatePath = "../systemd"
)

// SystemdService 表示一个 systemd 服务配置
type SystemdService struct {
	name        string
	template    string
	servicePath string
	User        string
	Group       string
	ExecStart   string
	ExecReload  string
	WorkingDir  string
	ExtraEnvs   []string
	service     *ini.File
}

func NewSystemdService(name, tmplfile string) (*SystemdService, error) {
	tmpl := filepath.Join(environment.GlobalEnv().ProgramPath, SystemdTemplatePath, tmplfile)
	cfg, err := ini.LoadSources(ini.LoadOptions{
		AllowShadows:             true,
		SpaceBeforeInlineComment: true,
	}, tmpl)
	if err != nil {
		return &SystemdService{}, fmt.Errorf("加载ini文件(%s)失败: %v", tmpl, err)
	}
	return &SystemdService{name: name, template: tmplfile, servicePath: filepath.Join(SystemdPath, name), ExtraEnvs: make([]string, 0), service: cfg}, err
}

func (s *SystemdService) FormatBody() error {
	section := s.service.Section("Service")

	if s.User == "" {
		return fmt.Errorf("systemd User 不能为空")
	}

	if s.Group == "" {
		return fmt.Errorf("systemd Group 不能为空")
	}

	if s.ExecStart == "" {
		return fmt.Errorf("systemd ExecStart 不能为空")
	}

	section.Key("User").SetValue(s.User)
	section.Key("Group").SetValue(s.Group)
	section.Key("ExecStart").SetValue(s.ExecStart)

	if s.WorkingDir != "" {
		section.Key("WorkingDirectory").SetValue(s.WorkingDir)
	}

	if s.ExecReload != "" {
		section.Key("ExecReload").SetValue(s.ExecReload)
	}

	if section.Key("WorkingDirectory").String() == "" {
		section.DeleteKey("WorkingDirectory")
	}

	if section.Key("ExecReload").String() == "" {
		section.DeleteKey("ExecReload")
	}

	// 添加环境变量
	var e struct {
		Environment []string `ini:"Environment,allowshadow"`
	}
	e.Environment = append(e.Environment, section.Key("Environment").ValueWithShadows()...)
	e.Environment = append(e.Environment, s.ExtraEnvs...)
	return section.ReflectFrom(&e)
}

func (s *SystemdService) Save() error {
	if err := s.FormatBody(); err != nil {
		return err
	}
	if err := s.service.SaveTo(s.servicePath); err != nil {
		return err
	}
	return s.DaemonReload()
}

func (s *SystemdService) Remove() error {
	if !s.IsExists() {
		return nil
	}
	if err := fileutil.MoveToBackup(s.servicePath); err != nil {
		return err
	}
	return s.DaemonReload()
}

func (s *SystemdService) IsExists() bool {
	return fileutil.IsExists(s.servicePath)
}

func (s *SystemdService) DaemonReload() error {
	return system.SystemdDaemonReload()
}

func (s *SystemdService) Enable() error {
	return system.SystemCtl(s.name, "enable")
}

func (s *SystemdService) Disable() error {
	return system.SystemCtl(s.name, "disable")
}

func (s *SystemdService) Start() error {
	if err := system.SystemCtl(s.name, "start"); err != nil {
		return err
	}
	// 等待 5 秒, 防止进程还未就绪
	time.Sleep(5 * time.Second)
	return nil
}

func (s *SystemdService) Stop() error {
	if err := system.SystemCtl(s.name, "stop"); err != nil {
		return fmt.Errorf("停止 systemd 服务(%s)失败： %w", s.name, err)
	}
	// 等待 5 秒, 防止进程还未就绪
	time.Sleep(5 * time.Second)
	return nil
}

func (s *SystemdService) EnableAndStart() error {
	if err := s.Enable(); err != nil {
		return fmt.Errorf("启用 systemd 服务(%s)失败: %w", s.name, err)
	}
	return s.Start()
}

func (s *SystemdService) DisableAndStop() error {
	if err := s.Stop(); err != nil {
		return fmt.Errorf("停止 systemd 服务(%s)失败: %w", s.name, err)
	}
	return s.Disable()
}

func (s *SystemdService) ResourceLimit(limit string) error {
	return system.SystemResourceLimit(s.name, limit)
}
