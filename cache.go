package daoongorm

import "context"

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
