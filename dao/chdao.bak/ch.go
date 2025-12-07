/*
 * @Author: lsne
 * @Date: 2024-09-10 17:09:20
 */

package chdao

import (
	"database/sql"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
)

// database := "db01"
// username := "user01"
// password := "Testaa.123"

type ChConn struct {
	HostPort []string
	Username string
	Password string
	Database string
	Conn     *sql.DB
}

func NewChConn(host []string, username, password, database string) (*ChConn, error) {
	var err error
	c := &ChConn{
		HostPort: host,
		Username: username,
		Password: password,
		Database: database,
	}

	c.Conn = clickhouse.OpenDB(&clickhouse.Options{
		Addr: c.HostPort,
		Auth: clickhouse.Auth{
			Database: c.Database,
			Username: c.Username,
			Password: c.Password,
		},
		// TLS: &tls.Config{
		// 	InsecureSkipVerify: true,
		// },
		Settings: clickhouse.Settings{
			"max_execution_time": 60,
		},
		DialTimeout: time.Second * 30,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
		Debug:                true,
		BlockBufferSize:      10,
		MaxCompressionBuffer: 10240,
		ClientInfo: clickhouse.ClientInfo{ // optional, please see Client info section in the README.md
			Products: []struct {
				Name    string
				Version string
			}{
				{Name: "my-app", Version: "0.1"},
			},
		},
	})

	c.Conn.SetMaxIdleConns(5)
	c.Conn.SetMaxOpenConns(10)
	c.Conn.SetConnMaxLifetime(time.Hour)

	err = c.Conn.Ping()
	return c, err
}

func (c *ChConn) Close() {
	_ = c.Conn.Close()
}

func (c *ChConn) Ping() error {
	return c.Conn.Ping()
}

func (c *ChConn) Exec(sql string, params ...interface{}) (result sql.Result, err error) {
	return c.Conn.Exec(sql, params...)
}
