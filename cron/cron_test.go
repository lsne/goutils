/*
 * @Author: lsne
 * @Date: 2023-12-12 00:19:37
 */

package cron

import (
	"fmt"
	"time"
)

func cronTest() {
	c := NewCron()

	c.cron.AddFunc("@every 1s", func() {
		fmt.Println("tick every 1 second")
	})

	c.Start()
	time.Sleep(time.Second * 5)
}
