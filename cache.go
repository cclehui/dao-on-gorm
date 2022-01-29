package daoongorm

import "context"

type CacheInterface interface {
	SetCache(ctx context.Context, key string, value interface{}, ttl int) (err error)
	GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error)
	DeleteCache(ctx context.Context, key string) (err error)
}

var globalCacheUtil CacheInterface = &NopCacheUtil{}

// 设置全局缓存操作util
func SetGlobalCacheUtil(cacheUtil CacheInterface) {
	globalCacheUtil = cacheUtil
}

type NopCacheUtil struct{}

func (ncu *NopCacheUtil) SetCache(ctx context.Context,
	key string, value interface{}, ttl int) (err error) {
	return nil
}

func (ncu *NopCacheUtil) GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error) {
	return false, nil
}

func (ncu *NopCacheUtil) DeleteCache(ctx context.Context, key string) (err error) {
	return nil
}
