package daoongorm

import "time"

// 全局属性变量 可以配置成自己需要的值
var (
	DaoCacheExpire int    = 86400 * 7  // dao缓存超时时间
	DaoCachePrefix string = "daocache" // dao缓存key前缀

	// DaoCachePackagePrefix 业务包/服务级前缀, 默认空.
	//
	// 多个服务共用同一个 redis 实例 (或同一个 key 前缀) 时, 用它给各自的缓存 key 加上命名空间,
	// 避免不同服务同名表撞 key. 空串表示不启用, key 结构和老版本完全一致.
	DaoCachePackagePrefix string = ""

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
