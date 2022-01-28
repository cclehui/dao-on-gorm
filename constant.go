package daoongorm

import "time"

const (
	DaoCachePrefix = "daocache" // dao缓存key前缀
	DaoCacheExpire = 86400 * 7  // dao缓存超时时间

	TypeNameTime = "Time"

	FieldNameCreateAt  = "CreatedAt"
	FieldNameUpdatedAt = "UpdatedAt"
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
