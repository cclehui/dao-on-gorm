package daoongorm

import (
	"context"
	"strings"
)

type CacheInterface interface {
	Set(ctx context.Context, key string, value interface{}, ttl int) (err error)
	Get(ctx context.Context, key string, value interface{}) (hit bool, err error)
	Del(ctx context.Context, key string) (err error)
}

var globalCacheUtil CacheInterface = &NopCacheUtil{}

// 设置全局缓存操作util
func SetGlobalCacheUtil(cacheUtil CacheInterface) {
	globalCacheUtil = cacheUtil
}

// SetDaoCachePackagePrefix 设置缓存 key 的包级前缀, 空串表示不启用.
//
// 只影响 cache key, 不影响表名/主键, 也不影响 GetUniqKey.
// 例如设置成 "identity_hub" 后 key 从
//
//	daocache:db:user_groups:1
//
// 变成
//
//	identity_hub:daocache:db:user_groups:1
func SetDaoCachePackagePrefix(prefix string) {
	DaoCachePackagePrefix = strings.TrimSpace(prefix)
}

type NopCacheUtil struct{}

func (ncu *NopCacheUtil) Set(ctx context.Context,
	key string, value interface{}, ttl int) (err error) {
	return nil
}

func (ncu *NopCacheUtil) Get(ctx context.Context, key string, value interface{}) (hit bool, err error) {
	return false, nil
}

func (ncu *NopCacheUtil) Del(ctx context.Context, key string) (err error) {
	return nil
}
