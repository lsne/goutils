package strutil

import (
	"fmt"
	"math/rand"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

const (
	digits = "0123456789"
	lowers = "abcdefghijklmnopqrstuvwxyz"
	uppers = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	// symbols   = "!@#$%^&*()_+-=[]{}|;:,.<>?~"  // TODO: 确定使用哪一个 symbols
	symbols  = "!#$%&*+-.:<=>?@^_|~" // 移除了可能引起问题的字符
	letters  = lowers + uppers
	allChars = letters + digits + symbols
)

// GenerateString 生成纯字母随机字符串（大小写）, 至少4位
func GenerateString(length int) string {
	if length < 4 {
		length = 4
	}
	buf := make([]byte, length)
	for i := range buf {
		buf[i] = letters[rand.Intn(len(letters))]
	}
	return string(buf)
}

// GeneratePasswd 生成满足复杂度要求的随机密码, 至少4位
// 要求：至少包含1个数字、1个特殊字符、1个小写、1个大写字母
func GeneratePasswd(length int) string {
	if length < 4 {
		length = 4
	}

	// 确保每类字符至少出现一次
	buf := make([]byte, length)
	buf[0] = digits[rand.Intn(len(digits))]
	buf[1] = symbols[rand.Intn(len(symbols))]
	buf[2] = lowers[rand.Intn(len(lowers))]
	buf[3] = uppers[rand.Intn(len(uppers))]

	// 剩余位置随机填充
	for i := 4; i < length; i++ {
		buf[i] = allChars[rand.Intn(len(allChars))]
	}

	// 打乱顺序（可选，但推荐以避免固定模式）
	rand.Shuffle(len(buf), func(i, j int) {
		buf[i], buf[j] = buf[j], buf[i]
	})

	return string(buf)
}

// ValidatePassword
// CheckPasswordLevel 校验密码复杂度
func ValidatePassword(ps string) error {
	const minLen = 16
	if len(ps) < minLen {
		example := GeneratePasswd(minLen)
		return fmt.Errorf("超级管理员密码必须大于等于%d位, 且包含大小写字母、数字、特殊字符；当前长度：%d; 示例：%s",
			minLen, len(ps), example)
	}

	var (
		hasDigit  bool
		hasLower  bool
		hasUpper  bool
		hasSymbol bool
	)

	for _, r := range ps {
		switch {
		case unicode.IsDigit(r):
			hasDigit = true
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case strings.ContainsRune(symbols, r):
			hasSymbol = true
		}
		// 提前退出优化（可选）
		if hasDigit && hasLower && hasUpper && hasSymbol {
			break
		}
	}

	if !hasDigit || !hasLower || !hasUpper || !hasSymbol {
		example := GeneratePasswd(minLen)
		missing := []string{}
		if !hasDigit {
			missing = append(missing, "数字")
		}
		if !hasLower {
			missing = append(missing, "小写字母")
		}
		if !hasUpper {
			missing = append(missing, "大写字母")
		}
		if !hasSymbol {
			missing = append(missing, "特殊字符")
		}
		return fmt.Errorf("超级管理员密码缺少: %s; 示例: %s", strings.Join(missing, "、"), example)
	}

	return nil
}

// parseMemorySize 通用的内存字符串解析函数
func ParseMemorySizeMB(memoryStr string) (int, error) {
	// 编译正则表达式（可考虑提前编译为全局变量以提升性能）
	re := regexp.MustCompile(MemoryUnitPattern)
	matches := re.FindStringIndex(memoryStr)
	if matches == nil {
		return 0, fmt.Errorf("内存参数格式错误，必须包含有效单位后缀（如 MB 或 GB）: %q", memoryStr)
	}

	// 提取数值部分（可能包含前后空格）
	numStr := strings.TrimSpace(memoryStr[:matches[0]])
	if numStr == "" {
		return 0, fmt.Errorf("内存参数中缺少有效数值: %q", memoryStr)
	}

	// 解析数值（支持负数？业务上应禁止）
	num, err := strconv.Atoi(numStr)
	if err != nil {
		return 0, fmt.Errorf("内存数值格式无效（应为整数）: %q, 错误: %w", numStr, err)
	}
	if num <= 0 {
		return 0, fmt.Errorf("内存大小必须为正整数，当前值: %d", num)
	}

	// 提取并标准化单位
	unit := strings.ToUpper(memoryStr[matches[0]:])
	switch unit {
	case "M", "MB":
		// 单位已是 MB，无需转换
	case "G", "GB":
		num *= 1024 // 转换为 MB
	default:
		// 理论上不会走到这里（正则已限制），但保留防御性检查
		return 0, fmt.Errorf("不支持的内存单位: %q, 仅支持 M/MB/G/GB", unit)
	}
	return num, nil
}

// Quote 将字符串转义为安全的 shell 单引号形式（类似 Python shlex.quote）
func Quote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "'\"'\"'") + "'"
}

// ResolveLogPath 根据 base 目录和日志路径，返回日志的绝对路径。
// - 如果 logfile 是绝对路径，直接返回；
// - 否则，将其解析为相对于 dir 的路径，并返回规范化后的绝对路径。
func ResolveLogPath(basedir, dir string) string {
	if filepath.IsAbs(dir) {
		return dir
	}
	// 拼接并规范化路径（自动处理 ../）
	return filepath.Clean(filepath.Join(basedir, dir))
}
