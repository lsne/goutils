/*
 * @Author: lsne
 * @Date: 2023-12-11 22:56:36
 */

package cron

import (
	"github.com/robfig/cron/v3"
)

type Cron struct {
	cron *cron.Cron
}

func NewCron() *Cron {
	return &Cron{cron: cron.New()}
}

func (c *Cron) Start() {
	c.cron.Start()
}

func (c *Cron) Stop() {
	c.cron.Stop()
}
