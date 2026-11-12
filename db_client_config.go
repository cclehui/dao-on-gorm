package daoongorm

import (
	"github.com/cclehui/dao-on-gorm/internal/ctime"
)

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
	// 查询超时时间: 落到 driver 的 readTimeout
	QueryTimeout ctime.Duration `yaml:"queryTimeout"`
	// 执行超时时间: 落到 driver 的 writeTimeout
	ExecTimeout ctime.Duration `yaml:"execTimeout"`
	// 事务超时时间
	TranTimeout ctime.Duration `yaml:"tranTimeout"`
	// gorm 日志级别: silent / error / warn / info, 空值按 warn
	LogLevel string `yaml:"logLevel"`
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
