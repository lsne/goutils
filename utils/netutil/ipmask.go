package netutil

import (
	"fmt"
	"net"
	"strings"
)

// IsIPWithMask 验证字符串是否为 IP/MASK 格式
func IsIPWithMask(cidrStr string) error {
	_, _, err := net.ParseCIDR(cidrStr)
	return err
}

func ValidateIPOrCIDR(s string) error {
	if strings.Count(s, "/") == 0 {
		// Plain IP
		if ip := net.ParseIP(s); ip != nil {
			return nil
		}
		return fmt.Errorf("invalid IP address: %q", s)
	}

	if strings.Count(s, "/") == 1 {
		// CIDR
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return fmt.Errorf("invalid CIDR: %w", err)
		}
		return nil
	}

	return fmt.Errorf("invalid format: expected IP or IP/mask, got %q", s)
}

func inferIPv4CIDR(ip net.IP) string {
	ip4 := ip.To4()
	if ip4 == nil {
		return ip.String() + "/128" // fallback to IPv6
	}

	a, b, c, d := ip4[0], ip4[1], ip4[2], ip4[3]
	switch {
	case d != 0:
		return ip.String() + "/32"
	case c != 0:
		return ip.String() + "/24"
	case b != 0:
		return ip.String() + "/16"
	case a != 0:
		return ip.String() + "/8"
	default: // 0.0.0.0
		return "0.0.0.0/0"
	}
}

func EnsureIPWithMask(s string) (string, error) {
	if strings.Contains(s, "/") {
		// Already has mask, validate it
		_, _, err := net.ParseCIDR(s)
		if err != nil {
			return "", err
		}
		return s, nil
	}

	// No mask: parse as IP
	ip := net.ParseIP(s)
	if ip == nil {
		return "", fmt.Errorf("invalid IP address: %q", s)
	}

	if ip.To4() != nil {
		return inferIPv4CIDR(ip), nil
	}
	// IPv6
	return ip.String() + "/128", nil
}
