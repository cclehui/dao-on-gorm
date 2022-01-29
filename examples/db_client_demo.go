package dao

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/cclehui/dao-on-gorm/internal/ctime"
	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// db_client_demo
// 使用时自己根据自己的框架情况去封装，只要实现 GormDBClient interface{} 即可

type DBClientDemo struct {
	// 主连接
	*gorm.DB
	// 只读连接
	read []*gorm.DB
	// 只读连接索引号
	idx int64
	// 配置文件
	conf *DBClientConfig
}

// 连接数据库， 获取新链接
func NewDBClientDemo(c *DBClientConfig) (*DBClientDemo, error) {
	ormDB := new(DBClientDemo)
	ormDB.conf = c

	d, err := connectGORM(c, c.DSN)
	if err != nil {
		return nil, err
	}
	ormDB.DB = d

	if len(c.ReadDSN) == 0 {
		c.ReadDSN = []*DSNConfig{c.DSN}
	}
	rs := make([]*gorm.DB, 0, len(c.ReadDSN))
	for _, rd := range c.ReadDSN {
		d, err := connectGORM(c, rd)
		if err != nil {
			return nil, err
		}
		rs = append(rs, d)

		ormDB.read = rs

	}
	return ormDB, nil
}

// Table 设定表名
func (db *DBClientDemo) Table(ctx context.Context, name string) *gorm.DB {
	return db.DataSource(ctx, false).Table(name)
}

// ReadOnlyTable 设定只读表名
func (db *DBClientDemo) ReadOnlyTable(ctx context.Context, name string) *gorm.DB {
	return db.DataSource(ctx, true).Table(name)
}

// DataSource 设定数据源
func (db *DBClientDemo) DataSource(ctx context.Context, isReadOnly bool) *gorm.DB {
	return db.getCurrentDB(isReadOnly).Set("scope_context", ctx)
}

// 获取当前连接
func (db *DBClientDemo) getCurrentDB(isReadOnly bool) *gorm.DB {
	if isReadOnly {
		return db.ReadOnly()
	}
	return db.DB
}

// ReadOnly 获取只读连接
func (db *DBClientDemo) ReadOnly() *gorm.DB {
	idx := db.readIndex()
	for i := range db.read {
		if rd := db.read[(idx+i)%len(db.read)]; rd != nil {
			return rd
		}
	}
	return db.DB
}

// 获取只读索引
func (db *DBClientDemo) readIndex() int {
	if len(db.read) == 0 {
		return 0
	}
	v := atomic.AddInt64(&db.idx, 1) % int64(len(db.read))
	atomic.StoreInt64(&db.idx, v)
	return int(v)
}

// 事务函数
type TransactionFunction func(ctx context.Context, tx *DBClientDemo) error

func (db *DBClientDemo) Transaction(ctx context.Context, tansFunc TransactionFunction) (err error) {
	transactionCtx, cancel := context.WithTimeout(ctx, time.Duration(db.conf.TranTimeout))
	defer cancel()

	tx := db.Begin()
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit().Error
		}
	}()

	txDB := db.clone(tx, []*gorm.DB{tx})

	err = tansFunc(transactionCtx, txDB)

	return err
}

func (db *DBClientDemo) clone(write *gorm.DB, read []*gorm.DB) *DBClientDemo {
	return &DBClientDemo{
		DB:   write,
		read: read,
		idx:  0,
		conf: db.conf,
	}
}

func (db *DBClientDemo) GetDBClientConfig() *DBClientConfig {
	return db.conf
}

type EndpointConfig struct {
	Address string `yaml:"address"`
	Port    int    `yaml:"port"`
}

// DSN配置
type DSNConfig struct {
	UserName string          `yaml:"userName"`
	Password string          `yaml:"password"`
	Endpoint *EndpointConfig `yaml:"endpoint"`
	DBName   string          `yaml:"dbName"`
	Options  []string        `yaml:"options"`
}

type DBClientConfig struct {
	// 主dsn配置
	DSN *DSNConfig `yaml:"dsn"`
	// 只读dsn配置
	ReadDSN []*DSNConfig `yaml:"readDSN"`
	// 最大可用数量
	Active int `yaml:"active"`
	// 最大闲置数量
	Idle int `yaml:"idle"`
	// 闲置超时时间
	IdleTimeout ctime.Duration `yaml:"idleTimeout"`
	// 查询超时时间
	QueryTimeout ctime.Duration `yaml:"queryTimeout"`
	// 执行超时时间
	ExecTimeout ctime.Duration `yaml:"execTimeout"`
	// 事务超时时间
	TranTimeout ctime.Duration `yaml:"tranTimeout"`
}

func concatConnectURI(dsnConfig *DSNConfig) string {
	uri := fmt.Sprintf("%s:%s@(%s:%d)/%s",
		dsnConfig.UserName, dsnConfig.Password,
		dsnConfig.Endpoint.Address, dsnConfig.Endpoint.Port,
		dsnConfig.DBName)
	if len(dsnConfig.Options) != 0 {
		uri = fmt.Sprintf("%s?%s", uri, strings.Join(dsnConfig.Options, "&"))
	}

	return uri
}

// 建立连接
func connectGORM(c *DBClientConfig, dsnConfig *DSNConfig) (*gorm.DB, error) {
	d, err := gorm.Open(mysql.Open(concatConnectURI(dsnConfig)), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	rawDB, err := d.DB()
	if err != nil {
		err = errors.WithStack(err)
		return nil, err
	}
	rawDB.SetMaxOpenConns(c.Active)
	rawDB.SetMaxIdleConns(c.Idle)
	rawDB.SetConnMaxLifetime(time.Duration(c.IdleTimeout))

	return d, nil
}
