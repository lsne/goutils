package netutil

import (
	"fmt"
	"net"
	"regexp"
	"strings"

	"github.com/lsne/goutils/utils/convutil"
)

// IsValidHostname 验证字符串是否是合法的主机名
func ValidHostname(hostname string) bool {
	if len(hostname) == 0 || len(hostname) > 253 {
		return false
	}

	// 基本格式检查
	if matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-\.]+$`, hostname); !matched {
		return false
	}

	// 检查标签（以点分隔的部分）
	labels := strings.Split(hostname, ".")
	for _, label := range labels {
		if len(label) == 0 || len(label) > 63 {
			return false
		}
		// 标签不能以连字符开头或结尾
		if strings.HasPrefix(label, "-") || strings.HasSuffix(label, "-") {
			return false
		}
	}

	return true
}

// IsValidHostnameWithDNS 验证主机名并通过DNS解析确认
func ResolveHostname(hostname string) bool {
	if !ValidHostname(hostname) {
		return false
	}

	// 尝试解析主机名
	addrs, err := net.LookupHost(hostname)
	return err == nil && len(addrs) > 0
}

// IsHostnameOrIP 验证字符串是否是合法的主机名或IP地址
func ValidHostnameOrIP(host string) bool {
	// 首先检查是否是IP地址
	if net.ParseIP(host) != nil {
		return true
	}

	// 验证是否是合法的主机名
	return ValidHostname(host)
}

// IsHostnameOrIPWithDNS 验证主机名或IP，对主机名进行DNS解析
func ResolveHostnameOrIP(host string) bool {
	if net.ParseIP(host) != nil {
		return true
	}

	return ResolveHostname(host)
}

func SplitHostnameOrIPList(s string) (hosts []string, err error) {
	if s == "" {
		return hosts, nil
	}

	list := strings.Split(s, ",")
	for _, l := range list {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if !ValidHostnameOrIP(l) {
			return hosts, fmt.Errorf(" (%s) IP地址或主机名格式不合法", s)
		}
		hosts = append(hosts, l)
	}
	return hosts, nil
}

// JoinIPPort 判断 ip, port 合法后, 组合成 IP(HOST):PORT 格式
func JoinHostPort(host string, port uint16) (string, error) {
	if net.ParseIP(host) == nil && !ValidHostname(host) {
		return "", fmt.Errorf("invalid IP address: %q", host)
	}
	return net.JoinHostPort(host, fmt.Sprintf("%d", port)), nil
}

// SplitIPPort 拆分字符串并判断拆分后的 ip(host), port 是否合法
func SplitHostPort(addr string) (host string, port uint16, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid address format %q: %w", addr, err)
	}

	if net.ParseIP(host) == nil && !ValidHostname(host) {
		return "", 0, fmt.Errorf("host %q is not a valid IP address or hostname", host)
	}

	port, err = convutil.StringToUint16(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("%s, 不是可用的端口", portStr)
	}

	return host, port, nil
}
