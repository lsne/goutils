/*
 * @Author: lsne
 * @Date: 2025-12-06 19:16:27
 */

package redisdao

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/lsne/goutils/utils/convutil"

	"github.com/redis/go-redis/v9"
)

type RedisClient struct {
	Host        string
	Port        uint16
	Password    string
	Timeout     int
	client      *redis.Client
	monitorConn *redis.Conn
	monitorCmd  *redis.MonitorCmd
}

func NewRedisClient(host string, port uint16, password string, timeout int) (*RedisClient, error) {
	c := &RedisClient{
		Host:     host,
		Port:     port,
		Password: password,
		Timeout:  timeout,
	}

	c.client = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%d", c.Host, c.Port),
		Password:     c.Password,
		MaxRetries:   -1,
		DialTimeout:  time.Duration(c.Timeout) * time.Second,
		ReadTimeout:  time.Duration(c.Timeout) * time.Second,
		WriteTimeout: time.Duration(c.Timeout) * time.Second,
	})

	return c, nil
}

func (c *RedisClient) Close() {
	if c.client != nil {
		_ = c.client.Close()
	}
}

func (c *RedisClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	_, err := c.client.Ping(ctx).Result()
	return err
}

func (c *RedisClient) ConfigRewrite() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	if _, err := c.client.ConfigRewrite(ctx).Result(); err != nil {
		return err
	}
	return nil
}

func (c *RedisClient) ReplicaOf(master string, port uint16) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	if _, err := c.client.SlaveOf(ctx, master, fmt.Sprintf("%d", port)).Result(); err != nil {
		return err
	}
	return c.ConfigRewrite()
}

func (c *RedisClient) ClusterMyID() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	return c.client.ClusterMyID(ctx).Result()
}

func (c *RedisClient) SlaveStatus() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	replicationInfo, err := c.client.InfoMap(ctx, "Replication").Result()
	if err != nil {
		return "", fmt.Errorf("failed to get replication info: %w", err)
	}

	replicationSection, ok := replicationInfo["Replication"]
	if !ok {
		return "", errors.New("missing 'Replication' section in INFO response")
	}

	status, ok := replicationSection["master_link_status"]
	if !ok {
		return "", errors.New("missing 'master_link_status' field")
	}

	return status, nil
}

func (c *RedisClient) Info(section string) (map[string]map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	return c.client.InfoMap(ctx, section).Result()
}

func (c *RedisClient) SlaveOfOK() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	replicationInfo, err := c.client.InfoMap(ctx, "Replication").Result()
	if err != nil {
		return false, fmt.Errorf("failed to get replication info: %w", err)
	}

	replicationSection, ok := replicationInfo["Replication"]
	if !ok {
		return false, errors.New("missing 'Replication' section in INFO response")
	}

	status, ok := replicationSection["master_link_status"]
	if !ok {
		return false, errors.New("missing 'master_link_status' field")
	}

	sync_progress, ok := replicationSection["master_sync_in_progress"]
	if !ok {
		return false, errors.New("missing 'master_sync_in_progress' field")
	}

	attempts, ok := replicationSection["master_current_sync_attempts"]
	if !ok {
		return false, errors.New("missing 'master_current_sync_attempts' field")
	}

	if status == "up" || (status == "down" && sync_progress == "1" && attempts == "1") {
		return true, nil
	}

	return false, nil
}

func (c *RedisClient) SlaveIPs() (ips []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	replicationInfo, err := c.client.InfoMap(ctx, "Replication").Result()
	if err != nil {
		return ips, fmt.Errorf("failed to get replication info: %w", err)
	}

	replicationSection, ok := replicationInfo["Replication"]
	if !ok {
		return ips, errors.New("missing 'Replication' section in INFO response")
	}

	r, _ := regexp.Compile("^slave[0-9]+$")

	for key, value := range replicationSection {
		if ok := r.MatchString(key); !ok {
			continue
		}
		slaveline := strings.Split(value, ",")

		ipinfo := strings.Split(slaveline[0], "=")
		if len(ipinfo) < 2 {
			continue
		}
		ip := ipinfo[1]
		portinfo := strings.Split(slaveline[1], "=")
		if len(portinfo) < 2 {
			continue
		}
		port := portinfo[1]

		ips = append(ips, ip+":"+port)
	}

	return ips, nil
}

func (c *RedisClient) ClusterNodes() (nodes []ClusterNode, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	s, err := c.client.ClusterNodes(ctx).Result()
	if err != nil {
		return nodes, err
	}

	for _, line := range strings.Split(s, "\n") {
		if line == "" {
			continue
		}
		nodeInfo := strings.Split(line, " ")
		hostPort := strings.Split(strings.Split(nodeInfo[1], "@")[0], ":")
		port, err := convutil.StringToUint16(hostPort[1])
		if err != nil {
			return nodes, err
		}

		// role := nodeInfo[2]
		// if strings.Contains(role, ",") {
		// 	roles := strings.Split(role, ",")
		// 	if len(roles) >= 2 && roles[0] == "myself" {
		// 		role = roles[1]
		// 	} else {
		// 		role = roles[0]
		// 	}
		// }

		var role string = "master"
		var fail bool = false
		if strings.Contains(nodeInfo[2], "slave") {
			role = "slave"
		}

		if strings.Contains(nodeInfo[2], "fail") {
			fail = true
		}

		nodes = append(nodes, ClusterNode{
			ClusterID: nodeInfo[0],
			Host:      hostPort[0],
			Port:      port,
			Role:      role,
			Fail:      fail,
			MasterID:  nodeInfo[3],
			Connected: nodeInfo[7],
		})
	}
	return nodes, nil
}

func (c *RedisClient) ClusterForget(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	_, err := c.client.ClusterForget(ctx, id).Result()
	return err
}

func (c *RedisClient) IsMaster() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	replicationInfo, err := c.client.InfoMap(ctx, "Replication").Result()
	if err != nil {
		return false, fmt.Errorf("failed to get replication info: %w", err)
	}

	replicationSection, ok := replicationInfo["Replication"]
	if !ok {
		return false, errors.New("missing 'Replication' section in INFO response")
	}

	role, ok := replicationSection["role"]
	if !ok {
		return false, errors.New("missing 'role' field")
	}

	return role == "master", nil
}

func (c *RedisClient) GetAOFStatus() (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()

	replicationInfo, err := c.client.InfoMap(ctx, "Persistence").Result()
	if err != nil {
		return false, fmt.Errorf("failed to get replication info: %w", err)
	}
	replicationSection, ok := replicationInfo["Persistence"]
	if !ok {
		return false, errors.New("missing 'Persistence' section in INFO response")
	}
	progressed, ok := replicationSection["aof_rewrite_in_progress"]
	if !ok {
		return false, errors.New("missing 'aof_rewrite_in_progress' field")
	}
	return progressed == "0", nil
}

func (c *RedisClient) FlushAOF() error {
	stat := false
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(c.Timeout)*time.Second)
	defer cancel()
	_, err := c.client.BgRewriteAOF(ctx).Result()
	if err != nil {
		return err
	}

	for i := 1; i <= 60; i++ {
		time.Sleep(3 * time.Second)
		if f, err := c.GetAOFStatus(); err != nil {
			return err
		} else {
			if f {
				stat = true
				break
			} else {
				fmt.Println("等到AOF文件持久化完成...")
			}
		}
	}

	if !stat {
		return fmt.Errorf("AOF 一直未持久化完成,请检查")
	}

	return nil
}

// Monitor 执行 redis 的  monitor 命令
// 可以使用 for 遍历返回信息
//
//	 在执行 Start() 后开始监听
//		for v := range ch {
//			fmt.Println("monitor: ", v)
//		}
func (c *RedisClient) Monitor(ch chan string) {
	c.monitorConn = c.client.Conn()
	c.monitorCmd = c.monitorConn.Monitor(context.Background(), ch)
}

func (c *RedisClient) MonitorStart() {
	c.monitorCmd.Start()
}

func (c *RedisClient) MonitorStop() {
	c.monitorCmd.Stop()
}

func (c *RedisClient) MonitorClose() error {
	return c.monitorConn.Close()
}
