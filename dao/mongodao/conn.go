/*
 * @Author: lsne
 * @Date: 2025-12-07 15:30:18
 */

package mongodao

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/v2/bson"
	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"
)

type Collection struct {
	Id           string             `json:"id" bson:"_id"`
	LastmodEpoch primitive.ObjectID `json:"lastmod_epoch" bson:"lastmodEpoch"`
	Lastmod      primitive.DateTime `json:"lastmod" bson:"lastmod"`
	Dropped      bool               `json:"dropped" bson:"dropped"`
	Key          bson.D             `json:"key" bson:"key"`
	Unique       bool               `json:"unique" bson:"unique"`
	//Uuid         string              `json:"uuid" bson:"uuid"`
}

type MongoClient struct {
	Hosts      []string
	Port       int
	User       string
	Pass       string
	Replicaset string
	Type       string
	Database   string
	URL        string
	Conn       *mongo.Client
}

func NewMongoClient(host []string, port int, t string) *MongoClient {
	return &MongoClient{
		Hosts: host,
		Port:  port,
		Type:  t,
	}
}

// 初始化配置
func (m *MongoClient) InitConfig() {
	if m.User == "" {
		m.User = "admin"
	}
	if m.Pass == "" {
		m.Pass = "xxx"
	}
	if m.Database == "" {
		m.Database = "admin"
	}
	if m.Type == "rs" && m.Replicaset == "" {
		m.Replicaset = "md" + strconv.Itoa(m.Port)
	}
}

// Connect 创建新连接
func (m *MongoClient) Connect() error {
	sep := ":" + strconv.Itoa(m.Port)
	hostString := strings.Join(m.Hosts, sep+",")
	hostString = hostString + sep

	switch m.Type {
	case "rs":
		m.URL = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=%s&replicaSet=%s&connectTimeoutMS=3000&socketTimeoutMS=3000&serverSelectionTimeoutMS=3000", m.User, m.Pass, hostString, m.Database, m.Replicaset)
	case "sc":
		m.URL = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=%s&connectTimeoutMS=3000&socketTimeoutMS=3000&serverSelectionTimeoutMS=3000", m.User, m.Pass, hostString, m.Database)
	case "direct":
		m.URL = fmt.Sprintf("mongodb://%s:%s@%s/?authSource=%s&connectTimeoutMS=3000&socketTimeoutMS=3000&serverSelectionTimeoutMS=3000&connect=direct", m.User, m.Pass, hostString, m.Database)
	default:
		return errors.New("未知的集群类型")
	}

	//connOptions := options.Client().ApplyURI(m.URL)
	//connOptions = connOptions.SetConnectTimeout(3 * time.Second)
	//connOptions = connOptions.SetSocketTimeout(3 * time.Second)
	//connOptions = connOptions.SetServerSelectionTimeout(3 * time.Second)

	// ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	// defer cancel()
	// conn, err := mongo.Connect(ctx, options.Client().ApplyURI(m.URL).SetReadPreference(readpref.Primary()))
	conn, err := mongo.Connect(options.Client().ApplyURI(m.URL).SetReadPreference(readpref.Primary()))
	if err != nil {
		return err
	}

	m.Conn = conn
	return nil
}

// 运行command命令
func (m *MongoClient) RunCommand(dbname string, cmd bson.D) (result bson.M, err error) {
	opts := options.RunCmd().SetReadPreference(readpref.Primary())
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	if err := m.Conn.Database(dbname).RunCommand(ctx, cmd, opts).Decode(&result); err != nil {
		return result, err
	}
	return result, nil
}

// 获取数据库列表
func (m *MongoClient) GetDBList() (dbList []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if dbList, err = m.Conn.ListDatabaseNames(ctx, bson.M{}); err != nil {
		return dbList, fmt.Errorf("获取数据列表库失败: %w", err)
	}
	return dbList, nil
}

// 获取数据库列表 dbsize
func (m *MongoClient) GetDBListResult() (ldr mongo.ListDatabasesResult, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if ldr, err = m.Conn.ListDatabases(ctx, bson.M{}); err != nil {
		return ldr, fmt.Errorf("获取数据列表库失败: %w", err)
	}
	return ldr, nil
}

// 判断数据库是否存在
func (m *MongoClient) DBExist(dbname string) (exist bool, err error) {
	dbs, err := m.GetDBList()
	if err != nil {
		return false, err
	}
	for _, db := range dbs {
		if db == dbname {
			return true, nil
		}
	}
	return false, nil
}

// 获取集合
func (m *MongoClient) GetCollectionList(dbname string) (colList []string, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if colList, err = m.Conn.Database(dbname).ListCollectionNames(ctx, bson.M{}); err != nil {
		return colList, fmt.Errorf("从库: %s 中获取集合列表失败", dbname)
	}
	return colList, nil
}

// 判断数据库是否存在
func (m *MongoClient) ColExist(dbname, colName string) (exist bool, err error) {
	cols, err := m.GetCollectionList(dbname)
	if err != nil {
		return false, err
	}
	for _, col := range cols {
		if col == colName {
			return true, nil
		}
	}
	return false, nil
}

// 获取角色
func (m *MongoClient) GetRole(roleName, dbname string) (roles bson.A, err error) {
	var result bson.M
	cmd := bson.D{{Key: "rolesInfo", Value: bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: dbname}}}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return roles, fmt.Errorf("获取用户:%s, 在库:%s 里的角色失败", roleName, dbname)
	}
	if _, ok := result["roles"]; ok {
		roles = result["roles"].(bson.A)
	}
	return roles, nil
}

// 判断角色是否存在
func (m *MongoClient) RoleExist(roleName, dbname string) (exist bool, err error) {
	roles, err := m.GetRole(roleName, dbname)
	if err != nil {
		return false, err
	}
	return len(roles) != 0, nil
}

// 获取用户
func (m *MongoClient) GetUser(username, dbname string) (users bson.A, err error) {
	var result bson.M
	cmd := bson.D{{Key: "usersInfo", Value: bson.D{{Key: "user", Value: username}, {Key: "db", Value: dbname}}}}
	if result, err = m.RunCommand("admin", cmd); err != nil {
		return users, fmt.Errorf("获取用户:%s, 在库:%s 里的用户失败", username, dbname)
	}
	if _, ok := result["users"]; ok {
		users = result["users"].(bson.A)
	}
	return users, nil
}

// 判断用户是否存在
func (m *MongoClient) UserExist(username, dbname string) (exist bool, err error) {
	users, err := m.GetUser(username, dbname)
	if err != nil {
		return false, err
	}
	return len(users) != 0, nil
}

// 创建数据库
func (m *MongoClient) CreateDB(dbname string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if _, err := m.Conn.Database(dbname).Collection("test").InsertOne(ctx, bson.M{"dba": "init"}); err != nil {
		return fmt.Errorf("创建库: %s 失败: %w", dbname, err)
	}
	return nil
}

// 库级别启用分片功能
func (m *MongoClient) EnableSharding(dbname string) error {
	cmd := bson.D{{Key: "enableSharding", Value: dbname}}
	if _, err := m.RunCommand("admin", cmd); err != nil {
		return err
	}
	return nil
}

// 创建角色, 主要是给分片的每个集合增加权限
func (m *MongoClient) CreateRole(roleName, dbname, colName string) error {
	if roleName == "" || dbname == "" || colName == "" {
		return errors.New("角色名,库名,集合名都不能为空")
	}

	// 拼接创建角色的命令, 比较复杂, 去掉了 dropCollection 的权限, 防止业务创建集合后, 手动又删除集合然后写数据, 导致不分片数据不均衡
	actions := []string{"listCollections", "createCollection", "convertToCapped", "killCursors", "collStats", "find", "insert", "remove", "update", "listIndexes", "createIndex", "dropIndex", "dbStats", "renameCollectionSameDB", "dbHash"}
	resource := bson.D{{Key: "db", Value: dbname}, {Key: "collection", Value: colName}}
	privilege := bson.D{{Key: "resource", Value: resource}, {Key: "actions", Value: actions}}
	privileges := bson.A{privilege}
	cmd := bson.D{{Key: "createRole", Value: roleName}, {Key: "privileges", Value: privileges}, {Key: "roles", Value: bson.A{}}}

	if _, err := m.RunCommand(dbname, cmd); err != nil {
		return fmt.Errorf("创建授权角色 %s.%s 失败", dbname, roleName)
	}

	return nil
}

// 删除角色, 针对的分片集群的每个集合的角色
func (m *MongoClient) DropRole(roleName, dbname string) error {
	if roleName == "" || dbname == "" {
		return errors.New("角色名,库名都不能为空")
	}

	cmd := bson.D{{Key: "dropRole", Value: roleName}}

	if _, err := m.RunCommand(dbname, cmd); err != nil {
		return fmt.Errorf("删除角色 %s.%s 失败", dbname, roleName)
	}

	return nil
}

// 创建用户
func (m *MongoClient) CreateUser(username, password, dbname string) error {
	if username == "" || dbname == "" || password == "" {
		return errors.New("用户名,密码,库名都不能为空")
	}

	role1 := bson.D{{Key: "role", Value: "readWriteAnyDatabase"}, {Key: "db", Value: dbname}}
	role2 := bson.D{{Key: "role", Value: "dbAdminAnyDatabase"}, {Key: "db", Value: dbname}}
	role3 := bson.D{{Key: "role", Value: "clusterManager"}, {Key: "db", Value: dbname}}
	role4 := bson.D{{Key: "role", Value: "clusterMonitor"}, {Key: "db", Value: dbname}}
	roles := bson.A{role1, role2, role3, role4}
	cmd := bson.D{{Key: "createUser", Value: username}, {Key: "pwd", Value: password}, {Key: "roles", Value: roles}}

	if _, err := m.RunCommand(dbname, cmd); err != nil {
		return fmt.Errorf("为db: %s 创建用户: %s 失败", dbname, username)
	}
	return nil
}

// 给用户新增角色
func (m *MongoClient) GrantRolesToUser(username, roleName, dbname string) error {
	if username == "" || dbname == "" || roleName == "" {
		return errors.New("用户名,库名,角色名都不能为空")
	}

	role := bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: dbname}}
	roles := bson.A{role}
	cmd := bson.D{{Key: "grantRolesToUser", Value: username}, {Key: "roles", Value: roles}}

	if _, err := m.RunCommand(dbname, cmd); err != nil {
		return fmt.Errorf("为db: %s 库中的用户: %s 新增角色: %s 失败", dbname, username, roleName)
	}
	return nil
}

// 给用户回收角色
func (m *MongoClient) RevokeRolesFromUser(username, roleName, dbname string) error {
	if username == "" || dbname == "" || roleName == "" {
		return errors.New("用户名,库名,角色名都不能为空")
	}

	role := bson.D{{Key: "role", Value: roleName}, {Key: "db", Value: dbname}}
	roles := bson.A{role}
	cmd := bson.D{{Key: "revokeRolesFromUser", Value: username}, {Key: "roles", Value: roles}}

	if _, err := m.RunCommand(dbname, cmd); err != nil {
		return fmt.Errorf("为db: %s 库中的用户: %s 回收角色: %s 失败", dbname, username, roleName)
	}
	return nil
}

// 给集合创建片键
func (m *MongoClient) ShardCollection(dbname, colName string, keyType string, keyString []string, unique bool) error {
	if dbname == "" || colName == "" || len(keyString) == 0 {
		return errors.New("库名,集合名, 片键都不能为空")
	}

	// 解析keys为mongo识别的有序结构
	var keys bson.D
	switch keyType {
	case "hashed":
		if len(keyString) != 1 {
			return errors.New("hashed分片不支持复合索引")
		}
		if unique {
			return errors.New("hashed分片不支持唯一属性")
		}
		keys = append(keys, bson.E{Key: keyString[0], Value: "hashed"})
	case "ranged":
		for _, key := range keyString {
			keys = append(keys, bson.E{Key: key, Value: 1})
		}
	default:
		return errors.New("不支持的片键类型")
	}

	// 前端传 "field_hash" 这种格式的key时用, 如果将来升级为mongodb 4.4 支持 hashed 和 ranged 混合片键的时候, 可能会用到
	//for _, key := range keyString {
	//	var field bson.E
	//	kv := strings.Split(key, "_")
	//	if kv[1] == "hashed" {
	//		field = bson.E{Key: kv[0], Value: kv[1]}
	//	} else {
	//		v, e  := strconv.Atoi(kv[1])
	//		if e != nil {
	//			logger.Ins().Error(e)
	//			return errors.New("转换keys排序为数值失败")
	//		}
	//		field = bson.E{Key: kv[0], Value: v}
	//	}
	//
	//	keys = append(keys, field)
	//}

	cmd := bson.D{{Key: "shardCollection", Value: dbname + "." + colName}, {Key: "key", Value: keys}, {Key: "unique", Value: unique}}

	if _, err := m.RunCommand("admin", cmd); err != nil {
		return fmt.Errorf("为db: %s 库中的集合: %s 设置片键失败", dbname, colName)
	}
	return nil
}

// 删除集合
func (m *MongoClient) DropCollection(dbname, colName string) error {
	if dbname == "" || colName == "" {
		return errors.New("库名,集合名都不能为空")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.Conn.Database(dbname).Collection(colName).Drop(ctx); err != nil {
		return fmt.Errorf("删除集合 %s.%s 失败: %w", dbname, colName, err)
	}
	return nil
}

func (m *MongoClient) GetShardCollectionList() (cols []*Collection, err error) {
	f := map[string]interface{}{
		"dropped": false,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cur, err := m.Conn.Database("config").Collection("collections").Find(ctx, f)
	if err != nil {
		return nil, err
	}

	for cur.Next(ctx) {
		var elem Collection
		//var elem = make(map[string]interface{})
		if err = cur.Decode(&elem); err != nil {
			return nil, err
		}
		cols = append(cols, &elem)
	}

	return cols, nil
}

func (m *MongoClient) ConfigModify(key string, value interface{}) error {
	var conf string
	switch key {
	case "cache_size":
		conf = fmt.Sprintf("cache_size=%dG", int(value.(float64)))
	default:
		return fmt.Errorf("参数: %s 不正确, 或暂不支持该参数的修改", key)
	}

	cmd := bson.D{{Key: "setParameter", Value: 1}, {Key: "wiredTigerEngineRuntimeConfig", Value: conf}}
	if _, err := m.RunCommand("admin", cmd); err != nil {
		return err
	}
	return nil
}
