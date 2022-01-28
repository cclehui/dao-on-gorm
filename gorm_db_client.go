package daoongorm

import (
	"context"

	"gorm.io/gorm"
)

type GormDBClient interface {
	Table(ctx context.Context, name string) *gorm.DB         // 读写连接
	ReadOnlyTable(ctx context.Context, name string) *gorm.DB // 只读连接
}
