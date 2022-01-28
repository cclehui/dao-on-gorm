package daoongorm

import "context"

type CacheInterface interface {
	SetCache(ctx context.Context, key string, value interface{}, ttl int) (err error)
	GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error)
	DeleteCache(ctx context.Context, key string) (err error)
}

var cacheUtil CacheInterface = &nopCacheUtil{}

func SetCacheUtil(cacheUtil CacheInterface) {
	cacheUtil = cacheUtil
}

type nopCacheUtil struct{}

func (ncu *nopCacheUtil) SetCache(ctx context.Context,
	key string, value interface{}, ttl int) (err error) {
	return nil
}

func (ncu *nopCacheUtil) GetCache(ctx context.Context, key string, value interface{}) (hit bool, err error) {
	return false, nil
}

func (ncu *nopCacheUtil) DeleteCache(ctx context.Context, key string) (err error) {
	return nil
}
