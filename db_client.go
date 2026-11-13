package daoongorm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	loggergorm "gorm.io/gorm/logger"
)

// afterCommitRegistry 事务对象可选实现: 在 COMMIT 成功之后执行缓存失效.
//
// 用可选接口而不是扩大 DBClientInterface, 外部自定义实现不受影响;
// 未实现时退化为"事务内不写缓存", 只是缓存会多存活一个 TTL.
type afterCommitRegistry interface {
	AddAfterCommit(fn func(ctx context.Context))
}

// DBClientInterface 的一个实现

type DBClient struct {
	*gorm.DB                 // 读写连接
	read     []*gorm.DB      // 只读连接
	idx      int64           // 只读连接索引号
	conf     *DBClientConfig // 配置信息

	afterCommitMu sync.Mutex
	afterCommit   []func(ctx context.Context)
}

// 连接数据库， 获取新链接
func NewDBClient(c *DBClientConfig) (*DBClient, error) {
	if c == nil || c.DSN == nil {
		return nil, errors.New("DBClientConfig.DSN 不能为空")
	}

	ormDB := new(DBClient)
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
		if rd == nil {
			continue
		}

		readDB, err := connectGORM(c, rd)
		if err != nil {
			return nil, err
		}

		rs = append(rs, readDB)
	}

	ormDB.read = rs

	return ormDB, nil
}

// Table 设定表名
func (db *DBClient) Table(ctx context.Context, name string) *gorm.DB {
	return db.DataSource(ctx, false).Table(name)
}

// ReadOnlyTable 设定只读表名
func (db *DBClient) ReadOnlyTable(ctx context.Context, name string) *gorm.DB {
	return db.DataSource(ctx, true).Table(name)
}

// DataSource 设定数据源.
//
// 必须用 WithContext: 之前用 Set("scope_context", ctx) 只挂了个自定义 key,
// 请求取消、deadline 和链路追踪都传不到 gorm/SQL 层.
func (db *DBClient) DataSource(ctx context.Context, isReadOnly bool) *gorm.DB {
	if ctx == nil {
		ctx = context.Background()
	}

	return db.getCurrentDB(isReadOnly).WithContext(ctx)
}

// 获取当前连接. 事务 clone 的 read 就是事务本身, 所以 ReadOnly 在事务里也会落到 tx 上.
func (db *DBClient) getCurrentDB(isReadOnly bool) *gorm.DB {
	if isReadOnly {
		return db.ReadOnly()
	}

	return db.DB
}

// ReadOnly 获取只读连接
func (db *DBClient) ReadOnly() *gorm.DB {
	idx := db.readIndex()

	for i := range db.read {
		if rd := db.read[(idx+i)%len(db.read)]; rd != nil {
			return rd
		}
	}

	return db.DB
}

// 获取只读索引
func (db *DBClient) readIndex() int {
	if len(db.read) == 0 {
		return 0
	}

	v := atomic.AddInt64(&db.idx, 1) % int64(len(db.read))

	return int(v)
}

// AddAfterCommit 注册一个"提交成功后执行"的回调. 事务回滚时不会执行.
func (db *DBClient) AddAfterCommit(fn func(ctx context.Context)) {
	if fn == nil {
		return
	}

	db.afterCommitMu.Lock()
	defer db.afterCommitMu.Unlock()

	db.afterCommit = append(db.afterCommit, fn)
}

func (db *DBClient) runAfterCommit(ctx context.Context) {
	db.afterCommitMu.Lock()
	callbacks := db.afterCommit
	db.afterCommit = nil
	db.afterCommitMu.Unlock()

	for _, fn := range callbacks {
		fn(ctx)
	}
}

// 事务函数
type TransactionFunction func(ctx context.Context, tx *DBClient) error

// Transaction 在事务里执行 fn, 出错或 panic 回滚.
//
// 事务内的 DAO 构造请传 OptionSetTX(tx). 提交成功后统一失效缓存:
// 事务里提前删缓存会让并发读把未提交的旧值重新灌回去.
func (db *DBClient) Transaction(ctx context.Context, tansFunc TransactionFunction) (err error) {
	transactionCtx, cancel := context.WithTimeout(ctx, time.Duration(db.conf.TranTimeout))
	defer cancel()

	// 用 WithContext(ctx) 开启事务: 之前直接 Begin() 会让事务语句丢失 ctx,
	// 超时/取消时只能等数据库自己断连.
	tx := db.DB.WithContext(transactionCtx).Begin()
	if tx.Error != nil {
		return tx.Error
	}

	txDB := db.clone(tx, []*gorm.DB{tx})

	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}

		if err != nil {
			tx.Rollback()

			return
		}

		if commitErr := tx.Commit().Error; commitErr != nil {
			err = commitErr

			return
		}

		// 只有真正提交成功才让缓存失效; 回滚时旧缓存仍然是对的.
		txDB.runAfterCommit(ctx)
	}()

	err = tansFunc(transactionCtx, txDB)

	return err
}

func (db *DBClient) clone(write *gorm.DB, read []*gorm.DB) *DBClient {
	return &DBClient{
		DB:   write,
		read: read,
		idx:  0,
		conf: db.conf,
	}
}

func (db *DBClient) GetDBClientConfig() *DBClientConfig {
	return db.conf
}

func concatConnectURI(dbConfig *DBClientConfig, dsnConfig *DSNConfig) string {
	options := append([]string{}, dsnConfig.Options...)
	options = appendConnectTimeouts(options, dbConfig)

	uri := fmt.Sprintf("%s:%s@(%s:%d)/%s",
		dsnConfig.UserName, dsnConfig.Password,
		dsnConfig.Endpoint.Address, dsnConfig.Endpoint.Port,
		dsnConfig.DBName)
	if len(options) != 0 {
		uri = fmt.Sprintf("%s?%s", uri, strings.Join(options, "&"))
	}

	return uri
}

// appendConnectTimeouts 把 queryTimeout / execTimeout 落到 driver 层.
//
// go-sql-driver 的 readTimeout/writeTimeout 是连接级超时, 到点直接断连接,
// 不会像 context.WithTimeout 那样需要有人调用 cancel. 调用方显式写在 options
// 里的同名参数优先, 这里不覆盖.
func appendConnectTimeouts(options []string, dbConfig *DBClientConfig) []string {
	has := func(key string) bool {
		for _, option := range options {
			if strings.HasPrefix(option, key+"=") {
				return true
			}
		}

		return false
	}

	if dbConfig.QueryTimeout > 0 && !has("readTimeout") {
		options = append(options, fmt.Sprintf("readTimeout=%s", time.Duration(dbConfig.QueryTimeout)))
	}

	if dbConfig.ExecTimeout > 0 && !has("writeTimeout") {
		options = append(options, fmt.Sprintf("writeTimeout=%s", time.Duration(dbConfig.ExecTimeout)))
	}

	return options
}

// resolveLogLevel 把配置里的字符串映射成 gorm 日志级别, 未知值退回 warn.
func resolveLogLevel(raw string) loggergorm.LogLevel {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "silent":
		return loggergorm.Silent
	case "error":
		return loggergorm.Error
	case "info":
		return loggergorm.Info
	default:
		return loggergorm.Warn
	}
}

// 建立连接
func connectGORM(c *DBClientConfig, dsnConfig *DSNConfig) (*gorm.DB, error) {
	d, err := gorm.Open(mysql.Open(concatConnectURI(c, dsnConfig)), &gorm.Config{
		Logger: loggergorm.Default.LogMode(resolveLogLevel(c.LogLevel)),
	})
	if err != nil {
		return nil, errors.WithStack(err)
	}

	rawDB, err := d.DB()
	if err != nil {
		return nil, errors.WithStack(err)
	}

	if c.Active > 0 {
		rawDB.SetMaxOpenConns(c.Active)
	}

	if c.Idle > 0 {
		rawDB.SetMaxIdleConns(c.Idle)
	}

	if c.IdleTimeout > 0 {
		rawDB.SetConnMaxLifetime(time.Duration(c.IdleTimeout))
	}

	return d, nil
}
