package fileutil

import (
	"fmt"

	"gopkg.in/ini.v1"
)

// INILoadFromFile 从INI配置文件加载配置到结构体
func INILoadFromFile(filename string, config interface{}, iniConfig ini.LoadOptions) error {
	cfg, err := ini.LoadSources(iniConfig, filename)
	if err != nil {
		return fmt.Errorf("加载ini文件(%s)失败: %v", filename, err)
	}

	if err = cfg.MapTo(config); err != nil {
		return fmt.Errorf("将ini文件(%s)映射到结构体对象(%t)失败: %v", filename, config, err)
	}
	return nil
}

// INISaveToFile 保存结构体数据到操作系统INI配置文件
func INISaveToFile(filename string, config interface{}) error {
	cfg := ini.Empty(ini.LoadOptions{IgnoreInlineComment: true}) //AllowNestedValues: true 允许嵌套值,应该没用
	if err := ini.ReflectFrom(cfg, config); err != nil {
		return fmt.Errorf("结构体对象(%t)映射到ini对象(%s) 错误: %v", config, filename, err)
	}
	if err := cfg.SaveTo(filename); err != nil {
		return fmt.Errorf("对象(%t)保存到(%s)文件错误: %v", config, filename, err)
	}
	return nil
}
