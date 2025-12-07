package netutil

import (
	"fmt"
	"net"
	"time"
)

// 按顺序选择一个可用的端口号, 到3万还没选出来,就还是用默认吧
func RandomPort(port uint16) uint16 {
	for p := port; p <= 30000; p++ {
		if LocalPortAvailable(p) {
			return p
		}
	}
	return port
}

// LocalPortAvailable 检测端口是否可用
func LocalPortAvailable(port uint16) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return false
	}
	defer ln.Close()
	return true
}

// 连接远程机器端口号
func CanConnectToTCP(address string) (results bool, err error) {
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false, err
	}
	conn.Close()
	return true, nil
}
