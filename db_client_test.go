package daoongorm

import (
	"context"
	"strings"
	"testing"
	"time"

	loggergorm "gorm.io/gorm/logger"

	"github.com/cclehui/dao-on-gorm/internal/ctime"
)

// 只构造客户端结构体, 不连库, 用于验证纯函数和超时配置.
func newTestDBClientWithNilRead(conf *DBClientConfig) *DBClient {
	return &DBClient{conf: conf}
}

func contains(raw, sub string) bool {
	return strings.Contains(raw, sub)
}

func TestResolveLogLevel(t *testing.T) {
	cases := map[string]loggergorm.LogLevel{
		"silent": loggergorm.Silent,
		"error":  loggergorm.Error,
		"info":   loggergorm.Info,
		"warn":   loggergorm.Warn,
		"":       loggergorm.Warn,
		"weird":  loggergorm.Warn,
	}

	for raw, want := range cases {
		if got := resolveLogLevel(raw); got != want {
			t.Errorf("resolveLogLevel(%q) = %v, want %v", raw, got, want)
		}
	}
}

// queryTimeout/execTimeout 之前只在配置里声明, 从没落到 driver; 这里保证拼进 DSN.
func TestConcatConnectURIFallsBackToDriverTimeouts(t *testing.T) {
	conf := &DBClientConfig{
		QueryTimeout: ctime.Duration(3 * time.Second),
		ExecTimeout:  ctime.Duration(7 * time.Second),
	}
	dsn := &DSNConfig{
		UserName: "u",
		Password: "p",
		Endpoint: &EndpointConfig{Address: "127.0.0.1", Port: 3306},
		DBName:   "db",
		Options:  []string{"charset=utf8mb4"},
	}

	uri := concatConnectURI(conf, dsn)

	for _, want := range []string{"readTimeout=3s", "writeTimeout=7s", "charset=utf8mb4"} {
		if !contains(uri, want) {
			t.Errorf("uri %q missing %q", uri, want)
		}
	}
}

// 调用方显式配置的 readTimeout 优先, 不能被配置项覆盖.
func TestConcatConnectURIPrefersExplicitTimeout(t *testing.T) {
	conf := &DBClientConfig{QueryTimeout: ctime.Duration(3 * time.Second)}
	dsn := &DSNConfig{
		Endpoint: &EndpointConfig{Address: "127.0.0.1", Port: 3306},
		Options:  []string{"readTimeout=30s"},
	}

	uri := concatConnectURI(conf, dsn)

	if contains(uri, "readTimeout=3s") {
		t.Errorf("explicit readTimeout should win, uri = %q", uri)
	}
}

func TestNewDBClientRejectsEmptyConfig(t *testing.T) {
	if _, err := NewDBClient(nil); err == nil {
		t.Fatal("nil config should fail")
	}

	if _, err := NewDBClient(&DBClientConfig{}); err == nil {
		t.Fatal("missing DSN should fail")
	}
}

func TestReadOnlyFallsBackToWriteConn(t *testing.T) {
	conf := &DBClientConfig{TranTimeout: ctime.Duration(time.Second)}
	db := newTestDBClientWithNilRead(conf)

	if got := db.ReadOnly(); got != nil {
		t.Fatalf("ReadOnly without replicas should return the write conn (nil here), got %v", got)
	}
}

func TestAfterCommitRunsOnceAndInOrder(t *testing.T) {
	db := &DBClient{conf: &DBClientConfig{}}

	var order []string
	db.AddAfterCommit(func(context.Context) { order = append(order, "first") })
	db.AddAfterCommit(func(context.Context) { order = append(order, "second") })

	db.runAfterCommit(context.Background())

	if len(order) != 2 || order[0] != "first" || order[1] != "second" {
		t.Fatalf("order = %v", order)
	}

	// 已经执行过的回调不能被第二个事务重复触发.
	db.runAfterCommit(context.Background())

	if len(order) != 2 {
		t.Fatalf("callbacks ran more than once: %v", order)
	}
}

func TestAddAfterCommitIgnoresNil(t *testing.T) {
	db := &DBClient{conf: &DBClientConfig{}}

	db.AddAfterCommit(nil)
	db.runAfterCommit(context.Background())

	if len(db.afterCommit) != 0 {
		t.Fatal("nil callback should be ignored")
	}
}
