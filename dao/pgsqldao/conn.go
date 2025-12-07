package pgsqldao

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/jackc/pgx/v5"
)

type PostgreSQLClient struct {
	Host     string
	Port     uint16
	User     string
	Password string
	Dbname   string
	Info     string
	conn     *pgx.Conn
}

func NewPostgreSQLClient(host string, port uint16, user, password, dbname string) (*PostgreSQLClient, error) {
	var err error
	c := &PostgreSQLClient{
		Host:     host,
		Port:     port,
		User:     user,
		Password: password,
		Dbname:   dbname,
	}

	// c.Info = fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=%s", c.Host, c.Port, c.User, c.Password, c.Dbname, "prefer")

	eUser := url.QueryEscape(c.User)
	ePass := url.QueryEscape(c.Password)
	url := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s", eUser, ePass, c.Host, c.Port, c.Dbname, "prefer")

	c.conn, err = pgx.Connect(context.Background(), url)
	if err != nil {
		return c, fmt.Errorf("unable to connect to database: %w", err)
	}
	return c, nil
}

func (p *PostgreSQLClient) Close() {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()

	if p.conn != nil {
		if err := p.conn.Close(ctx); err != nil {
			fmt.Println(err)
		}
	}
}

func (p *PostgreSQLClient) Ping() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(10)*time.Second)
	defer cancel()
	return p.conn.Ping(ctx)
}

func (p *PostgreSQLClient) Select() error {
	var r int
	return p.conn.QueryRow(context.Background(), "select 1;").Scan(&r)
}

func (p *PostgreSQLClient) VersionString() (string, error) {
	var version string
	sql := "show server_version;"
	err := p.conn.QueryRow(context.Background(), sql).Scan(&version)
	return version, err
}

func (p *PostgreSQLClient) VersionInt() (version uint64, err error) {
	sql := "SELECT current_setting('server_version_num');"
	err = p.conn.QueryRow(context.Background(), sql).Scan(&version)
	return version, err
}

func (p *PostgreSQLClient) ReloadConfig() error {
	sql := "select pg_reload_conf();"
	var success bool
	if err := p.conn.QueryRow(context.Background(), sql).Scan(&success); err != nil {
		return err
	}
	if !success {
		return fmt.Errorf("执行 select pg_reload_conf() 失败")
	}
	return nil
}

// Promote 提升为主库,参数示例: ("true", 60)
func (p *PostgreSQLClient) Promote(wait string, seconds int) error {
	var success bool
	sql := "select pg_promote($1,$2);"
	// sql := fmt.Sprintf("select pg_promote($1,$2);", wait, seconds)
	if err := p.conn.QueryRow(context.Background(), sql, wait, seconds).Scan(&success); err != nil {
		return err
	}
	if !success {
		return errors.New("提升为主库失败")
	}
	return nil
}

func (p *PostgreSQLClient) IsSlave() (bool, error) {
	var slave bool
	sql := "SELECT pg_is_in_recovery();"
	err := p.conn.QueryRow(context.Background(), sql).Scan(&slave)
	if err != nil {
		return false, fmt.Errorf("判断是否为从库失败: %v", err)
	}
	return slave, nil
}

func (p *PostgreSQLClient) DBExist(dbname string) (bool, error) {
	var n int
	sql := "select count(*) from pg_catalog.pg_database where datname = $1;"
	err := p.conn.QueryRow(context.Background(), sql, dbname).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("查询数据库是否存在失败: %v", err)
	}
	return n != 0, nil
}

func (p *PostgreSQLClient) UserExist(username string) (bool, error) {
	var n int
	sql := "select count(*) from pg_catalog.pg_user where usename = $1;"
	err := p.conn.QueryRow(context.Background(), sql, username).Scan(&n)
	if err != nil {
		return false, fmt.Errorf("获取用户列表失败: %v", err)
	}
	return n != 0, nil
}

func (p *PostgreSQLClient) IsReplicationGrant(username string) bool {
	var repl bool
	sql := "select userepl from pg_catalog.pg_user where usename= $1"
	err := p.conn.QueryRow(context.Background(), sql, username).Scan(&repl)
	if err != nil {
		return false
	}
	return repl
}

func (p *PostgreSQLClient) CreateDB(dbname string) error {
	sql := fmt.Sprintf("CREATE DATABASE \"%s\";", dbname)
	_, err := p.conn.Exec(context.Background(), sql)

	return err
}

func (p *PostgreSQLClient) CreateDBWithOwner(username, dbname string) error {
	sql := fmt.Sprintf("CREATE DATABASE \"%s\"  OWNER  %s;", dbname, username)
	_, err := p.conn.Exec(context.Background(), sql)
	return err
}

func (p *PostgreSQLClient) GrantDBUser(username, dbname string) error {
	sql := fmt.Sprintf("ALTER DATABASE \"%s\"  OWNER TO  %s;", dbname, username)
	_, err := p.conn.Exec(context.Background(), sql)
	return err
}

func (p *PostgreSQLClient) CreateUser(username, password, privileges string) error {
	var sql string
	if privileges == "DBUSER" {
		sql = fmt.Sprintf("CREATE USER \"%s\"  ENCRYPTED password '%s';", username, password)
	} else {
		sql = fmt.Sprintf("CREATE USER \"%s\" WITH %s ENCRYPTED password '%s';", username, privileges, password)
	}
	_, err := p.conn.Exec(context.Background(), sql)
	return err
}

func (p *PostgreSQLClient) AlterUserExpireAt(username, expireAt string) error {
	sql := fmt.Sprintf("alter user %s with valid until '%s';", username, expireAt)
	_, err := p.conn.Exec(context.Background(), sql)
	return err
}

func (p *PostgreSQLClient) AlterPassword(username, password string) error {
	sql := fmt.Sprintf("ALTER USER %s WITH PASSWORD '%s';", username, password)
	_, err := p.conn.Exec(context.Background(), sql)
	return err
}

func (p *PostgreSQLClient) DBSize() (uint64, error) {
	var size uint64
	sql := "select sum(pg_database_size(datname)) from pg_catalog.pg_database;"
	err := p.conn.QueryRow(context.Background(), sql).Scan(&size)
	if err != nil {
		return 0, fmt.Errorf("获取PG大小失败: %v", err)
	}
	return size, nil
}

func (p *PostgreSQLClient) ReplicationIp() ([]string, error) {
	var repl []string
	sql := "select client_addr from pg_catalog.pg_stat_replication;"
	rows, err := p.conn.Query(context.Background(), sql)
	if err != nil {
		return repl, fmt.Errorf("获取从库数量失败: %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return repl, err
		}
		repl = append(repl, ip)
	}
	return repl, nil
}

func (p *PostgreSQLClient) PGHbaFilePath() (string, error) {
	var path string
	sql := "show hba_file;"
	err := p.conn.QueryRow(context.Background(), sql).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("获取 pg_hba.conf 文件位置失败: %v", err)
	}
	return path, nil
}

func (p *PostgreSQLClient) PGFilePath() (string, error) {
	var path string
	sql := "show config_file;"
	err := p.conn.QueryRow(context.Background(), sql).Scan(&path)
	if err != nil {
		return "", fmt.Errorf("获取 postgresql.conf 文件位置失败: %v", err)
	}
	return path, nil
}
