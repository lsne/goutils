/*
 * @Author: lsne
 * @Date: 2024-09-05 10:34:12
 */

package main

import (
	"fmt"
	"testgo/testclickhouse/ch"
)

func main() {
	// 定义多个 ClickHouse 节点
	nodes := []string{
		"1.1.1.1:9000",
		"1.1.1.2:9000",
		"1.1.1.3:9000",
		"1.1.1.4:9000",
	}

	username := "user01"
	password := "123456"
	database := "db01"

	conn, err := ch.NewChConn(nodes, username, password, database)
	if err != nil {
		fmt.Println(err)
		panic("创建连接失败")
	}

	defer conn.Close()

	// 插入数据
	sql := "INSERT INTO t1_all (content, count) VALUES (?, ?)"
	if _, err := conn.Exec(sql, "test1", 1); err != nil {
		fmt.Printf("insert %s row %d faild!\n", table, i)
	} else {
		fmt.Printf("insert %s row %d success!\n", table, i)
	}
}
