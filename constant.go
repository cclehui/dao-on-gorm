package daoongorm

import "time"

// 全局属性变量 可以配置成自己需要的值
var (
	DaoCacheExpire int    = 86400 * 7  // dao缓存超时时间
	DaoCachePrefix string = "daocache" // dao缓存key前缀

	FieldNameCreatedAt = "CreatedAt"
	FieldNameUpdatedAt = "UpdatedAt"
)

const (
	TypeNameTime = "Time"
)

type DBIDInt struct {
	ID int `gorm:"column:id" structs:"id" json:"id"`
}

type DBIDInt64 struct {
	ID int64 `gorm:"column:id" structs:"id" json:"id"`
}

type DBUpdateAt struct {
	UpdatedAt time.Time `gorm:"column:updated_at" structs:"updated_at" json:"updated_at"`
}
