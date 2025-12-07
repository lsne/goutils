/*
@Author : lsne
@Date : 2020-12-04 19:48
*/

package netutil

import (
	"fmt"
	"net"
	"os"
	"strconv"

	"github.com/lsne/goutils/utils/convutil"
)

// 获取本地IP地址
func LocalIP() ([]string, error) {
	var ips []string
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return ips, err
	}
	for _, address := range addrs {
		if ipNet, ok := address.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			ips = append(ips, ipNet.IP.String())
		}
	}
	return ips, nil
}

// 验证是否为IP地址
func IsIPAddress(addr string) error {
	if net.ParseIP(addr) == nil {
		return fmt.Errorf("IP地址(%s)格式不正确", addr)
	}
	return nil
}

// IsIPv4 判断字符串是否为IPv4地址
func IsIPv4(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.To4() != nil
}

// IsIPv6 判断字符串是否为IPv6地址
func IsIPv6(addr string) bool {
	ip := net.ParseIP(addr)
	if ip == nil {
		return false
	}
	return ip.To4() == nil && ip.To16() != nil
}

// 是 IP:PORT 或 HOSTNAME:PORT 格式
func ValidHostPort(s string) bool {
	_, _, err := net.SplitHostPort(s)
	return err == nil
}

// 是 IP:PORT 格式
func ValidIPPort(s string) bool {
	host, portStr, err := net.SplitHostPort(s)
	if err != nil {
		return false
	}

	// Parse port
	port, err := strconv.Atoi(portStr)
	if err != nil || port < 1 || port > 65535 {
		return false
	}

	// Parse IP (must be IP, not hostname)
	ip := net.ParseIP(host)
	return ip != nil
}

// JoinIPPort 判断 ip, port 合法后, 组合成 IP:PORT 格式
func JoinIPPort(ip string, port uint16) (string, error) {
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return "", fmt.Errorf("invalid IP address: %q", ip)
	}
	return net.JoinHostPort(parsedIP.String(), fmt.Sprintf("%d", port)), nil
}

// SplitIPPort 拆分字符串并判断拆分后的 ip, port 是否合法
func SplitIPPort(addr string) (ip string, port uint16, err error) {
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid address format %q: %w", addr, err)
	}

	// Validate that 'host' is a valid IP (not a hostname)
	parsedIP := net.ParseIP(host)
	if parsedIP == nil {
		return "", 0, fmt.Errorf("host %q is not a valid IP address", host)
	}

	port, err = convutil.StringToUint16(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("%s, 不是可用的端口", portStr)
	}

	return parsedIP.String(), port, nil
}

// ResolveIPv6 resolves the given domain and returns the first public/global IPv6 address.
// Returns empty string and error if no valid IPv6 is found.
func ResolveIPv6(domain string) (string, error) {
	if domain == "" {
		return "", fmt.Errorf("domain is empty")
	}

	ips, err := net.LookupIP(domain)
	if err != nil {
		return "", fmt.Errorf("failed to resolve domain %q: %w", domain, err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("no IP addresses returned for %q", domain)
	}

	for _, ip := range ips {
		if ip.To4() == nil && isGlobalUnicast(ip) {
			return ip.String(), nil
		}
	}

	return "", fmt.Errorf("no global IPv6 address found for %q", domain)
}

// isGlobalUnicast checks if an IPv6 address is a global unicast address (not loopback, link-local, multicast, etc.)
func isGlobalUnicast(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsLoopback() && !ip.IsLinkLocalUnicast()
}

func CheckIPv6Environment() error {
	hostname, err := os.Hostname()
	if err != nil {
		return fmt.Errorf("failed to get hostname: %w", err)
	}

	ips, err := net.LookupIP(hostname)
	if err != nil {
		return fmt.Errorf("failed to resolve hostname %q: %w", hostname, err)
	}

	// Get all global IPv6 addresses from interfaces
	localIPv6Set := make(map[string]bool)
	ifaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to list network interfaces: %w", err)
	}

	for _, iface := range ifaces {
		addrs, err := iface.Addrs()
		if err != nil {
			continue // skip problematic interfaces
		}
		for _, addr := range addrs {
			if ipnet, ok := addr.(*net.IPNet); ok {
				ip := ipnet.IP
				if ip.To4() == nil && isGlobalUnicast(ip) {
					localIPv6Set[ip.String()] = true
				}
			}
		}
	}

	if len(localIPv6Set) == 0 {
		return fmt.Errorf("no global IPv6 address found on any interface")
	}

	// Check if any resolved IP matches local IPv6
	for _, ip := range ips {
		if ip.To4() == nil && localIPv6Set[ip.String()] {
			return nil // match found
		}
	}

	return fmt.Errorf("resolved IPv6 addresses for %q do not match any local global IPv6 address", hostname)
}
