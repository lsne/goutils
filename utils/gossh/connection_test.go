/*
 * @Author: lsne
 * @Date: 2025-11-10 20:51:05
 */

package gossh

import (
	"fmt"
	"testing"
)

func TestNewConnection(t *testing.T) {
	conn, err := NewConnection("192.168.0.2", 22, "username", "password", "", 10)
	if err != nil {
		fmt.Println("连接失败 ", err)
		return
	}

	// conn, err := NewConnection("192.168.0.2", 22, "username", "", "/home/username/.ssh/id_rsa", 10)
	// if err != nil {
	// 	fmt.Println("连接失败 ", err)
	// 	return
	// }

	// 普通用户执行命令
	sdtout, sdterr, err := conn.Run("ls -l /tmp/")
	fmt.Println("错误信息: ", err)
	fmt.Println("输出信息:", string(sdterr))
	fmt.Println("输出信息:", string(sdtout))

	// sudo 执行命令
	sdtout, sdterr, err = conn.Sudo("ls -l /root/")
	fmt.Println("错误信息: ", err)
	fmt.Println("输出信息:", string(sdterr))
	fmt.Println("输出信息:", string(sdtout))

	// scp 文件到远程服务器
	// conn.Scp("E:\\test.a", "/home/ls/abc")
}
