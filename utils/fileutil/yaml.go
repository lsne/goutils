package fileutil

import (
	"os"

	"gopkg.in/yaml.v2"
)

// YAMLLoadFromFile 从 YAML 配置文件加载配置到结构体
func YAMLLoadFromFile(filename string, config interface{}) error {
	content, err := os.ReadFile(filename)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(content, config)
}

// YAMLSaveToFile 保存结构体数据到操作系统 YAML 配置文件
func YAMLSaveToFile(filename string, config interface{}) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(filename, data, 0777)
}
